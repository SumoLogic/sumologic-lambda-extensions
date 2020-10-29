package main

import (
	cfg "config"
	"context"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	sumocli "sumoclient"
	"syscall"
	"utils"

	"github.com/sirupsen/logrus"
)

const (
	maxchannelLength       = 20
	maxConcurrentConsumers = 3
)

var (
	logger          = logrus.New().WithField("Name", extensionName)
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	dataQueue       = make(chan []byte, 20)
	httpServer      = lambdaapi.NewHTTPServer(dataQueue, logger)
	sumoclient      = sumocli.NewLogSenderClient(logger)
)

var config *cfg.LambdaExtensionConfig

func init() {
	logger.Info("Initializing Extension.........")
	// Register early so Runtime could start in parallel
	logger.Info("Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during extension registration."+err.Error())
	}
	logger.Info("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))
	// Start HTTP Server before subscription in a goRoutine
	go httpServer.HTTPServerStart()

	// Creating config and performing validation
	config, err = cfg.GetConfig()
	if err != nil {
		extensionClient.InitError(nil, "Error during Fetching Env Variables."+err.Error())
	}
	// Subscribe to Logs API
	logger.Info("Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
	if err != nil {
		extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
	}
	logger.Info("Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))
	logger.Info("Successfully Intialized Sumo Logic Extension.")

}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case rawmsg := <-dataQueue:
			logger.Debug("Consuming data from dataQueue")
			err := sumoclient.SendLogs(ctx, rawmsg)
			if err != nil {
				extensionClient.ExitError(ctx, "Error during Send Logs to Sumo Logic."+err.Error())
				return
			}

		default:
			logger.Info("Waiting for Run Time API event...")
			nextResponse, err := extensionClient.NextEvent(ctx)
			if err != nil {
				logger.Error("Error:", err.Error())
				logger.Info("Exiting")
				return
			}
			// Exit if we receive a SHUTDOWN event
			if nextResponse.EventType == lambdaapi.Shutdown {
				logger.Info("Received SHUTDOWN event")
				logger.Info("Exiting")
				return
			} else if nextResponse.EventType == lambdaapi.Invoke {
				logger.Info("Received Invoke event.", utils.PrettyPrint(nextResponse))
			}
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
