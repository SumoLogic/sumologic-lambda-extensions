package lambdaapi

import (
	"context"
	ioutil "io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubscribeToLogsAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPut, "Method is not PUT")
		assertNotEmpty(t, r.Header.Get(extensionNameHeader), "Extension Name Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertNoError(t, err, "Received error")
		defer r.Body.Close()
		assertNotEmpty(t, reqBytes, "Received error in request")

		w.Header().Add(extensionIdentiferHeader, "test-sumo-id")
		w.WriteHeader(200)
	}))

	defer srv.Close()
	client := NewClient(srv.URL[7:], extensionName)

	// Without Context
	response, err := client.SubscribeToLogsAPI(nil, []string{"platform", "function", "extension"})
	commonAsserts(t, client, response, err)

	// With Context
	response, err = client.SubscribeToLogsAPI(context.Background(), []string{"platform", "function", "extension"})
	commonAsserts(t, client, response, err)
}
