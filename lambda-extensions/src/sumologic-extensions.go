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

	// Creating config and performing validation
	var err error
	config, err = cfg.GetConfig()
	if err != nil {
		extensionClient.InitError(nil, "Error during Fetching Env Variables."+err.Error())
	}

	logger.Logger.SetLevel(config.LogLevel)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)
	quitQueue = make(chan bool, 1)

	// Start HTTP Server before subscription in a goRoutine
	producer = workers.NewTaskProducer(dataQueue, quitQueue, logger)
	go producer.Start()

	// Creating SumoTaskConsumer
	consumer = workers.NewTaskConsumer(dataQueue, config, logger)
}

func runTimeAPIInit() int64 {
	// Register early so Runtime could start in parallel
	logger.Debug("Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during extension registration."+err.Error())
		panic(err)
	}
	logger.Debug("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))

	// Subscribe to Logs API
	logger.Debug("Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
	if err != nil {
		extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
		panic(err)
	}
	logger.Debug("Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))

	// Call next to say registration is successful and get the deadtimems
	nextResponse := nextEvent(nil)
	return nextResponse.DeadlineMs
}

func nextEvent(ctx context.Context) *lambdaapi.NextEventResponse {
	nextResponse, err := extensionClient.NextEvent(ctx)
	if err != nil {
		logger.Error("Error:", err.Error())
		logger.Info("Exiting")
		return nil
	}
	logger.Debugf("Received EventType: %s as: %v", nextResponse.EventType, nextResponse)
	return nextResponse
}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	DeadlineMs := runTimeAPIInit()
	var totalMessagedProcessed int
	startTime := time.Now()
	// The For loop will continue till we recieve a shutdown event.
	for {
		select {
		case <-ctx.Done():
			consumer.FlushDataQueue()
			close(quitQueue)
			return
		default:
			currentMessagedProcessed := consumer.DrainQueue(ctx, DeadlineMs)
			messagesChanged, durationComplete := utils.TotalMessagesCountChanged(totalMessagedProcessed, totalMessagedProcessed+currentMessagedProcessed, config.ProcessingSleepTime, startTime)
			totalMessagedProcessed = totalMessagedProcessed + currentMessagedProcessed
			// Call the next event is we reach timeout or no new message are received based on sleep time.
			if !utils.IsTimeRemaining(DeadlineMs) || durationComplete {
				logger.Debugf("Total Messages: %v, Current Messages: %v, messages changes: %s, duration Complete: %s, start Time: %s, Sleep Time: %s", totalMessagedProcessed, currentMessagedProcessed, messagesChanged, durationComplete, startTime, config.ProcessingSleepTime)
				logger.Info("Waiting for Run Time API event...")
				// This statement will freeze lambda
				nextResponse := nextEvent(ctx)
				// Next invoke will start from here
				logger.Infof("Received Next Event as %s", nextResponse.EventType)
				DeadlineMs = nextResponse.DeadlineMs
				if nextResponse.EventType == lambdaapi.Shutdown {
					consumer.FlushDataQueue()
					close(quitQueue)
					return
				}
				totalMessagedProcessed = 0
			}
			if messagesChanged {
				startTime = time.Now()
			}
		}
	}
}

func main() {
	logger.Info("Starting the Sumo Logic Extension................")
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
	logger.Info("Stopping the Sumo Logic Extension................")
}
