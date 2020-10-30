package workers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	// receiverIP is Web Server Constants
	receiverIP = "0.0.0.0"
	// receiverPort is Web Server Constants
	receiverPort = 4243
)

// TaskProducer exposes methods for producing tasks
type TaskProducer interface {
	Start() error
}

type httpServer struct {
	dataQueue chan []byte
	logger    *logrus.Entry
}

// NewTaskProducer is to return a new object
func NewTaskProducer(consumerQueue chan []byte, logger *logrus.Entry) TaskProducer {
	return &httpServer{dataQueue: consumerQueue, logger: logger}
}

// Start is to start the HTTP Server
func (httpServer *httpServer) Start() error {
	http.HandleFunc("/", httpServer.logsHandler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", receiverIP, receiverPort), nil)
	if err != nil {
		panic(err)
	}
	return err
}

// logsHandler is Server Implementation to get Logs from logs API.
func (httpServer *httpServer) logsHandler(writer http.ResponseWriter, request *http.Request) {
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
