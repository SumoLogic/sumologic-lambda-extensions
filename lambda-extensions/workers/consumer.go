package workers

import (
	"context"
	"sync"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"
	sumocli "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/sumoclient"

	"github.com/sirupsen/logrus"
)

// TaskConsumer exposing methods every consmumer should implement
type TaskConsumer interface {
	FlushDataQueue()
	DrainQueue(context.Context, int64) int
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
func (sc *sumoConsumer) FlushDataQueue() {
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
				sc.logger.Debugln("Unable to flush DataQueue", err.Error())
				// TODO: raise alert if flush fails
			}
			close(sc.dataQueue)
			sc.logger.Debugf("DataQueue completely drained")
			break Loop
		}
	}

}

func (sc *sumoConsumer) consumeTask(ctx context.Context, wg *sync.WaitGroup, rawmsg []byte) {
	defer wg.Done()
	err := sc.sumoclient.SendLogs(ctx, rawmsg)
	if err != nil {
		sc.logger.Error("Error during Send Logs to Sumo Logic.", err.Error())
		// TODO: raise alert if send logs fails
	}
	return
}

func (sc *sumoConsumer) DrainQueue(ctx context.Context, deadtimems int64) int {
	wg := new(sync.WaitGroup)
	//sc.logger.Debug("Consuming data from dataQueue")
	counter := 0
Loop:
	for i := 0; i < sc.config.MaxConcurrentRequests && len(sc.dataQueue) != 0; i++ {
		//Receives block when the buffer is empty.
		select {
		case rawmsg := <-sc.dataQueue:
			counter++
			wg.Add(1)
			go sc.consumeTask(ctx, wg, rawmsg)
		default:
			sc.logger.Debugf("DataQueue completely drained")
			break Loop
		}

	}
	//sc.logger.Debugf("Waiting for %d consumer to finish their tasks", counter)
	wg.Wait()
	return counter
}
