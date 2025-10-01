package lambdaapi

import (
	"bytes"
	"context"
	"fmt"
	ioutil "io"
	"log"
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

// MakeRequestWithContext is to hit the URL and get the response with the content, method, headers, URL and request provided.
func (client *Client) MakeRequestWithContext(ctx context.Context, headers map[string]string, request *bytes.Buffer, methodType, URL string) ([]byte, error) {
	// Creating an HTTP Request with Context.
	httpReq, err := http.NewRequestWithContext(ctx, methodType, URL, request)
	if err != nil {
		return nil, err
	}
	// Setting all the headers passed
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	// Calling the URL
	httpRes, err := client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := httpRes.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()
	body, err := ioutil.ReadAll(httpRes.Body)
	if err != nil {
		return nil, err
	}
	if httpRes.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status %s and response %s", httpRes.Status, string(body))
	}
	// Get the Extension ID from the headers
	id := httpRes.Header.Get(extensionIdentiferHeader)
	if len(id) != 0 {
		client.extensionID = id
	}
	return body, nil
}

// MakeRequest is to hit the URL and get the response with the content, method, headers, URL and request provided.
func (client *Client) MakeRequest(headers map[string]string, request *bytes.Buffer, methodType, URL string) ([]byte, error) {
	// Creating an HTTP Request with Context.
	httpReq, err := http.NewRequest(methodType, URL, request)
	if err != nil {
		return nil, err
	}
	// Setting all the headers passed
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	// Calling the URL
	httpRes, err := client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := httpRes.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()
	body, err := ioutil.ReadAll(httpRes.Body)
	if err != nil {
		return nil, err
	}
	if httpRes.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status %s and response %s", httpRes.Status, string(body))
	}
	// Get the Extension ID from the headers
	id := httpRes.Header.Get(extensionIdentiferHeader)
	if len(id) != 0 {
		client.extensionID = id
	}
	return body, nil
}
