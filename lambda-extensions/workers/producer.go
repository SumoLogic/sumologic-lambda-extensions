package workers

import (
	"fmt"
	ioutil "io"
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
		defer func() {
			if err := request.Body.Close(); err != nil {
				httpServer.logger.Errorf("failed to close body: %v", err)
			}
		}()
		reqBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			// TODO: raise alert if read fails
			httpServer.logger.Error("Read from Logs API failed: ", err.Error())
		}

		httpServer.logger.Debugf("Producing data into dataQueue - %d \n", len(reqBody))
		payload := []byte(reqBody)
		// Sends to a buffered channel block only when the buffer is full
		httpServer.dataQueue <- payload
		writer.WriteHeader(http.StatusOK)
	}
}
