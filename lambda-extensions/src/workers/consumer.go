package workers

import (
	cfg "config"
	"context"
	sumocli "sumoclient"
	"sync"

	"github.com/sirupsen/logrus"
)

// TaskConsumer exposing methods every consmumer should implement
type TaskConsumer interface {
	FlushDataQueue()
	DrainQueue(context.Context)
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
	rawMsgArr := make([][]byte, sc.config.MaxDataQueueLength)
	for {
		rawmsg, more := <-sc.dataQueue
		if !more {
			err := sc.sumoclient.FlushAll(rawMsgArr)
			if err != nil {
				sc.logger.Debugln("Unable to flushing DataQueue", err.Error())
			}
			close(sc.dataQueue)
			return
		}
		rawMsgArr = append(rawMsgArr, rawmsg)
	}

}

func (sc *sumoConsumer) consumeTask(ctx context.Context, wg *sync.WaitGroup, rawmsg []byte) {
	defer wg.Done()
	err := sc.sumoclient.SendLogs(ctx, rawmsg)
	if err != nil {
		sc.logger.Error("Error during Send Logs to Sumo Logic.", err.Error())
	}
	return
}

func (sc *sumoConsumer) DrainQueue(ctx context.Context) {
	wg := new(sync.WaitGroup)
	sc.logger.Debug("Consuming data from dataQueue")
	counter := 0
	for i := 0; i < sc.config.MaxConcurrentRequests; i++ {
		rawmsg := <-sc.dataQueue // read from a closed channel will be the zero value
		if len(rawmsg) > 0 {
			counter++
			wg.Add(1)
			go sc.consumeTask(ctx, wg, rawmsg)
		} else {
			break
		}
	}
	sc.logger.Debugf("Waiting for %d consumer to finish their tasks", counter)
	wg.Wait()
}
