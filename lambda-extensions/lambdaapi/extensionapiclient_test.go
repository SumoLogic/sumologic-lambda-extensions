package lambdaapi

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterExtension(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error in request")
		defer r.Body.Close()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(RegisterResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.RegisterExtension(nil)

	// With Context
	response, err = client.RegisterExtension(context.Background())
	asserts(t, client, response, err)
}

func TestNextEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodGet, "Method is not GET")
		assertNotEmpty(t, r.Header.Get(extensionIdentiferHeader), "Extension ID Header not present")
		defer r.Body.Close()

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(NextEventResponse{
			EventType:          Invoke,
			DeadlineMs:         1234,
			RequestID:          "5678",
			InvokedFunctionArn: "arn:aws:test",
		})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.NextEvent(nil)
	asserts(t, client, response, err)

	// With Context
	response, err = client.NextEvent(context.Background())
	asserts(t, client, response, err)
}

func TestInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")
		assertNotEmpty(t, r.Header.Get(extensionErrorType), "Extension Error Header not present")
		assertEqual(t, r.Header.Get(extensionErrorType), "INIT ERROR", "Extension Error did not match")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error in request")
		defer r.Body.Close()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(StatusResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.InitError(nil, "INIT ERROR")
	asserts(t, client, response, err)

	// With Context
	response, err = client.InitError(context.Background(), "INIT ERROR")
	asserts(t, client, response, err)
}

func TestExitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")
		assertNotEmpty(t, r.Header.Get(extensionErrorType), "Extension Error Header not present")
		assertEqual(t, r.Header.Get(extensionErrorType), "EXIT ERROR", "Extension Error did not match")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error in request")
		defer r.Body.Close()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(StatusResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.ExitError(nil, "EXIT ERROR")
	asserts(t, client, response, err)

	// With Context
	response, err = client.ExitError(context.Background(), "EXIT ERROR")
	asserts(t, client, response, err)
}
