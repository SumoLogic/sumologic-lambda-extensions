package workers

import (
	"context"
	"strings"
	"sync"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"
	sumocli "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/sumoclient"

	"github.com/sirupsen/logrus"
)

type SubEventType string

const (
	// RuntimeDone event is sent when lambda function is finished it's execution
	RuntimeDone SubEventType = "platform.runtimeDone"
)

// TaskConsumer exposing methods every consmumer should implement
type TaskConsumer interface {
	FlushDataQueue(context.Context)
	DrainQueue(context.Context) int
}

// sumoConsumer to drain log from dataQueue
type sumoConsumer struct {
	dataQueue  chan []byte
	logger     *logrus.Entry
	config     *cfg.LambdaExtensionConfig
	sumoclient sumocli.LogSender
}

// NewTaskConsumer returns a new consumer
func NewTaskConsumer(consumerQueue chan []byte, config *cfg.LambdaExtensionConfig, logger *logrus.Entry) TaskConsumer {
	return &sumoConsumer{
		dataQueue:  consumerQueue,
		logger:     logger,
		sumoclient: sumocli.NewLogSenderClient(logger, config),
		config:     config,
	}
}

// FlushDataQueue drains the dataqueue commpletely
func (sc *sumoConsumer) FlushDataQueue(ctx context.Context) {
	if sc.config.EnableFailover {
		var rawMsgArr [][]byte
	Loop:
		for {
			//Receives block when the buffer is empty.
			select {
			case rawmsg := <-sc.dataQueue:
				rawMsgArr = append(rawMsgArr, rawmsg)
			default:
				err := sc.sumoclient.FlushAll(rawMsgArr)
				if err != nil {
					sc.logger.Errorln("Unable to flush DataQueue", err.Error())
					// putting back all the msg to the queue in case of failure
					for _, msg := range rawMsgArr {
						sc.dataQueue <- msg
					}
					// TODO: raise alert if flush fails
				}
				close(sc.dataQueue)
				sc.logger.Debugf("DataQueue completely drained")
				break Loop
			}
		}
	} else {
		// calling drainqueue (during shutdown) if failover is not enabled
		maxCallsNeededForCompleteDraining := (len(sc.dataQueue) / sc.config.MaxConcurrentRequests) + 1
		for i := 0; i < maxCallsNeededForCompleteDraining; i++ {
			sc.DrainQueue(ctx)
		}
	}

}

func (sc *sumoConsumer) consumeTask(ctx context.Context, wg *sync.WaitGroup, rawmsg []byte) {
	defer wg.Done()
	err := sc.sumoclient.SendLogs(ctx, rawmsg)
	if err != nil {
		sc.logger.Error("Error during Send Logs to Sumo Logic.", err.Error())
		// putting back the msg to the queue in case of failure
		sc.dataQueue <- rawmsg
		// TODO: raise alert if send logs fails
	}
}

func (sc *sumoConsumer) DrainQueue(ctx context.Context) int {
	//sc.logger.Debug("Consuming data from dataQueue")

	var rawMsgArr [][]byte
	var logsStr string
	var runtime_done = 0
Loop:
	for {
		//Receives block when the buffer is empty.
		select {
		case rawmsg := <-sc.dataQueue:
			rawMsgArr = append(rawMsgArr, rawmsg)
			logsStr = string(rawmsg)
			sc.logger.Debugf("DrainQueue: logsStr: %s", logsStr)
			if strings.Contains(logsStr, string(RuntimeDone)) {
				runtime_done = 1
			}

		default:
			err := sc.sumoclient.SendAllLogs(ctx, rawMsgArr)
			if err != nil {
				sc.logger.Errorln("Unable to flush DataQueue", err.Error())
				// putting back all the msg to the queue in case of failure
				for _, msg := range rawMsgArr {
					sc.dataQueue <- msg
				}
				// TODO: raise alert if flush fails
			} else {
				sc.logger.Debugf("DrainQueue: DataQueue completely drained")
			}
			break Loop
		}
	}
	sc.logger.Debugf("DrainQueue: Runtime done or not? %d", runtime_done)
	return runtime_done
}
