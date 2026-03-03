package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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
var managedInstanceProducer workers.ManagedInstanceTaskProducer
var managedInstanceConsumer workers.ManagedInstanceTaskConsumer
var config *cfg.LambdaExtensionConfig
var dataQueue chan []byte
var flushSignal chan string
var isManagedInstance bool

func init() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "2006-01-02T15:04:05.999999999Z07:00"
	Formatter.FullTimestamp = true
	logger.Logger.SetFormatter(Formatter)

	logger.Logger.SetOutput(os.Stdout)

	// Creating config and performing validation
	var err error
	config, err = cfg.GetConfig()
	if err != nil {
		logger.Error("Error during Fetching Env Variables: ", err.Error())
	}

	logger.Logger.SetLevel(config.LogLevel)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)

	// Check initialization type to determine if managed instance mode should be used
	initializationType := os.Getenv("AWS_LAMBDA_INITIALIZATION_TYPE")
	if initializationType == "lambda-managed-instances" {
		isManagedInstance = true
		logger.Debug("Initializing in Managed Instance mode")

		// Initialize flushSignal channel for managed instance mode communication
		flushSignal = make(chan string, 10) // Buffered channel to prevent blocking

		// Initialize Managed Instance Producer and start it in a goroutine
		managedInstanceProducer = workers.NewManagedInstanceTaskProducer(dataQueue, flushSignal, logger)
		go func() {
			if err := managedInstanceProducer.Start(); err != nil {
				logger.Errorf("managedInstanceProducer Start failed: %v", err)
			}
		}()

		// Initialize Managed Instance Consumer and start it
		managedInstanceConsumer = workers.NewManagedInstanceTaskConsumer(dataQueue, flushSignal, config, logger)
		// Start the consumer's independent processing loop
		ctx := context.Background()
		managedInstanceConsumer.Start(ctx)

		logger.Debug("Managed Instance mode initialization complete")
	} else {
		logger.Debug("Initializing in standard mode")
		// Start HTTP Server before subscription in a goRoutine
		producer = workers.NewTaskProducer(dataQueue, logger)
		go func() {
			if err := producer.Start(); err != nil {
				logger.Errorf("producer Start failed: %v", err)
			}
		}()

		// Creating SumoTaskConsumer
		consumer = workers.NewTaskConsumer(dataQueue, config, logger)
		logger.Debug("Standard mode initialization complete")
	}

	logger.Debug("Is Managed Instance value: ", isManagedInstance)
}

func runTimeAPIInit() (int64, error) {
	// Register early so Runtime could start in parallel
	logger.Debug("Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(context.TODO(), isManagedInstance)
	if err != nil {
		return 0, err
	}
	logger.Debug("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))

	// Subscribe to Telemetry API
	logger.Debug("Subscribing Extension to Telemetry API........")
	subscribeResponse, err := extensionClient.SubscribeToTelemetryAPI(context.TODO(), config.LogTypes, config.TelemetryTimeoutMs, config.TelemetryMaxBytes, config.TelemetryMaxItems, isManagedInstance)
	if err != nil {
		return 0, err
	}

	logger.Debug("Successfully subscribed to Telemetry API: ", utils.PrettyPrint(string(subscribeResponse)))

	// Call next to say registration is successful and get the deadtimems
	if !isManagedInstance {
		nextResponse, err := nextEvent(context.TODO())
		if err != nil {
			return 0, err
		}
		return nextResponse.DeadlineMs, nil
	}
	return 0, nil
}

func nextEvent(ctx context.Context) (*lambdaapi.NextEventResponse, error) {
	nextResponse, err := extensionClient.NextEvent(ctx)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Received EventType: %s as: %v", nextResponse.EventType, nextResponse)
	return nextResponse, nil
}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	_, err := runTimeAPIInit()
	if err != nil {
		logger.Error("Error during Registration: ", err.Error())
		return
	}

	// The For loop will continue till we recieve a shutdown event.
	for {
		select {
		case <-ctx.Done():
			consumer.FlushDataQueue(ctx)
			return
		default:
			if !isManagedInstance {
				logger.Debugf("switching to other go routine")
				runtime.Gosched()
				logger.Infof("Calling DrainQueue from processEvents")
				// for {
				runtime_done := consumer.DrainQueue(ctx)
				if runtime_done == 1 {
					logger.Infof("Exiting DrainQueueLoop: Runtime is Done")
				}
			}

			// }

			// This statement will freeze lambda
			nextResponse, err := nextEvent(ctx)
			if err != nil {
				logger.Error("Error during Next Event call: ", err.Error())
				return
			}
			// Next invoke will start from here
			logger.Infof("Received Next Event as %s", nextResponse.EventType)
			if nextResponse.EventType == lambdaapi.Shutdown {
				consumer.DrainQueue(ctx)
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
			_, err := nextEvent(ctx)
			if err != nil {
				logger.Error("error during Next Event call: ", err.Error())
			}
		}
	}()
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
	logger.Info("Stopping the Sumo Logic Extension................")
}
