package lambdaapi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// LambdaLogsAPIClient is the type struct for Logs API.
type LambdaLogsAPIClient struct {
	runTimeAPIURL    string
	agentIdentifier  string
	subscriptionBody string
}

// NewLambdaLogsAPIClient is - Getting the AWS_LAMBDA_RUNTIME_API env variable and Creating a new object.
func NewLambdaLogsAPIClient(agentIdentifier, subscriptionBody string) *LambdaLogsAPIClient {
	// Should be "127.0.0.1:9001"
	apiAddress := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	return &LambdaLogsAPIClient{
		runTimeAPIURL:    fmt.Sprintf("http://%v/2020-08-15", apiAddress),
		agentIdentifier:  agentIdentifier,
		subscriptionBody: subscriptionBody,
	}
}

// Subscribe is - Subscribe to Logs API to receive the Lambda Logs.
func (client *LambdaLogsAPIClient) Subscribe() {
	log.WithFields(log.Fields{
		"apiAddress": client.runTimeAPIURL,
	}).Info("Subscribing to Logs API on ")

	request, error := http.NewRequest("PUT", client.runTimeAPIURL+"/logs", bytes.NewBuffer([]byte(client.subscriptionBody)))
	if error != nil {
		log.Fatalln(error)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(lambdaAgentIdentifierHeaderKey, client.agentIdentifier)

	timeout := time.Duration(5 * time.Second)
	httpClient := http.Client{Timeout: timeout}

	response, error := httpClient.Do(request)
	if error != nil {
		log.Fatalln(error)
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.WithFields(log.Fields{
			"Status": response.Status, "Body": response.Body,
		}).Error("Could not subscribe to Logs API. ")
	}
	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
		log.Fatalln(error)
	}
	log.WithFields(log.Fields{
		"Response": string(body),
	}).Info("Successfully subscribed to Logs API: ")
}
