package main

import (
	// "bytes"

	// "net/http"
	"context"
	"encoding/json"
	"fmt"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	printPrefix     = fmt.Sprintf("[%s]", extensionName)
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	httpServer      = lambdaapi.NewHTTPServer()
)

func init() {
	fmt.Println(printPrefix, "Initializing Extension.........")
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		fmt.Println(printPrefix, "Received", s)
	}()
	// Register early so Runtime could start in parallel
	fmt.Println(printPrefix, "Registering Extension to Run Time API Client..........")
	registerResponse, error := extensionClient.RegisterExtension(ctx)
	if error != nil {
		panic(error)
	}
	fmt.Println(printPrefix, "Succcessfully Registered with Run Time API Client: ", PrettyPrint(registerResponse))
	// Start HTTP Server before subscription in a goRoutine
	go httpServer.HTTPServerStart()
	// Subscribe to Logs API
	fmt.Println(printPrefix, "Subscribing Extension to Logs API........")
	subscribeResponse, error := extensionClient.SubscribeToLogsAPI(ctx)
	if error != nil {
		panic(error)
	}
	fmt.Println(printPrefix, "Successfully subscribed to Logs API: ", PrettyPrint(string(subscribeResponse)))
	fmt.Println(printPrefix, "Successfully Intialized Sumo Logic Extension.")
}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println(printPrefix, "Waiting for Run Time API event...")
			nextResponse, error := extensionClient.NextEvent(ctx)
			if error != nil {
				fmt.Println(printPrefix, "Error:", error.Error())
				fmt.Println(printPrefix, "Exiting")
				return
			}
			//println(printPrefix, "Received Run Time API event:", PrettyPrint(nextResponse))
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

func processLogs(ctx context.Context) {
	for len(httpServer.Queue) != 0 {
		// Send the Data to Sumo Logic, Data in JSON Array format already but as a string.
		data := httpServer.Queue[0]
		httpServer.Queue = httpServer.Queue[1:]
	}
}

// PrettyPrint is to print the object
func PrettyPrint(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return ""
	}
	return string(data)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		println(printPrefix, "Received", s)
	}()

	//Call Send to Sumo here in a different GO Routine with Context, to find out if the things are done
	go processLogs(ctx)
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
