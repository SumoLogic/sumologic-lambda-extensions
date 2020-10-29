package lambdaapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	// ReceiverIP is Web Server Constants
	ReceiverIP = "0.0.0.0"
	// ReceiverPort is Web Server Constants
	ReceiverPort = 4243
)

// HTTPServer is a struct with list
type HTTPServer struct {
	dataQueue chan []byte
}

// NewHTTPServer is to return a new object
func NewHTTPServer(consumerQueue chan []byte) *HTTPServer {
	return &HTTPServer{dataQueue: consumerQueue}
}

// HTTPServerStart is to start the HTTP Server
func (httpServer *HTTPServer) HTTPServerStart() {
	http.HandleFunc("/", httpServer.LogsHandler)
	http.ListenAndServe(fmt.Sprintf("%s:%d", ReceiverIP, ReceiverPort), nil)
}

// LogsHandler is Server Implementation to get Logs from logs API.
func (httpServer *HTTPServer) LogsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case "POST":
		reqBody, error := ioutil.ReadAll(request.Body)
		if error != nil {
			panic(error)
		}
		fmt.Println("Producing data into dataQueue")
		httpServer.dataQueue <- []byte(reqBody)
	}
}
