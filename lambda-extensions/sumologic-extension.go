package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/utils"

	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/lambdaapi"
	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/workers"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"

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

func init() {
	logger.Logger.SetOutput(os.Stdout)

	// Creating config and performing validation
	var err error
	config, err = cfg.GetConfig()
	if err != nil {
		logger.Error("Error during Fetching Env Variables: ", err.Error())
	}

	logger.Logger.SetLevel(config.LogLevel)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)

	// Start HTTP Server before subscription in a goRoutine
	producer = workers.NewTaskProducer(dataQueue, logger)
	go producer.Start()

	// Creating SumoTaskConsumer
	consumer = workers.NewTaskConsumer(dataQueue, config, logger)
}

func runTimeAPIInit() (int64, error) {
	// Register early so Runtime could start in parallel
	logger.Debug("Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		return 0, err
	}
	logger.Debug("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))

	// Subscribe to Logs API
	logger.Debug("Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
	if err != nil {
		return 0, err
	}
	logger.Debug("Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))

	// Call next to say registration is successful and get the deadtimems
	nextResponse, err := nextEvent(nil)
	if err != nil {
		return 0, err
	}
	return nextResponse.DeadlineMs, nil
}

func nextEvent(ctx context.Context) (*lambdaapi.NextEventResponse, error) {
	nextResponse, err := extensionClient.NextEvent(ctx)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Received EventType: %s as: %v", nextResponse.EventType, nextResponse)
	return nextResponse, nil
}

func flushData(ctx context.Context, DeadlineMs int64) {
	if config.EnableFailover {
		consumer.FlushDataQueue()
	} else {
		consumer.DrainQueue(ctx, DeadlineMs)
	}
}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	DeadlineMs, err := runTimeAPIInit()
	if err != nil {
		logger.Error("Error during Registration: ", err.Error())
		return
	}
	// The For loop will continue till we recieve a shutdown event.
	for {
		select {
		case <-ctx.Done():
			flushData(ctx, 0)
			return
		default:
			go consumer.DrainQueue(ctx, DeadlineMs)
			// This statement will freeze lambda
			nextResponse, err := nextEvent(ctx)
			if err != nil {
				logger.Error("Error during Next Event call: ", err.Error())
				return
			}
			// Next invoke will start from here
			logger.Infof("Received Next Event as %s", nextResponse.EventType)
			if nextResponse.EventType == lambdaapi.Shutdown {
				flushData(ctx, 0)
				return
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
	defer func() {
		if err := recover(); err != nil {
			logger.Error("Extension failed:", err)
			nextEvent(ctx)
		}
	}()
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
	logger.Info("Stopping the Sumo Logic Extension................")
}
