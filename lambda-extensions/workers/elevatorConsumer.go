package workers

import (
	"context"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"
	sumocli "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/sumoclient"

	"github.com/sirupsen/logrus"
)

// ElevatorTaskConsumer exposes methods for consuming tasks in elevator mode
type ElevatorTaskConsumer interface {
	Start(context.Context)
	FlushDataQueue(context.Context)
	DrainQueue(context.Context) int
}

// elevatorSumoConsumer drains log from dataQueue in elevator mode
type elevatorSumoConsumer struct {
	dataQueue   chan []byte
	flushSignal chan string
	logger      *logrus.Entry
	config      *cfg.LambdaExtensionConfig
	sumoclient  sumocli.LogSender
}

// NewElevatorTaskConsumer returns a new elevator consumer
// flushSignal channel is used to receive signals from producer to trigger flushing
func NewElevatorTaskConsumer(consumerQueue chan []byte, flushSignal chan string, config *cfg.LambdaExtensionConfig, logger *logrus.Entry) ElevatorTaskConsumer {
	return &elevatorSumoConsumer{
		dataQueue:   consumerQueue,
		flushSignal: flushSignal,
		logger:      logger,
		sumoclient:  sumocli.NewLogSenderClient(logger, config),
		config:      config,
	}
}

// Start starts the elevator consumer in a goroutine to listen for flush signals independently
func (esc *elevatorSumoConsumer) Start(ctx context.Context) {
	esc.logger.Info("Starting Elevator Consumer")
	go esc.processFlushSignals(ctx)
}

// processFlushSignals continuously listens for flush signals and triggers queue draining
// This runs independently without needing callbacks from main thread
func (esc *elevatorSumoConsumer) processFlushSignals(ctx context.Context) {
	esc.logger.Info("Elevator Consumer: Started listening for flush signals")

	for {
		select {
		case <-ctx.Done():
			esc.logger.Info("Elevator Consumer: Context cancelled, flushing remaining data")
			esc.FlushDataQueue(ctx)
			return

		case signal := <-esc.flushSignal:
			esc.logger.Infof("Elevator Consumer: Received flush signal: %s", signal)

			switch signal {
			case "queue_threshold":
				esc.logger.Info("Elevator Consumer: Draining queue due to 80% threshold")
				esc.DrainQueue(ctx)

			case "platform.report":
				esc.logger.Info("Elevator Consumer: Draining queue due to platform.report event")
				esc.DrainQueue(ctx)

			default:
				esc.logger.Warnf("Elevator Consumer: Unknown flush signal received: %s", signal)
			}
		}
	}
}

// FlushDataQueue drains the dataqueue completely (called during shutdown)
func (esc *elevatorSumoConsumer) FlushDataQueue(ctx context.Context) {
	esc.logger.Info("Elevator Consumer: Flushing DataQueue")

	if esc.config.EnableFailover {
		var rawMsgArr [][]byte
	Loop:
		for {
			select {
			case rawmsg := <-esc.dataQueue:
				rawMsgArr = append(rawMsgArr, rawmsg)
			default:
				if len(rawMsgArr) > 0 {
					err := esc.sumoclient.FlushAll(rawMsgArr)
					if err != nil {
						esc.logger.Errorln("Elevator Consumer: Unable to flush DataQueue", err.Error())
						// putting back all the msg to the queue in case of failure
						for _, msg := range rawMsgArr {
							select {
							case esc.dataQueue <- msg:
							default:
								esc.logger.Warnf("Elevator Consumer: Failed to requeue message, queue full")
							}
						}
					} else {
						esc.logger.Infof("Elevator Consumer: Successfully flushed %d messages", len(rawMsgArr))
					}
				}
				close(esc.dataQueue)
				esc.logger.Debugf("Elevator Consumer: DataQueue completely drained and closed")
				break Loop
			}
		}
	} else {
		// calling drainqueue (during shutdown) if failover is not enabled
		maxCallsNeededForCompleteDraining := (len(esc.dataQueue) / esc.config.MaxConcurrentRequests) + 1
		for i := 0; i < maxCallsNeededForCompleteDraining; i++ {
			esc.DrainQueue(ctx)
		}
		esc.logger.Info("Elevator Consumer: DataQueue drained without failover")
	}
}

// DrainQueue drains the current contents of the queue
func (esc *elevatorSumoConsumer) DrainQueue(ctx context.Context) int {
	esc.logger.Debug("Elevator Consumer: Draining data from dataQueue")

	var rawMsgArr [][]byte
	var logsStr string
	var runtime_done = 0

	// Collect all available messages from the queue
Loop:
	for {
		select {
		case rawmsg := <-esc.dataQueue:
			rawMsgArr = append(rawMsgArr, rawmsg)
			logsStr = string(rawmsg)
			esc.logger.Debugf("Elevator Consumer: DrainQueue: logsStr length: %d", len(logsStr))

		default:
			// No more messages in queue, send what we have
			if len(rawMsgArr) > 0 {
				esc.logger.Infof("Elevator Consumer: Sending %d messages to Sumo Logic", len(rawMsgArr))
				err := esc.sumoclient.SendAllLogs(ctx, rawMsgArr)
				if err != nil {
					esc.logger.Errorln("Elevator Consumer: Unable to send logs to Sumo Logic", err.Error())
					// putting back all the msg to the queue in case of failure
					for _, msg := range rawMsgArr {
						select {
						case esc.dataQueue <- msg:
						default:
							esc.logger.Warn("Elevator Consumer: Failed to requeue message, queue full")
						}
					}
				} else {
					esc.logger.Infof("Elevator Consumer: Successfully sent %d messages", len(rawMsgArr))
				}
			} else {
				esc.logger.Debug("Elevator Consumer: No messages to drain")
			}
			break Loop
		}
	}

	esc.logger.Debugf("Elevator Consumer: DrainQueue complete. Runtime done: %d", runtime_done)
	return runtime_done
}
