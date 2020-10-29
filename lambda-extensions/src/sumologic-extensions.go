package main

import (
	cfg "config"
	"context"
	"fmt"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	sumocli "sumoclient"
	"syscall"
	"utils"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	printPrefix     = fmt.Sprintf("[%s]", extensionName)
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	dataQueue       = make(chan []byte)
	httpServer      = lambdaapi.NewHTTPServer(dataQueue)
	sumoclient      = sumocli.NewLogSenderClient()
)

var config *cfg.LambdaExtensionConfig

func init() {
	fmt.Println(printPrefix, "Initializing Extension.........")
	// Register early so Runtime could start in parallel
	fmt.Println(printPrefix, "Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during extension registration."+err.Error())
	}
	fmt.Println(printPrefix, "Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))
	// Start HTTP Server before subscription in a goRoutine
	go httpServer.HTTPServerStart()

	// Creating config and performing validation
	config, err = cfg.GetConfig()
	if err != nil {
		extensionClient.InitError(nil, "Error during Fetching Env Variables."+err.Error())
	}
	// Subscribe to Logs API
	fmt.Println(printPrefix, "Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
	if err != nil {
		extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
	}
	fmt.Println(printPrefix, "Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))
	fmt.Println(printPrefix, "Successfully Intialized Sumo Logic Extension.")

}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case rawmsg := <-dataQueue:
			fmt.Println("Consuming data from dataQueue")
			err := sumoclient.SendLogs(rawmsg)
			if err != nil {
				extensionClient.ExitError(nil, "Error during Send Logs to Sumo Logic."+err.Error())
				return
			}
		default:
			fmt.Println(printPrefix, "Waiting for Run Time API event...")
			nextResponse, err := extensionClient.NextEvent(ctx)
			if err != nil {
				fmt.Println(printPrefix, "Error:", err.Error())
				fmt.Println(printPrefix, "Exiting")
				return
			}
			//println(printPrefix, "Received Run Time API event:", utils.PrettyPrint(nextResponse))
			// Exit if we receive a SHUTDOWN event
			if nextResponse.EventType == lambdaapi.Shutdown {
				fmt.Println(printPrefix, "Received SHUTDOWN event")
				// TODO: do something here and if failed send a ExitError
				fmt.Println(printPrefix, "Exiting")
				return
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
		fmt.Println(printPrefix, "Received", s)
	}()
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
