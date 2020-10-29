package sumoclient

import (
	"bytes"
	"config"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"utils"

	uuid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	connectionTimeoutValue = 10000
	maxRetryAttempts       = 5
	sleepTime              = 300 * time.Millisecond
	maxDataPayloadSize     = 1024 * 1024 // 1 MB
)

// LogSender interface which needs to be implemented to send logs
type LogSender interface {
	SendLogs(context.Context, []byte) error
}

// sumoLogicClient implements LogSender interface
type sumoLogicClient struct {
	connectionTimeout int
	httpClient        http.Client
	config            *config.LambdaExtensionConfig
	logger            *logrus.Entry
}

type responseBody []interface{}

// NewLogSenderClient returns interface pointing to the concrete version of LogSender client
func NewLogSenderClient(logger *logrus.Entry) LogSender {
	cfg, _ := config.GetConfig()
	var logSenderClient LogSender = &sumoLogicClient{
		connectionTimeout: connectionTimeoutValue,
		httpClient:        http.Client{Timeout: time.Duration(connectionTimeoutValue * int(time.Millisecond))},
		config:            cfg,
		logger:            logger,
	}
	return logSenderClient
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

func getchunkSize() {

}

// SendToSumo send logs to sumo http endpoint returns
func (s *sumoLogicClient) SendLogs(ctx context.Context, rawmsg []byte) error {
	var err error
	if len(rawmsg) > 0 {
		var msg responseBody
		var err error
		err = json.Unmarshal(rawmsg, &msg)
		if err != nil {
			return err
		}
		strobj := string(rawmsg)
		s.postToSumo(ctx, &strobj)
		// for _, obj := range msg {
		// 	// Todo add chunking
		// 	strobj := fmt.Sprintf("%v", obj)
		// 	s.postToSumo(ctx, &strobj)
		// }
	}
	return err
}

func (s *sumoLogicClient) postToSumo(ctx context.Context, logStringToSend *string) error {
	s.logger.Info("Attempting to send to Sumo Endpoint")

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
		s.logger.Infof("Waiting for %v ms to retry\n", sleepTime)
		time.Sleep(sleepTime)
		err := utils.Retry(func(attempt int64) (bool, error) {
			var errRetry error
			buf := createBuffer()
			response, errRetry = s.makeRequest(ctx, buf)
			if (errRetry != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
				if errRetry == nil {
					errRetry = fmt.Errorf("statuscode %v", response.StatusCode)
				}
				s.logger.Infof("Not able to post: ", errRetry)
				s.logger.Infof("Waiting for %v ms to retry attempts done: %v\n", sleepTime, attempt)
				time.Sleep(sleepTime)
				return attempt < maxRetryAttempts, errRetry
			} else if response.StatusCode == 200 {
				s.logger.Debugf("Post of logs successful after retry %v attempts\n", attempt)
				return true, nil
			}
			return attempt < maxRetryAttempts, errRetry
		}, s.config.MaxRetry)
		if err != nil {
			s.logger.Infof("Finished retrying Error: ", err)
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

// func main() {
// 	os.Setenv("MAX_RETRY", "3")
// 	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw=")
// 	os.Setenv("S3_BUCKET_NAME", "test-angad")
// 	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
// 	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
// 	client := NewLogSenderClient()
// 	var logs = "hello world"
// 	fmt.Println(client.SendLogs(&logs))
// }
