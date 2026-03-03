package workers

import (
	"context"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"
	sumocli "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/sumoclient"

	"github.com/sirupsen/logrus"
)

// ManagedInstanceTaskConsumer exposes methods for consuming tasks in managed instance mode
type ManagedInstanceTaskConsumer interface {
	Start(context.Context)
	FlushDataQueue(context.Context)
	DrainQueue(context.Context) int
}

// managedInstanceSumoConsumer drains log from dataQueue in managed instance mode
type managedInstanceSumoConsumer struct {
	dataQueue   chan []byte
	flushSignal chan string
	logger      *logrus.Entry
	config      *cfg.LambdaExtensionConfig
	sumoclient  sumocli.LogSender
}

// NewManagedInstanceTaskConsumer returns a new managed instance consumer
// flushSignal channel is used to receive signals from producer to trigger flushing
func NewManagedInstanceTaskConsumer(consumerQueue chan []byte, flushSignal chan string, config *cfg.LambdaExtensionConfig, logger *logrus.Entry) ManagedInstanceTaskConsumer {
	return &managedInstanceSumoConsumer{
		dataQueue:   consumerQueue,
		flushSignal: flushSignal,
		logger:      logger,
		sumoclient:  sumocli.NewLogSenderClient(logger, config),
		config:      config,
	}
}

// Start starts the managed instance consumer in a goroutine to listen for flush signals independently
func (esc *managedInstanceSumoConsumer) Start(ctx context.Context) {
	esc.logger.Info("Starting Managed Instance Consumer")
	go esc.processFlushSignals(ctx)
}

// processFlushSignals continuously listens for flush signals and triggers queue draining
// This runs independently without needing callbacks from main thread
func (esc *managedInstanceSumoConsumer) processFlushSignals(ctx context.Context) {
	esc.logger.Info("Managed Instance Consumer: Started listening for flush signals")

	for {
		select {
		case <-ctx.Done():
			esc.logger.Info("Managed Instance Consumer: Context cancelled, flushing remaining data")
			esc.FlushDataQueue(ctx)
			return

		case signal := <-esc.flushSignal:
			esc.logger.Infof("Managed Instance Consumer: Received flush signal: %s", signal)

			switch signal {
			case "queue_threshold":
				esc.logger.Info("Managed Instance Consumer: Draining queue due to 80% threshold")
				esc.DrainQueue(ctx)

			case "platform.report":
				esc.logger.Info("Managed Instance Consumer: Draining queue due to platform.report event")
				esc.DrainQueue(ctx)

			default:
				esc.logger.Warnf("Managed Instance Consumer: Unknown flush signal received: %s", signal)
			}
		}
	}
}

// FlushDataQueue drains the dataqueue completely (called during shutdown)
func (esc *managedInstanceSumoConsumer) FlushDataQueue(ctx context.Context) {
	esc.logger.Info("Managed Instance Consumer: Flushing DataQueue")

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
						esc.logger.Errorln("Managed Instance Consumer: Unable to flush DataQueue", err.Error())
						// putting back all the msg to the queue in case of failure
						for _, msg := range rawMsgArr {
							select {
							case esc.dataQueue <- msg:
							default:
								esc.logger.Warnf("Managed Instance Consumer: Failed to requeue message, queue full")
							}
						}
					} else {
						esc.logger.Infof("Managed Instance Consumer: Successfully flushed %d messages", len(rawMsgArr))
					}
				}
				close(esc.dataQueue)
				esc.logger.Debugf("Managed Instance Consumer: DataQueue completely drained and closed")
				break Loop
			}
		}
	} else {
		// calling drainqueue (during shutdown) if failover is not enabled
		maxCallsNeededForCompleteDraining := (len(esc.dataQueue) / esc.config.MaxConcurrentRequests) + 1
		for i := 0; i < maxCallsNeededForCompleteDraining; i++ {
			esc.DrainQueue(ctx)
		}
		esc.logger.Info("Managed Instance Consumer: DataQueue drained without failover")
	}
}

// DrainQueue drains the current contents of the queue
func (esc *managedInstanceSumoConsumer) DrainQueue(ctx context.Context) int {
	esc.logger.Debug("Managed Instance Consumer: Draining data from dataQueue")

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
			esc.logger.Debugf("Managed Instance Consumer: DrainQueue: logsStr length: %d", len(logsStr))

		default:
			// No more messages in queue, send what we have
			if len(rawMsgArr) > 0 {
				esc.logger.Infof("Managed Instance Consumer: Sending %d messages to Sumo Logic", len(rawMsgArr))
				err := esc.sumoclient.SendAllLogs(ctx, rawMsgArr)
				if err != nil {
					esc.logger.Errorln("Managed Instance Consumer: Unable to send logs to Sumo Logic", err.Error())
					// putting back all the msg to the queue in case of failure
					for _, msg := range rawMsgArr {
						select {
						case esc.dataQueue <- msg:
						default:
							esc.logger.Warn("Managed Instance Consumer: Failed to requeue message, queue full")
						}
					}
				} else {
					esc.logger.Infof("Managed Instance Consumer: Successfully sent %d messages", len(rawMsgArr))
				}
			} else {
				esc.logger.Debug("Managed Instance Consumer: No messages to drain")
			}
			break Loop
		}
	}

	esc.logger.Debugf("Managed Instance Consumer: DrainQueue complete. Runtime done: %d", runtime_done)
	return runtime_done
}
