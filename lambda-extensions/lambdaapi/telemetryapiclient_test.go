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

func TestSubscribeToTelemetryAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPut, "Method is not PUT")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.SubscribeToTelemetryAPI(context.TODO(), []string{"platform", "function", "extension"}, 1000, 262144, 10000, false)
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.SubscribeToTelemetryAPI(context.Background(), []string{"platform", "function", "extension"}, 1000, 262144, 10000, false)
	commonAsserts(t, client, response, err)
}

// TestSubscribeToTelemetryAPI_ManagedInstanceMode tests telemetry API subscription in managed instance mode
// In managed instance mode, schema version should be "2025-01-29" instead of "2022-07-01"
func TestSubscribeToTelemetryAPI_ManagedInstanceMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPut, "Method is not PUT")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error")
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("failed to close body: %v", err)
			}
		}()
		assertNotEmpty(t, reqBytes, "Received error in request")

		// Verify the request body contains managed instance mode schema version
		var reqBody map[string]interface{}
		err = json.Unmarshal(reqBytes, &reqBody)
		assertNoError(t, err, "Failed to unmarshal request body")

		schemaVersion, ok := reqBody["schemaVersion"].(string)
		if !ok {
			t.Error("schemaVersion field not found or not a string")
		}
		assertEqual(t, schemaVersion, "2025-01-29", "Expected managed instance mode schema version '2025-01-29'")

		// Verify other required fields are present
		_, destinationExists := reqBody["destination"]
		if !destinationExists {
			t.Error("destination field not found")
		}

		_, typesExists := reqBody["types"]
		if !typesExists {
			t.Error("types field not found")
		}

		_, bufferingExists := reqBody["buffering"]
		if !bufferingExists {
			t.Error("buffering field not found")
		}

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Test with isManagedInstance = true (context)
	response, err := client.SubscribeToTelemetryAPI(context.Background(), []string{"platform", "function", "extension"}, 1000, 262144, 10000, true)
	commonAsserts(t, client, response, err)

	// Test with isManagedInstance = true (without context)
	response, err = client.SubscribeToTelemetryAPI(context.TODO(), []string{"platform", "function", "extension"}, 1000, 262144, 10000, true)
	commonAsserts(t, client, response, err)
}
