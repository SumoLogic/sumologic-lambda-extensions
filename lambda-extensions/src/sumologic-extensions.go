package main

import (
	cfg "config"
	"context"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"utils"
	"workers"

	"github.com/sirupsen/logrus"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	logger          = logrus.New().WithField("Name", extensionName)
)
var producer workers.TaskProducer
var consumer workers.TaskConsumer
var config *cfg.LambdaExtensionConfig
var dataQueue chan []byte
var quitQueue chan bool

func init() {

	logger.Logger.SetOutput(os.Stdout)

	logger.Info("Initializing Extension.........")
	// Register early so Runtime could start in parallel
	logger.Info("Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during extension registration."+err.Error())
		panic(err)
	}
	logger.Info("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))

	// Creating config and performing validation
	config, err = cfg.GetConfig()
	if err != nil {
		extensionClient.InitError(nil, "Error during Fetching Env Variables."+err.Error())
		panic(err)
	}

	logger.Logger.SetLevel(config.LogLevel)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)
	quitQueue = make(chan bool, 1)

	// Start HTTP Server before subscription in a goRoutine
	producer = workers.NewTaskProducer(dataQueue, quitQueue, logger)
	go producer.Start()

	// Creating SumoTaskConsumer
	consumer = workers.NewTaskConsumer(dataQueue, config, logger)

	// Subscribe to Logs API
	logger.Info("Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
	if err != nil {
		extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
		panic(err)
	}
	logger.Info("Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))
	nextResponse, err := extensionClient.NextEvent(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during Next API call."+err.Error())
		panic(err)
	}
	logger.Info("Received Invoke event.", utils.PrettyPrint(nextResponse))
	logger.Info("Successfully Intialized Sumo Logic Extension.")

}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	receivedPayloadAfterInvoke := false
	for {
		select {
		case <-ctx.Done():
			consumer.FlushDataQueue()
			close(quitQueue)
			return
		default:
			messagedProcessed := consumer.DrainQueue(ctx)
			if messagedProcessed > 0 {
				receivedPayloadAfterInvoke = true
			}
			if len(dataQueue) == 0 && receivedPayloadAfterInvoke {
				logger.Info("Waiting for Run Time API event...")
				nextResponse, err := extensionClient.NextEvent(ctx)
				if err != nil {
					logger.Error("Error:", err.Error())
					logger.Info("Exiting")
					return
				}
				logger.Debugf("Received %s, %v", nextResponse.EventType, nextResponse)
				// Exit if we receive a SHUTDOWN event
				if nextResponse.EventType == lambdaapi.Shutdown {
					logger.Info("Received SHUTDOWN event")
					consumer.FlushDataQueue()
					close(quitQueue)
					return
				} else if nextResponse.EventType == lambdaapi.Invoke {
					logger.Info("Received Invoke event.", utils.PrettyPrint(nextResponse))
					receivedPayloadAfterInvoke = false
					break
				}
			}
			if len(dataQueue) == 0 && !receivedPayloadAfterInvoke {
				time.Sleep(config.ProcessingSleepTime) // since shutdown time is 2000ms
			}
			break

		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		logger.Info("Received", s)
	}()

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
