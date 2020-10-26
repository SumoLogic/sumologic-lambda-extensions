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

const (
	// Lambda Extension API Header Keys
	lambdaAgentNameHeaderKey       = "Lambda-Extension-Name"
	lambdaAgentIdentifierHeaderKey = "Lambda-Extension-Identifier"
)

// LambdaExtensionAPIClient is the type struct for Extension API.
type LambdaExtensionAPIClient struct {
	runTimeAPIURL    string
	agentName        string
	registrationBody string
}

// NewLambdaExtensionAPIClient is - Getting the AWS_LAMBDA_RUNTIME_API env variable and Creating a new object.
func NewLambdaExtensionAPIClient(agentName, registrationBody string) *LambdaExtensionAPIClient {
	// Should be "127.0.0.1:9001"
	apiAddress := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	return &LambdaExtensionAPIClient{
		runTimeAPIURL:    fmt.Sprintf("http://%v/2020-01-01/extension", apiAddress),
		agentName:        agentName,
		registrationBody: registrationBody,
	}
}

// Register is - Call the following method on initialization as early as possible, otherwise you may get a timeout error.
// Runtime initialization will start after all extensions are registered.
func (client *LambdaExtensionAPIClient) Register() string {
	log.WithFields(log.Fields{
		"apiAddress": client.runTimeAPIURL,
	}).Info("Registering to RuntimeAPIClient on ")

	request, error := http.NewRequest("POST", client.runTimeAPIURL+"/register", bytes.NewBuffer([]byte(client.registrationBody)))
	if error != nil {
		log.Fatalln(error)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(lambdaAgentNameHeaderKey, client.agentName)

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
		}).Error("Register request to RuntimeAPIClient failed. ")
	}
	agentIdentifier := response.Header.Get(lambdaAgentIdentifierHeaderKey)
	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
		log.Fatalln(error)
	}
	log.WithFields(log.Fields{
		"Response": string(body),
	}).Info("Registered with RuntimeAPIClient: ")
	return agentIdentifier
}

// Next is - Call the following method when the extension is ready to receive the next invocation
// and there is no job it needs to execute beforehand.
func (client *LambdaExtensionAPIClient) Next(agentIdentifier string) string {
	log.WithFields(log.Fields{
		"apiAddress": client.runTimeAPIURL, "agentIdentifier": agentIdentifier,
	}).Info("/event/next to RuntimeAPIClient on ")

	request, error := http.NewRequest("GET", client.runTimeAPIURL+"/event/next", nil)
	if error != nil {
		log.Fatalln(error)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(lambdaAgentIdentifierHeaderKey, agentIdentifier)

	//timeout := time.Duration(5 * time.Second)
	httpClient := http.Client{}

	response, error := httpClient.Do(request)
	if error != nil {
		log.Fatalln(error)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.WithFields(log.Fields{"Status": response.Status, "Body": response.Body}).Error("/event/next request to RuntimeAPIClient failed. ")
	}
	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
		log.Fatalln(error)
	}
	log.WithFields(log.Fields{
		"data": string(body),
	}).Info("Received response from RuntimeAPIClient: ")
	return string(body)
}
