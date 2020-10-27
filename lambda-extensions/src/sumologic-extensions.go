package main

import (
	// "bytes"
	"fmt"
	// "net/http"
	"os"
	"path/filepath"
	"time"
	"sumoclient"
	"lambdaapi"

	log "github.com/sirupsen/logrus"
)

const (
	// Subscription Body Constants. Subscribe to platform logs and receive them on ${local_ip}:4243 via HTTP protocol.
	timeoutMs = 1000
	maxBytes  = 262144
	maxLen    = 102
)

// SumoLogicExtension is the type struct for Extension API.
type SumoLogicExtension struct {
	agentName          string
	registrationBody   string
	subscriptionBody   string
	agentID            string
	extensionAPIClient *lambdaapi.LambdaExtensionAPIClient
	httpServer         *lambdaapi.HTTPServer
}

// NewSumoLogicExtension is - Creating a new object.
func NewSumoLogicExtension(agentName, registrationBody, subscriptionBody string) *SumoLogicExtension {
	return &SumoLogicExtension{
		agentName:        agentName,
		registrationBody: registrationBody,
		subscriptionBody: subscriptionBody,
	}
}

// SetUp is - calling the Extension RunTime API, HTTP listener and Logs API subscription.
func (sumoLogicExtension *SumoLogicExtension) SetUp() {
	log.WithFields(log.Fields{
		"agentName": sumoLogicExtension.agentName,
	}).Info("Intializing Sumo Logic Extension with ")
	sumoLogicExtension.extensionAPIClient = lambdaapi.NewLambdaExtensionAPIClient(sumoLogicExtension.agentName, sumoLogicExtension.registrationBody)
	// Register early so Runtime could start in parallel
	sumoLogicExtension.agentID = sumoLogicExtension.extensionAPIClient.Register()
	// Start listening before Logs API registration
	sumoLogicExtension.httpServer = lambdaapi.NewHTTPServer()
	go sumoLogicExtension.httpServer.HTTPServerInit()
	// Subscribe to Logs API
	logsAPIClient := lambdaapi.NewLambdaLogsAPIClient(sumoLogicExtension.agentID, sumoLogicExtension.subscriptionBody)
	logsAPIClient.Subscribe()
}

// RunForever is - listening to the LOGS API using next method.
func (sumoLogicExtension *SumoLogicExtension) RunForever() {
	log.WithFields(log.Fields{
		"agentName": sumoLogicExtension.agentName,
	}).Info("Serving Sumo Logic Extension with ")
	for {
		sumoLogicExtension.extensionAPIClient.Next(sumoLogicExtension.agentID)
		time.Sleep(1 * time.Second)
		for len(sumoLogicExtension.httpServer.Queue) != 0 {
			// Send the Data to Sumo Logic, Data in JSON Array format already but as a string.
			data := sumoLogicExtension.httpServer.Queue[0]
			sumoLogicExtension.httpServer.Queue = sumoLogicExtension.httpServer.Queue[1:]
			// Test Code. Needs to be replaced with SendToSumo Code.
			sumocli := sumoclient.NewSumoLogicClient()
			sumocli.SendToSumo(data)
		}
	}
}

func main() {
	// Register for the INVOKE events from the RUNTIME API
	registrationBody := `{"events": ["INVOKE"]}`

	subscriptionBody := fmt.Sprintf(`{
        "destination": {
            "protocol": "HTTP",
            "URI": "http://sandbox:%v"
        },
        "types": ["platform", "function"],
        "buffering": {
            "timeoutMs": %v,
            "max_bytes": %v,
            "max_len": %v
        }
    }`, lambdaapi.ReceiverPort, timeoutMs, maxBytes, maxLen)

	log.WithFields(log.Fields{
		"registrationBody": registrationBody, "subscriptionBody": subscriptionBody,
	}).Info("Starting Sumo Logic Extension ")
	// Note: Agent name has to be file name to register as an external extension
	sumoLogicExtension := NewSumoLogicExtension(filepath.Base(os.Args[0]), registrationBody, subscriptionBody)
	sumoLogicExtension.SetUp()
	sumoLogicExtension.RunForever()
}
