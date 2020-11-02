package sumoclient

import (
	"bytes"
	"config"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"utils"

	uuid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var isColdStart = true

// LogSender interface which needs to be implemented to send logs
type LogSender interface {
	SendLogs(context.Context, []byte) error
	FlushAll([][]byte) error
}

// sumoLogicClient implements LogSender interface
type sumoLogicClient struct {
	httpClient http.Client
	config     *config.LambdaExtensionConfig
	logger     *logrus.Entry
}

// It is assumed that logs will be array of json objects and all channel payloads satisfy this format
type responseBody []map[string]interface{}

// NewLogSenderClient returns interface pointing to the concrete version of LogSender client
func NewLogSenderClient(logger *logrus.Entry, cfg *config.LambdaExtensionConfig) LogSender {
	// setting the cold start variable here since this function is called
	var logSenderClient LogSender = &sumoLogicClient{
		httpClient: http.Client{Timeout: cfg.ConnectionTimeoutValue},
		config:     cfg,
		logger:     logger,
	}
	return logSenderClient
}

func (s *sumoLogicClient) getColdStart() bool {
	if isColdStart {
		isColdStart = false
	}
	return isColdStart
}

func (s *sumoLogicClient) makeRequest(ctx context.Context, buf *bytes.Buffer) (*http.Response, error) {

	request, err := http.NewRequestWithContext(ctx, "POST", s.config.SumoHTTPEndpoint, buf)
	if err != nil {
		s.logger.Errorf("http.NewRequest() error: %v\n", err)
		return nil, err
	}
	request.Header.Add("Content-Encoding", "gzip")
	request.Header.Add("X-Sumo-Client", "sumologic-lambda-extension")
	// if s.config.sumoName != "" {
	//     request.Header.Add("X-Sumo-Name", s.config.sumoName)
	// }
	// if s.config.sumoHost != "" {
	//     request.Header.Add("X-Sumo-Host", s.config.sumoHost)
	// }
	// if s.config.sumoCategory != "" {
	//     request.Header.Add("X-Sumo-Category", s.config.sumoCategory)
	// }
	response, err := s.httpClient.Do(request)
	return response, err
}

// getS3KeyName returns the key by combining function name, version, date and uuid(version 1)
func (s *sumoLogicClient) getS3KeyName() (string, error) {
	currentTime := time.Now()
	uniqueID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s/%s/%d/%02d/%02d/%02d/%d/%v.gz", s.config.FunctionName, s.config.FunctionVersion,
		currentTime.Year(), currentTime.Month(), currentTime.Day(),
		currentTime.Hour(), currentTime.Minute(), uniqueID)

	return key, nil
}

func (s *sumoLogicClient) failoverHandler(buf *bytes.Buffer) error {
	var err error
	if s.config.EnableFailover {

		s.logger.Debug("Trying to Send to S3")
		keyName, err := s.getS3KeyName()
		if err != nil {
			return err
		}
		err = utils.UploadToS3(&s.config.S3BucketName, &keyName, buf)
		if err != nil {
			err = fmt.Errorf("Failed to Send to S3 Bucket %s Path %s: %w", s.config.S3BucketName, keyName, err)
		}
	} else {
		s.logger.Debugln("FailOver is not enabled.")
	}
	return err
}

func (s *sumoLogicClient) FlushAll(msgQueue [][]byte) error {
	var err error

	if len(msgQueue) > 0 && s.config.EnableFailover {
		s.logger.Debugf("Attempting to send %d payloads from dataqueue to S3", len(msgQueue))
		var errorCount int = 0
		var totalitems int = 0
		var payload bytes.Buffer
		for _, rawmsg := range msgQueue {
			// converting to arr of maps
			msgArr, err := s.transformBytesToArrayOfMap(rawmsg)
			if err != nil {
				s.logger.Error(err.Error())
				errorCount++
				continue
			}
			if len(msgArr) > 0 {
				// enhancing logs
				s.enhanceLogs(msgArr)
				totalitems += len(msgArr)

				// converting back to string
				for _, item := range msgArr {
					b, err := json.Marshal(item)
					if err != nil {
						s.logger.Error("Error in coverting to json: ", err.Error())
						errorCount++
						continue
					}
					payload.WriteString(fmt.Sprintf("\n%s", string(b)))
				}
			}
		}
		s.logger.Debugf("Total log lines transformed: %d", totalitems)

		// compressing and pushing to S3
		gzippedBuffer := utils.CompressBuffer(&payload)
		s.failoverHandler(gzippedBuffer)

		if errorCount > 0 {
			err = fmt.Errorf("Total errors during flushing to S3: %d", errorCount)
		}
	} else {
		s.logger.Debugln("FailOver is not enabled.")
	}
	return err
}

func (s *sumoLogicClient) enhanceLogs(msg responseBody) {
	s.logger.Debugln("Enhancing logs")
	for _, item := range msg {
		item["FunctionName"] = s.config.FunctionName
		item["FunctionVersion"] = s.config.FunctionVersion
		item["IsColdStart"] = s.getColdStart()
	}
}

func (s *sumoLogicClient) transformBytesToArrayOfMap(rawmsg []byte) (responseBody, error) {
	s.logger.Debugln("Transforming bytes to array of maps")
	var msg responseBody
	var err error
	err = json.Unmarshal(rawmsg, &msg)
	if err != nil {
		return msg, fmt.Errorf("Error in parsing payload %s: %v", string(rawmsg), err)
	}
	return msg, err
}

func (s *sumoLogicClient) createChunks(msgArr responseBody) ([]string, error) {

	var err error
	var chunks []string
	var itemSize int
	var chunkSize int = 0
	var currentChunk bytes.Buffer
	var errorCount int = 0
	for _, item := range msgArr {
		b, err := json.Marshal(item)
		if err != nil {
			s.logger.Error("Error in coverting to json: ", err.Error())
			errorCount++
			continue
		}
		itemSize = binary.Size(b)
		if chunkSize+itemSize+1 >= s.config.MaxDataPayloadSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk = *bytes.NewBufferString(string(b))
			chunkSize = itemSize
		} else {
			chunkSize += itemSize + 1
			currentChunk.WriteString(fmt.Sprintf("\n%s", string(b)))
		}

	}
	chunks = append(chunks, currentChunk.String())
	if errorCount > 0 {
		err = fmt.Errorf("Dropping %d messages due to json parsing error", errorCount)
	}
	s.logger.Debugf("Chunks created: %d NumOfParsingError: %d", len(chunks), errorCount)
	return chunks, err
}

// SendToSumo send logs to sumo http endpoint returns
func (s *sumoLogicClient) SendLogs(ctx context.Context, rawmsg []byte) error {
	var err error
	if len(rawmsg) > 0 {
		// converting to arr of maps
		msgArr, err := s.transformBytesToArrayOfMap(rawmsg)
		if err != nil {
			return err
		}
		s.logger.Debugf("Total log lines transformed: %d", len(msgArr))
		s.enhanceLogs(msgArr)

		// converting back to chunks of string
		chunks, _ := s.createChunks(msgArr)
		for _, strobj := range chunks {
			s.postToSumo(ctx, &strobj)
		}
	}
	return err
}

func (s *sumoLogicClient) postToSumo(ctx context.Context, logStringToSend *string) error {
	s.logger.Debug("Attempting to send to Sumo Endpoint")

	// compressing here because Sumo recommends payload size of 1MB before compression
	bytedata := utils.Compress(logStringToSend)
	createBuffer := func() *bytes.Buffer {
		dest := make([]byte, len(bytedata))
		copy(dest, bytedata)
		return bytes.NewBuffer(dest)
	}
	buf := createBuffer()
	response, err := s.makeRequest(ctx, buf)

	if (err != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
		s.logger.Errorf("Not able to post statuscode:  %v %v\n", err, response)
		s.logger.Debugf("Waiting for %v ms to retry\n", s.config.RetrySleepTime)
		time.Sleep(s.config.RetrySleepTime)
		err := utils.Retry(func(attempt int) (bool, error) {
			var errRetry error
			buf := createBuffer()
			response, errRetry = s.makeRequest(ctx, buf)
			if (errRetry != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
				if errRetry == nil {
					errRetry = fmt.Errorf("statuscode %v", response.StatusCode)
				}
				s.logger.Error("Not able to post: ", errRetry)
				s.logger.Debugf("Waiting for %v ms to retry attempts done: %v\n", s.config.RetrySleepTime, attempt)
				time.Sleep(s.config.RetrySleepTime)
				return attempt < s.config.MaxRetryAttempts, errRetry
			} else if response.StatusCode == 200 {
				s.logger.Debugf("Post of logs successful after retry %v attempts\n", attempt)
				return true, nil
			}
			return attempt < s.config.MaxRetryAttempts, errRetry
		}, s.config.NumRetry)
		if err != nil {
			s.logger.Error("Finished retrying Error: ", err)
			buf = createBuffer()
			return s.failoverHandler(buf) // sending uncompressed logs to S3 so customer can easily view it
		}
	} else if response.StatusCode == 200 {
		s.logger.Debugf("Post of logs successful")
	}
	if response != nil {
		defer response.Body.Close()
	}

	return nil
}
