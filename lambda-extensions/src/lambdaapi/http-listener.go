package lambdaapi

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
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
	logger    *logrus.Entry
}

// NewHTTPServer is to return a new object
func NewHTTPServer(consumerQueue chan []byte, logger *logrus.Entry) *HTTPServer {
	return &HTTPServer{dataQueue: consumerQueue, logger: logger}
}

// HTTPServerStart is to start the HTTP Server
func (httpServer *HTTPServer) HTTPServerStart() {
	http.HandleFunc("/", httpServer.LogsHandler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", ReceiverIP, ReceiverPort), nil)
	if err != nil {
		panic(err)
	}
}

// LogsHandler is Server Implementation to get Logs from logs API.
func (httpServer *HTTPServer) LogsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case "POST":
		reqBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			panic(err)
		}
		httpServer.logger.Debug("Producing data into dataQueue")
		httpServer.dataQueue <- []byte(reqBody)
	}
}
