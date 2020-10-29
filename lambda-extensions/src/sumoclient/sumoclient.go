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

var maxDataPayloadSize int = 1024 * 1024 // 1 MB
var isColdStart = true

const (
	connectionTimeoutValue = 10000
	maxRetryAttempts       = 5
	sleepTime              = 300 * time.Millisecond
)

// LogSender interface which needs to be implemented to send logs
type LogSender interface {
	SendLogs(context.Context, []byte) error
	FlushAll([][]byte) error
}

// sumoLogicClient implements LogSender interface
type sumoLogicClient struct {
	connectionTimeout int
	httpClient        http.Client
	config            *config.LambdaExtensionConfig
	logger            *logrus.Entry
}

// It is assumed that logs will be array of json objects and all channel payloads satisfy this format
type responseBody []map[string]interface{}

// NewLogSenderClient returns interface pointing to the concrete version of LogSender client
func NewLogSenderClient(logger *logrus.Entry) LogSender {
	// setting the cold start variable here since this function is called

	cfg, _ := config.GetConfig()
	var logSenderClient LogSender = &sumoLogicClient{
		connectionTimeout: connectionTimeoutValue,
		httpClient:        http.Client{Timeout: time.Duration(connectionTimeoutValue * int(time.Millisecond))},
		config:            cfg,
		logger:            logger,
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
		s.logger.Debug("Total objects to S3: ", totalitems)

		// compressing and pushing to S3
		gzippedBuffer := utils.CompressBuffer(&payload)
		s.failoverHandler(gzippedBuffer)

		if errorCount > 0 {
			err = fmt.Errorf("Total errors during flusing to S3: %d", errorCount)
			s.logger.Error(err.Error())
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
		return msg, fmt.Errorf("Error in parsing payload: %v", err)
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
		if chunkSize+itemSize+1 >= maxDataPayloadSize {
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
		s.logger.Debugf("Waiting for %v ms to retry\n", sleepTime)
		time.Sleep(sleepTime)
		err := utils.Retry(func(attempt int64) (bool, error) {
			var errRetry error
			buf := createBuffer()
			response, errRetry = s.makeRequest(ctx, buf)
			if (errRetry != nil) || (response.StatusCode != 200 && response.StatusCode != 302 && response.StatusCode < 500) {
				if errRetry == nil {
					errRetry = fmt.Errorf("statuscode %v", response.StatusCode)
				}
				s.logger.Error("Not able to post: ", errRetry)
				s.logger.Debugf("Waiting for %v ms to retry attempts done: %v\n", sleepTime, attempt)
				time.Sleep(sleepTime)
				return attempt < maxRetryAttempts, errRetry
			} else if response.StatusCode == 200 {
				s.logger.Debugf("Post of logs successful after retry %v attempts\n", attempt)
				return true, nil
			}
			return attempt < maxRetryAttempts, errRetry
		}, s.config.MaxRetry)
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

/*
func main() {
	var logger = logrus.New().WithField("Name", "sumo-extension")
	logger.Logger.SetLevel(logrus.DebugLevel)
	logger.Logger.SetOutput(os.Stdout)
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		fmt.Println("Received", s)
	}()
	os.Setenv("MAX_RETRY", "3")
	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
	os.Setenv("S3_BUCKET_NAME", "test-angad")
	os.Setenv("S3_BUCKET_REGION", "test-angad")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
	os.Setenv("ENABLE_FAILOVER", "true")

	// validating config
	cfg, err := config.GetConfig()
	fmt.Println(cfg, err)

	// success scenario
	client := NewLogSenderClient(logger)
	var logs = []byte("[{\"key\": \"value\"}]")
	fmt.Println(client.SendLogs(ctx, logs))

	// retry scenario + failover
	client = NewLogSenderClient(logger)
	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw=")
	fmt.Println(client.SendLogs(ctx, logs))

	maxDataPayloadSize = 500
	// chunking large data
	var largedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	fmt.Println(client.SendLogs(ctx, largedata))

	// testing flushall
	var multiplelargedata = [][]byte{
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
	}
	fmt.Println(client.FlushAll(multiplelargedata))
}
*/
