package workers

import (
	"encoding/json"
	"fmt"
	ioutil "io"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	// managedReceiverIP is Web Server Constants for managed instance mode
	managedReceiverIP = "0.0.0.0"
	// managedReceiverPort is Web Server Constants for managed instance mode
	managedReceiverPort = 4243
	// queueThresholdPercent is the threshold percentage for triggering flush
	queueThresholdPercent = 0.8
)

// ManagedInstanceTaskProducer exposes methods for producing tasks in managed instance mode
type ManagedInstanceTaskProducer interface {
	Start() error
}

type managedInstanceHttpServer struct {
	dataQueue   chan []byte
	logger      *logrus.Entry
	flushSignal chan string // Signal channel to notify consumer to flush
}

type Event struct {
	Time   string          `json:"time"`
	Type   string          `json:"type"`
	Record json.RawMessage `json:"record"`
}

// NewManagedInstanceTaskProducer returns a new managed instance producer object
// flushSignal channel is used to signal consumer when queue is 80% full or platform.report is received
func NewManagedInstanceTaskProducer(consumerQueue chan []byte, flushSignal chan string, logger *logrus.Entry) ManagedInstanceTaskProducer {
	return &managedInstanceHttpServer{
		dataQueue:   consumerQueue,
		logger:      logger,
		flushSignal: flushSignal,
	}
}

// Start starts the HTTP Server for managed instance mode
func (mhs *managedInstanceHttpServer) Start() error {
	http.HandleFunc("/", mhs.logsHandler)
	mhs.logger.Info("Starting Managed Instance HTTP Server on port ", managedReceiverPort)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", managedReceiverIP, managedReceiverPort), nil)
	if err != nil {
		mhs.logger.Errorf("Managed Instance HTTP server failed to start: %v", err)
		panic(err)
	}
	return err
}

// checkQueueThreshold checks if dataQueue has reached 80% capacity and signals consumer
func (mhs *managedInstanceHttpServer) checkQueueThreshold() {
	queueLen := len(mhs.dataQueue)
	queueCap := cap(mhs.dataQueue)
	threshold := int(float64(queueCap) * queueThresholdPercent)

	mhs.logger.Debugf("Managed Instance Producer: Queue status - Length: %d, Capacity: %d, Threshold: %d", queueLen, queueCap, threshold)

	if queueLen >= threshold {
		mhs.logger.Infof("Managed Instance Producer: Queue reached %d%% capacity (%d/%d), signaling consumer to flush",
			int(queueThresholdPercent*100), queueLen, queueCap)
		// Send flush signal to consumer (non-blocking)
		select {
		case mhs.flushSignal <- "queue_threshold":
			mhs.logger.Debugf("Managed Instance Producer: Sent queue_threshold signal to consumer")
		default:
			mhs.logger.Warnf("Managed Instance Producer: Flush signal channel full, signal dropped")
		}
	}
}

// logsHandler is Server Implementation to get Logs from logs API for managed instance mode
func (mhs *managedInstanceHttpServer) logsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case "POST":
		defer func() {
			if err := request.Body.Close(); err != nil {
				mhs.logger.Errorf("failed to close body: %v", err)
			}
		}()

		reqBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			mhs.logger.Error("Read from Logs API failed: ", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		mhs.logger.Debugf("Managed Instance Producer: Producing data into dataQueue - %d bytes\n", len(reqBody))
		payload := []byte(reqBody)

		// Send payload to dataQueue (non-blocking to prevent deadlock)
		select {
		case mhs.dataQueue <- payload:
			mhs.logger.Debugf("Managed Instance Producer: Successfully queued data")
		default:
			mhs.logger.Warnf("Managed Instance Producer: dataQueue is full, dropping message")
		}

		// Check if queue has reached 80% capacity after adding data
		mhs.checkQueueThreshold()

		// Parse events and check for platform.report
		var events []Event
		err = json.Unmarshal(reqBody, &events)
		if err != nil {
			mhs.logger.Errorf("Managed Instance Producer: Error parsing JSON: %v", err)
		} else {
			mhs.logger.Debugf("Managed Instance Producer: Parsed %d events from telemetry payload\n", len(events))

			// Check for platform.report type
			for _, event := range events {
				if event.Type == "platform.report" {
					mhs.logger.Infof("Managed Instance Producer: Found platform.report event at time: %s\n", event.Time)
					// Send platform.report signal to consumer (non-blocking)
					select {
					case mhs.flushSignal <- "platform.report":
						mhs.logger.Debugf("Managed Instance Producer: Sent platform.report signal to consumer")
					default:
						mhs.logger.Warnf("Managed Instance Producer: Flush signal channel full, signal dropped")
					}
				}
			}
		}

		writer.WriteHeader(http.StatusOK)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
