package lambdaapi

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	extensionNameHeader      = "Lambda-Extension-Name"
	extensionIdentiferHeader = "Lambda-Extension-Identifier"
	extensionErrorType       = "Lambda-Extension-Function-Error-Type"
)

// Client is a simple client for the Lambda Extensions API
type Client struct {
	baseURL       string
	httpClient    *http.Client
	extensionID   string
	extensionName string
}

// NewClient returns a Lambda Extensions API client
func NewClient(awsLambdaRuntimeAPI, extensionName string) *Client {
	baseURL := fmt.Sprintf("http://%s/", awsLambdaRuntimeAPI)
	return &Client{
		baseURL:       baseURL,
		httpClient:    &http.Client{},
		extensionName: extensionName,
	}
}

// MakeRequest is to hit the URL and get the response with the content, method, headers, URL and request provided.
func (client *Client) MakeRequest(ctx context.Context, headers map[string]string, request []byte, methodType, URL string) ([]byte, error) {
	// Creating an HTTP Request with Context.
	httpReq, error := http.NewRequestWithContext(ctx, methodType, URL, bytes.NewBuffer(request))
	if error != nil {
		return nil, error
	}
	// Setting all the headers passed
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	// Calling the URL
	httpRes, error := client.httpClient.Do(httpReq)
	if error != nil {
		return nil, error
	}
	defer httpRes.Body.Close()
	body, error := ioutil.ReadAll(httpRes.Body)
	if error != nil {
		return nil, error
	}
	if httpRes.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed with status %s and response %s", httpRes.Status, string(body))
	}
	// Get the Extension ID from the headers
	id := httpRes.Header.Get(extensionIdentiferHeader)
	if len(id) != 0 {
		client.extensionID = id
	}
	return body, nil
}
