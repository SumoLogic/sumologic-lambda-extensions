package lambdaapi

import (
	"context"
	"encoding/json"
	ioutil "io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterExtension(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error while reading request")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(RegisterResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.RegisterExtension(context.TODO(), false)
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.RegisterExtension(context.Background(), false)
	commonAsserts(t, client, response, err)
}

func TestNextEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodGet, "Method is not GET")
		assertNotEmpty(t, r.Header.Get(extensionIdentiferHeader), "Extension ID Header not present")

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()

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
	response, err := client.NextEvent(context.TODO())
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.NextEvent(context.Background())
	commonAsserts(t, client, response, err)
}

func TestInitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")
		assertNotEmpty(t, r.Header.Get(extensionErrorType), "Extension Error Header not present")
		assertEqual(t, r.Header.Get(extensionErrorType), "INIT ERROR", "Extension Error did not match")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error in request")

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()

		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(StatusResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.InitError(context.TODO(), "INIT ERROR")
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.InitError(context.Background(), "INIT ERROR")
	commonAsserts(t, client, response, err)
}

func TestExitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")
		assertNotEmpty(t, r.Header.Get(extensionErrorType), "Extension Error Header not present")
		assertEqual(t, r.Header.Get(extensionErrorType), "EXIT ERROR", "Extension Error did not match")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error in request")

		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()

		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(StatusResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.ExitError(context.TODO(), "EXIT ERROR")
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.ExitError(context.Background(), "EXIT ERROR")
	commonAsserts(t, client, response, err)
}

// TestRegisterExtension_ManagedInstanceMode tests extension registration in managed instance mode
// In ManagedInstance mode, only SHUTDOWN events are registered (not INVOKE)
func TestRegisterExtension_ManagedInstanceMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error while reading request")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()
		assertNotEmpty(t, reqBytes, "Received error in request")

		// Verify the request body contains only SHUTDOWN event for managed instance mode
		var reqBody map[string]interface{}
		err = json.Unmarshal(reqBytes, &reqBody)
		assertNoError(t, err, "Failed to unmarshal request body")

		events, ok := reqBody["events"].([]interface{})
		if !ok {
			t.Error("Events field not found or not an array")
		}
		assertEqual(t, len(events), 1, "Expected 1 event for managed instance mode")
		assertEqual(t, events[0], "SHUTDOWN", "Expected only SHUTDOWN event for managed instance mode")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(RegisterResponse{
			FunctionName:    "test-function",
			FunctionVersion: "$LATEST",
			Handler:         "index.handler",
		})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Test with isManagedInstance = true
	response, err := client.RegisterExtension(context.Background(), true)
	commonAsserts(t, client, response, err)

	// Verify the response is properly unmarshaled
	if response.FunctionName != "test-function" {
		t.Errorf("Expected function name 'test-function', got '%s'", response.FunctionName)
	}
}

// TestRegisterExtension_ManagedInstanceModeWithoutContext tests managed instance mode without context
func TestRegisterExtension_ManagedInstanceModeWithoutContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error while reading request")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()

		var reqBody map[string]interface{}
		err = json.Unmarshal(reqBytes, &reqBody)
		assertNoError(t, err, "Failed to unmarshal request body")

		events, ok := reqBody["events"].([]interface{})
		if !ok {
			t.Error("Events field not found or not an array")
		}
		assertEqual(t, len(events), 1, "Expected 1 event for managed instance mode")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(RegisterResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Test with isManagedInstance = true and nil context
	response, err := client.RegisterExtension(context.TODO(), true)
	commonAsserts(t, client, response, err)
}

// TestRegisterExtension_ManagedInstanceModeEventValidation tests that managed instance mode registers correct events
func TestRegisterExtension_ManagedInstanceModeEventValidation(t *testing.T) {
	receivedEvents := make([]string, 0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error while reading request")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()

		var reqBody map[string]interface{}
		err = json.Unmarshal(reqBytes, &reqBody)
		assertNoError(t, err, "Failed to unmarshal request body")

		events, ok := reqBody["events"].([]interface{})
		if !ok {
			t.Error("Events field not found or not an array")
		}

		for _, e := range events {
			receivedEvents = append(receivedEvents, e.(string))
		}

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
		respBytes, _ := json.Marshal(RegisterResponse{})
		_, _ = w.Write(respBytes)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	_, err := client.RegisterExtension(context.Background(), true)
	assertNoError(t, err, "Failed to register extension in ManagedInstance mode")

	// Validate that INVOKE event is NOT present in managed instance mode
	for _, event := range receivedEvents {
		if event == "INVOKE" {
			t.Error("INVOKE event should not be registered in managed instance mode")
		}
	}

	// Validate that SHUTDOWN event IS present
	foundShutdown := false
	for _, event := range receivedEvents {
		if event == "SHUTDOWN" {
			foundShutdown = true
		}
	}
	if !foundShutdown {
		t.Error("SHUTDOWN event should be registered in managed instance mode")
	}
}
