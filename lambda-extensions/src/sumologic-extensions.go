package main

import (
	// "bytes"

	// "net/http"
	"bytes"
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
	// Register early so Runtime could start in parallel
	fmt.Println(printPrefix, "Registering Extension to Run Time API Client..........")
	registerResponse, err := extensionClient.RegisterExtension(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during extension registration."+err.Error())
	}
	fmt.Println(printPrefix, "Succcessfully Registered with Run Time API Client: ", PrettyPrint(registerResponse))
	// Start HTTP Server before subscription in a goRoutine
	go httpServer.HTTPServerStart()
	// Subscribe to Logs API
	fmt.Println(printPrefix, "Subscribing Extension to Logs API........")
	subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil)
	if err != nil {
		extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
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
			nextResponse, err := extensionClient.NextEvent(ctx)
			if err != nil {
				fmt.Println(printPrefix, "Error:", err.Error())
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
	for len(lambdaapi.Messages) != 0 {
		// Send the Data to Sumo Logic, Data in JSON Array format already but as a string.
		message := <-lambdaapi.Messages
		response, err := sendtosumo(message)
		if err != nil {
			extensionClient.ExitError(ctx, "Send to Sumo Fail"+string(response))
		}
	}
}

func sendtosumo(message string) ([]byte, error) {
	response, err := extensionClient.MakeRequest(nil, bytes.NewBuffer([]byte(message)), "POST", "https://collectors.au.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ysAXzCTwVXvj4Vw7QgS3nJ8nwOqiFLweEr_2VSqRQTeBz6mq0IQWIhR5G41qh4eQAhGImhQDt6Y75wHL5F8DJoyuush7AXp88rtIa0si-0A==")
	return response, err
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
		fmt.Println(printPrefix, "Received", s)
	}()
	//Call Send to Sumo here in a different GO Routine with Context, to find out if the things are done
	go processLogs(ctx)
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
