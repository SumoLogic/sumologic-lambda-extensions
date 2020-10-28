package lambdaapi

import (
	"context"
	"encoding/json"
)

// RegisterResponse is the body of the response for /register
type RegisterResponse struct {
	FunctionName    string `json:"functionName"`
	FunctionVersion string `json:"functionVersion"`
	Handler         string `json:"handler"`
}

// NextEventResponse is the response for /event/next
type NextEventResponse struct {
	EventType          EventType `json:"eventType"`
	DeadlineMs         int64     `json:"deadlineMs"`
	RequestID          string    `json:"requestId"`
	InvokedFunctionArn string    `json:"invokedFunctionArn"`
	Tracing            Tracing   `json:"tracing"`
}

// Tracing is part of the response for /event/next
type Tracing struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// StatusResponse is the body of the response for /init/error and /exit/error
type StatusResponse struct {
	Status string `json:"status"`
}

// EventType represents the type of events recieved from /event/next
type EventType string

const (
	// Invoke is a lambda invoke
	Invoke EventType = "INVOKE"
	// Shutdown is a shutdown event for the environment
	Shutdown EventType = "SHUTDOWN"
	// Base URL for extension
	extensionURL = "2020-01-01/extension/"
)

var (
	lambdaEvents = []EventType{"INVOKE", "SHUTDOWN"}
)

// RegisterExtension is to regsiter extension to Run Time API client. Call the following method on initialization as early as possible,
// otherwise you may get a timeout error. Runtime initialization will start after all extensions are registered.
func (client *Client) RegisterExtension(ctx context.Context) (*RegisterResponse, error) {
	URL := client.baseURL + extensionURL + "register"
	reqBody, error := json.Marshal(map[string]interface{}{
		"events": lambdaEvents,
	})
	if error != nil {
		return nil, error
	}
	headers := map[string]string{
		extensionNameHeader: client.extensionName,
	}
	response, error := client.MakeRequest(ctx, headers, reqBody, "POST", URL)
	if error != nil {
		return nil, error
	}
	registerResponse := RegisterResponse{}
	error = json.Unmarshal(response, &registerResponse)
	if error != nil {
		return nil, error
	}
	return &registerResponse, nil
}

// NextEvent is - Call the following method when the extension is ready to receive the next invocation
// and there is no job it needs to execute beforehand. blocks while long polling for the next lambda invoke or shutdown
func (client *Client) NextEvent(ctx context.Context) (*NextEventResponse, error) {
	URL := client.baseURL + extensionURL + "event/next"

	headers := map[string]string{
		extensionIdentiferHeader: client.extensionID,
	}
	response, error := client.MakeRequest(ctx, headers, nil, "GET", URL)
	if error != nil {
		return nil, error
	}
	nextEventResponse := NextEventResponse{}
	error = json.Unmarshal(response, &nextEventResponse)
	if error != nil {
		return nil, error
	}
	return &nextEventResponse, nil
}

// InitError reports an initialization error to the platform. Call it when you registered but failed to initialize
func (client *Client) InitError(ctx context.Context, errorType string) (*StatusResponse, error) {
	URL := client.baseURL + extensionURL + "/init/error"

	headers := map[string]string{
		extensionIdentiferHeader: client.extensionID,
		extensionErrorType:       errorType,
	}
	response, error := client.MakeRequest(ctx, headers, nil, "POST", URL)
	if error != nil {
		return nil, error
	}
	statusResponse := StatusResponse{}
	error = json.Unmarshal(response, &statusResponse)
	if error != nil {
		return nil, error
	}
	return &statusResponse, nil
}

// ExitError reports an error to the platform before exiting. Call it when you encounter an unexpected failure
func (client *Client) ExitError(ctx context.Context, errorType string) (*StatusResponse, error) {
	URL := client.baseURL + extensionURL + "/exit/error"

	headers := map[string]string{
		extensionIdentiferHeader: client.extensionID,
		extensionErrorType:       errorType,
	}
	response, error := client.MakeRequest(ctx, headers, nil, "POST", URL)
	if error != nil {
		return nil, error
	}
	statusResponse := StatusResponse{}
	error = json.Unmarshal(response, &statusResponse)
	if error != nil {
		return nil, error
	}
	return &statusResponse, nil
}
