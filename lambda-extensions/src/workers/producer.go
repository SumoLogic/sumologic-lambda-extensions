package workers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
	quitQueue chan bool
}

// NewTaskProducer is to return a new object
func NewTaskProducer(consumerQueue chan []byte, quitQueue chan bool, logger *logrus.Entry) TaskProducer {
	return &httpServer{dataQueue: consumerQueue, logger: logger, quitQueue: quitQueue}
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
		payload := []byte(reqBody)
		// Sends to a buffered channel block only when the buffer is full
		httpServer.dataQueue <- payload
		//httpServer.checkInvokeEnd(payload)
	}
}

func (httpServer *httpServer) checkInvokeEnd(payload []byte) {
	data := string(payload)
	const eventType = `"type":"platform.end"`
	if strings.Contains(data, eventType) {
		httpServer.quitQueue <- true
	}
}
