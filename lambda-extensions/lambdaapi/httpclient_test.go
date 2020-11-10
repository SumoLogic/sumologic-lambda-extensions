package lambdaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	lambdaRuntimeAPI = "127.0.0.1:8123"
	extensionName    = "sumologic-extension"
)

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Error(message)
}

func assertNotEmpty(t *testing.T, a interface{}, message string) {
	if a != nil {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v is nil", a)
	}
	t.Error(message)
}

func assertNoError(t *testing.T, a interface{}, message string) {
	if a == nil {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v has error", a)
	}
	t.Error(message)
}

func commonAsserts(t *testing.T, client *Client, response interface{}, err error) {
	assertNoError(t, err, "Make request has Error.")
	assertNotEmpty(t, response, "no output received.")
	assertEqual(t, "test-sumo-id", client.extensionID, "Extension ID does not match.")
}

func TestNewClient(t *testing.T) {
	client := NewClient(lambdaRuntimeAPI, extensionName)
	assertEqual(t, client.baseURL, "http://127.0.0.1:8123/", "Base URL does not match the expected URL")
	assertEqual(t, client.extensionName, extensionName, "Extension Name does not match the expected name")
}

func createTestClient(t *testing.T) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodGet, "Method is not GET")
		defer r.Body.Close()

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(NextEventResponse{})
		_, _ = w.Write(respBytes)
	}))
	defer srv.Close()
	return NewClient(srv.URL[7:], extensionName)
}

func TestMakeRequest(t *testing.T) {
	client := createTestClient(t)

	URL := client.baseURL + extensionURL + "event/next"
	headers := map[string]string{
		extensionNameHeader: client.extensionName,
	}

	response, err := client.MakeRequest(headers, bytes.NewBuffer(nil), "GET", URL)
	commonAsserts(t, client, response, err)
}

func TestMakeRequestWithContext(t *testing.T) {
	client := createTestClient(t)

	URL := client.baseURL + extensionURL + "event/next"
	headers := map[string]string{
		extensionNameHeader: client.extensionName,
	}

	response, err := client.MakeRequestWithContext(context.Background(), headers, bytes.NewBuffer(nil), "GET", URL)
	commonAsserts(t, client, response, err)
}
