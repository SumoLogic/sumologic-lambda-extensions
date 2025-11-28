package workers

import (
	"encoding/json"
	"fmt"
	ioutil "io"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	// elevatorReceiverIP is Web Server Constants for elevator mode
	elevatorReceiverIP = "0.0.0.0"
	// elevatorReceiverPort is Web Server Constants for elevator mode
	elevatorReceiverPort = 4243
	// queueThresholdPercent is the threshold percentage for triggering flush
	queueThresholdPercent = 0.8
)

// ElevatorTaskProducer exposes methods for producing tasks in elevator mode
type ElevatorTaskProducer interface {
	Start() error
}

type elevatorHttpServer struct {
	dataQueue   chan []byte
	logger      *logrus.Entry
	flushSignal chan string // Signal channel to notify consumer to flush
}

type Event struct {
	Time   string          `json:"time"`
	Type   string          `json:"type"`
	Record json.RawMessage `json:"record"`
}

// NewElevatorTaskProducer returns a new elevator producer object
// flushSignal channel is used to signal consumer when queue is 80% full or platform.report is received
func NewElevatorTaskProducer(consumerQueue chan []byte, flushSignal chan string, logger *logrus.Entry) ElevatorTaskProducer {
	return &elevatorHttpServer{
		dataQueue:   consumerQueue,
		logger:      logger,
		flushSignal: flushSignal,
	}
}

// Start starts the HTTP Server for elevator mode
func (ehs *elevatorHttpServer) Start() error {
	http.HandleFunc("/", ehs.logsHandler)
	ehs.logger.Info("Starting Elevator HTTP Server on port ", elevatorReceiverPort)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", elevatorReceiverIP, elevatorReceiverPort), nil)
	if err != nil {
		ehs.logger.Errorf("Elevator HTTP server failed to start: %v", err)
		panic(err)
	}
	return err
}

// checkQueueThreshold checks if dataQueue has reached 80% capacity and signals consumer
func (ehs *elevatorHttpServer) checkQueueThreshold() {
	queueLen := len(ehs.dataQueue)
	queueCap := cap(ehs.dataQueue)
	threshold := int(float64(queueCap) * queueThresholdPercent)

	ehs.logger.Debugf("Elevator Producer: Queue status - Length: %d, Capacity: %d, Threshold: %d", queueLen, queueCap, threshold)

	if queueLen >= threshold {
		ehs.logger.Infof("Elevator Producer: Queue reached %d%% capacity (%d/%d), signaling consumer to flush",
			int(queueThresholdPercent*100), queueLen, queueCap)
		// Send flush signal to consumer (non-blocking)
		select {
		case ehs.flushSignal <- "queue_threshold":
			ehs.logger.Debugf("Elevator Producer: Sent queue_threshold signal to consumer")
		default:
			ehs.logger.Warnf("Elevator Producer: Flush signal channel full, signal dropped")
		}
	}
}

// logsHandler is Server Implementation to get Logs from logs API for elevator mode
func (ehs *elevatorHttpServer) logsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case "POST":
		defer func() {
			if err := request.Body.Close(); err != nil {
				ehs.logger.Errorf("failed to close body: %v", err)
			}
		}()

		reqBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			ehs.logger.Error("Read from Logs API failed: ", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		ehs.logger.Debugf("Elevator Producer: Producing data into dataQueue - %d bytes\n", len(reqBody))
		payload := []byte(reqBody)

		// Send payload to dataQueue (non-blocking to prevent deadlock)
		select {
		case ehs.dataQueue <- payload:
			ehs.logger.Debugf("Elevator Producer: Successfully queued data")
		default:
			ehs.logger.Warnf("Elevator Producer: dataQueue is full, dropping message")
		}

		// Check if queue has reached 80% capacity after adding data
		ehs.checkQueueThreshold()

		// Parse events and check for platform.report
		var events []Event
		err = json.Unmarshal(reqBody, &events)
		if err != nil {
			log.Printf("Elevator Producer: Error parsing JSON: %v", err)
		} else {
			ehs.logger.Debugf("Elevator Producer: Parsed %d events from telemetry payload\n", len(events))

			// Check for platform.report type
			for _, event := range events {
				if event.Type == "platform.report" {
					ehs.logger.Infof("Elevator Producer: Found platform.report event at time: %s\n", event.Time)
					// Send platform.report signal to consumer (non-blocking)
					select {
					case ehs.flushSignal <- "platform.report":
						ehs.logger.Debugf("Elevator Producer: Sent platform.report signal to consumer")
					default:
						ehs.logger.Warnf("Elevator Producer: Flush signal channel full, signal dropped")
					}
				}
			}
		}

		writer.WriteHeader(http.StatusOK)
	default:
		http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
