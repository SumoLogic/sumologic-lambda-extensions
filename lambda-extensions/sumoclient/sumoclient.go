package sumoclient

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/utils"

	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"

	uuid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var isColdStart = true

// LogSender interface which needs to be implemented to send logs
type LogSender interface {
	SendLogs(context.Context, []byte) error
	SendAllLogs(context.Context, [][]byte) error
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
		err = fmt.Errorf("http.NewRequest() error: %v", err)
		return nil, err
	}
	request.Header.Add("Content-Encoding", "gzip")
	request.Header.Add("X-Sumo-Client", config.SumoLogicExtensionLayerVersionSuffix)
	// This is added to make it compatible with AWS Lambda and AWS Lambda ULM App
	request.Header.Add("X-Sumo-Name", s.getLogStream())
	request.Header.Add("X-Sumo-Host", s.getLogGroup())
	if s.config.SourceCategoryOverride != "" {
		request.Header.Add("X-Sumo-Category", s.config.SourceCategoryOverride)
	}
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
	// common prefix where all lambda logs will go

	key := fmt.Sprintf("%s/%s/%s/%s/%d/%02d/%02d/%02d/%d/%v.gz", config.ExtensionName, s.config.LambdaRegion, s.config.FunctionName, s.config.FunctionVersion,
		currentTime.Year(), currentTime.Month(), currentTime.Day(),
		currentTime.Hour(), currentTime.Minute(), uniqueID)

	return key, nil
}

func (s *sumoLogicClient) failoverHandler(buf *bytes.Buffer) error {

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
		return err
	}
	return nil
}

func (s *sumoLogicClient) FlushAll(msgQueue [][]byte) error {
	var err error

	if len(msgQueue) > 0 && s.config.EnableFailover {
		s.logger.Debugf("FlushAll - Attempting to send %d payloads from dataqueue to S3", len(msgQueue))
		var errorCount int = 0
		var totalitems int = 0
		var payload bytes.Buffer
		for _, rawmsg := range msgQueue {
			// converting to arr of maps
			msgArr, err := s.transformBytesToArrayOfMap(rawmsg)
			if err != nil {
				s.logger.Error("FlushAll - Error in transforming bytes to array of struct", err.Error())
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
						s.logger.Error("FlushAll - Error in coverting to json: ", err.Error())
						errorCount++
						continue
					}
					payload.WriteString(fmt.Sprintf("\n%s", string(b)))
				}
			}
		}
		s.logger.Debugf("FlushAll - Total log lines transformed: %d", totalitems)

		// compressing and pushing to S3
		gzippedBuffer := utils.CompressBuffer(&payload)
		senderr := s.failoverHandler(gzippedBuffer)
		if errorCount > 0 || senderr != nil {
			err = fmt.Errorf("FlushAll - Errors during chunk creation: %d, Errors during flushing to S3: %v", errorCount, senderr)
		}
	} else {
		s.logger.Info("FlushAll - Dropping messages as no failover enabled.")
	}
	return err
}

func (s *sumoLogicClient) createCWLogLine(item map[string]interface{}) {

	message, ok := item["record"].(map[string]interface{})
	if ok {
		delete(item, "record")
	}
	// Todo convert this to struct
	metric := message["metrics"].(map[string]interface{})
	cwMessageLine := fmt.Sprintf("REPORT RequestId: %v	Duration: %v ms	Billed Duration: %v ms 	Memory Size: %v MB	Max Memory Used: %v MB",
		message["requestId"], metric["durationMs"], metric["billedDurationMs"], metric["memorySizeMB"], metric["maxMemoryUsedMB"])
	item["message"] = cwMessageLine
}

func (s *sumoLogicClient) getLogGroup() string {
	return fmt.Sprintf("/aws/lambda/%s", s.config.FunctionName)
}

func (s *sumoLogicClient) getLogStream() string {
	currentTime := time.Now().UTC()
	currentDate := fmt.Sprintf("%d/%02d/%02d", currentTime.Year(), currentTime.Month(), currentTime.Day())
	return fmt.Sprintf("%s/[%s]%s", currentDate, s.config.FunctionVersion, config.ExtensionName)
}

func (s *sumoLogicClient) enhanceLogs(msg responseBody) {
	s.logger.Debugln("Enhancing logs")
	for _, item := range msg {
		// item["FunctionName"] = s.config.FunctionName
		// item["FunctionVersion"] = s.config.FunctionVersion
		// creating loggroup/logstream as they are not available in Env.
		// This is done to make it compatible with AWS Observability

		item["logGroup"] = s.getLogGroup()
		item["logStream"] = s.getLogStream()

		item["IsColdStart"] = s.getColdStart()
		item["LayerVersion"] = config.SumoLogicExtensionLayerVersionSuffix
		logType, ok := item["type"].(string)
		if ok && logType == "function" {
			message, ok := item["record"].(string)
			if ok {
				delete(item, "record")
			}
			message = strings.TrimSpace(message)
			json, err := utils.ParseJson(message)
			if err != nil {
				item["message"] = message
			} else {
				item["message"] = json
			}
		} else if ok && logType == "platform.report" {
			s.createCWLogLine(item)
		}
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
	if len(rawmsg) > 0 {
		// converting to arr of maps
		msgArr, err := s.transformBytesToArrayOfMap(rawmsg)
		if err != nil {
			return fmt.Errorf("SendLogs - transformBytesToArrayOfMap failed: %v", err)
		}
		s.logger.Debugf("SendLogs - Total log lines transformed: %d", len(msgArr))
		s.enhanceLogs(msgArr)

		// converting back to chunks of string
		chunks, err := s.createChunks(msgArr)
		if err != nil {
			return fmt.Errorf("SendLogs - createChunks failed: %v", err)
		}
		var errorCount int = 0
		for _, strobj := range chunks {
			err := s.postToSumo(ctx, &strobj)
			if err != nil {
				errorCount++
			}
		}
		if errorCount > 0 {
			err = fmt.Errorf("SendLogs - errors during postToSumo: %d", errorCount)
			return err
		}
	}
	return nil
}

func (s *sumoLogicClient) SendAllLogs(ctx context.Context, allMessages [][]byte) error {
	if (len(allMessages) == 0) {
		s.logger.Debugf("SendAllLogs: No messages to send")
		return nil
	}

	s.logger.Debugf("SendAllLogs: Attempting to send %d payloads from dataqueue to SumoLogic", len(allMessages))

	var errorCount int = 0
	var totalitems int = 0
	var payload responseBody
	for _, rawmsg := range allMessages {
		// converting to arr of maps
		msgArr, err := s.transformBytesToArrayOfMap(rawmsg)
		if err != nil {
			s.logger.Error("SendAllLogs: Error in transforming bytes to array of struct", err.Error())
			errorCount++
			continue
		}

		if len(msgArr) > 0 {
			// enhancing logs
			s.enhanceLogs(msgArr)
			totalitems += len(msgArr)
			// converting back to string
			for _, item := range msgArr {
				payload = append(payload, item)
			}
		}
	}
	s.logger.Debugf("SendAllLogs: Enhanced TotalLogItems - %d \n", totalitems)
	// converting back to chunks of string
	chunks, err := s.createChunks(payload)
	if err != nil {
		return fmt.Errorf("SendAllLogs: CreateChunks failed - %v", err)
	}
	for _, strobj := range chunks {
		err := s.postToSumo(ctx, &strobj)
		if err != nil {
			errorCount++
		}
	}
	if errorCount > 0 {
		err = fmt.Errorf("SendAllLogs: Errors during postToSumo - %d", errorCount)
		return err
	} else {
		s.logger.Debugf("SendAllLogs: Sent TotalChunks - %d \n", totalitems)
	}

	return nil
}

func (s *sumoLogicClient) postToSumo(ctx context.Context, logStringToSend *string) error {

	s.logger.Debug("postToSumo: Attempting to send to Sumo Endpoint")

	// compressing here because Sumo recommends payload size of 1MB before compression
	bytedata := utils.Compress(logStringToSend)
	createBuffer := func() *bytes.Buffer {
		dest := make([]byte, len(bytedata))
		copy(dest, bytedata)
		return bytes.NewBuffer(dest)
	}
	buf := createBuffer()
	response, err := s.makeRequest(ctx, buf)
	if response != nil {
		defer response.Body.Close()
	}
	if (err != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
		s.logger.Errorf("postToSumo: Not able to post statuscode -  %v %v\n", err, response)
		err := utils.Retry(func(attempt int) (bool, error) {
			s.logger.Debugf("postToSumo: Waiting for %v ms for retry attempt - %v\n", s.config.RetrySleepTime, attempt)
			time.Sleep(s.config.RetrySleepTime)
			buf := createBuffer()
			retryResponse, errRetry := s.makeRequest(ctx, buf)
			if (errRetry != nil) || (retryResponse.StatusCode != 200 && retryResponse.StatusCode != 302 && retryResponse.StatusCode < 500) {
				if errRetry == nil {
					errRetry = fmt.Errorf("statuscode %v", retryResponse.StatusCode)
				}
				s.logger.Error("postToSumo: Not able to post - ", errRetry)
				return attempt < s.config.MaxRetryAttempts, errRetry
			} else if retryResponse.StatusCode == 200 {
				s.logger.Debugf("postToSumo: Post of logs successful after retry %v attempts\n", attempt)
				return true, nil
			}
			return attempt < s.config.MaxRetryAttempts, errRetry
		}, s.config.NumRetry)
		if err != nil {
			s.logger.Error("postToSumo: Finished retrying Error - ", err)
			if s.config.EnableFailover {
				buf = createBuffer()
				err := s.failoverHandler(buf)
				if err != nil {
					s.logger.Errorf("postToSumo: Dropping messages as post to S3 failed - %v\n", err)
					return err
				}
			} else {
				s.logger.Info("postToSumo: Dropping messages as no failover enabled.")
			}
		}
	} else if response.StatusCode == 200 {
		s.logger.Debugf("postToSumo: Post of logs successful")
	}

	return nil
}
