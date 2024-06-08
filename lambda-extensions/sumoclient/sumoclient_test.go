package sumoclient

import (
	"context"
	"fmt"
	ioutil "io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"

	"github.com/sirupsen/logrus"
)

func setupEnv() {

	os.Setenv("SUMO_NUM_RETRIES", "3")
	os.Setenv("SUMO_S3_BUCKET_NAME", "test-bucket")
	os.Setenv("SUMO_S3_BUCKET_REGION", "us-east-1")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
	os.Setenv("AWS_LAMBDA_LOG_GROUP_NAME", "/aws/lambda/testfunctionpython")
	os.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2020/11/03/[$LATEST]e5ef8fe91380465fab7da53f5bac50f6")
	os.Setenv("SUMO_ENABLE_FAILOVER", "true")
	os.Setenv("SUMO_LOG_LEVEL", "5")
	os.Setenv("SUMO_MAX_DATAQUEUE_LENGTH", "10")
	os.Setenv("SUMO_MAX_CONCURRENT_REQUESTS", "3")
	os.Setenv("SUMO_LOG_LEVEL", "DEBUG")
	os.Setenv("SUMO_RETRY_SLEEP_TIME_MS", "50")
	os.Setenv("SUMO_LOG_TYPES", "function")
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Error(message)
}

func assertNotEmpty(t *testing.T, a interface{}, message string) {
	if a != nil {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v is nil", a)
	}
	t.Error(message)
}

func TestSumoClient(t *testing.T) {
	var logger = logrus.New().WithField("Name", "sumologic-extension")
	var config *cfg.LambdaExtensionConfig
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		logger.Info("Received", s)
	}()
	logger.Logger.SetOutput(os.Stdout)
	setupEnv()

	successEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, r.Method, http.MethodPost, "Method is not POST")
		assertNotEmpty(t, r.Header.Get("X-Sumo-Name"), "Source Name Header not present")
		assertNotEmpty(t, r.Header.Get("X-Sumo-Host"), "Source Host Header not present")

		reqBytes, err := ioutil.ReadAll(r.Body)
		assertEqual(t, err, nil, "Received error")
		defer r.Body.Close()
		assertNotEmpty(t, reqBytes, "Empty Data in Post")
		w.WriteHeader(200)
	}))

	defer successEndpointServer.Close()
	os.Setenv("SUMO_HTTP_ENDPOINT", successEndpointServer.URL)
	throttlingEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(429)
	}))
	defer throttlingEndpointServer.Close()

	t.Log("\nvalidating config\n======================")
	config, err = cfg.GetConfig()
	assertEqual(t, err, nil, "GetConfig should not generate error")

	logger.Logger.SetLevel(config.LogLevel)

	t.Log("\nsuccess scenario\n======================")
	client := NewLogSenderClient(logger, config)
	var logs = []byte("[{\"key\": \"value\"}]")
	assertEqual(t, client.SendLogs(ctx, logs), nil, "SendLogs should not generate error")

	config.MaxDataPayloadSize = 500
	t.Log("\nchunking large data\n======================")
	var largedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"{'log': 'logger error json statement in python: 0'}"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	assertEqual(t, client.SendLogs(ctx, largedata), nil, "SendLogs should not generate error")

	config.EnhanceJsonLogs = false
	t.Log("\n enhance json logs = false with single line large data\n======================")
	var singlinelargedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"[ERROR]\t2024-05-04T13:58:12.928Z\t917552a8-fabd-4cf0-a2ae-b7863210bc4e\tdd0509ae-6c9e-48d5-afe7-d05e60f6a69b logger error statement in python: 0\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	assertEqual(t, client.SendLogs(ctx, singlinelargedata), nil, "SendLogs should not generate error")

	config.EnhanceJsonLogs = false
	t.Log("\n enhance json logs = false with json line large data\n======================")
	var jsonlinelargedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"{'log': 'logger error json statement in python: 0'}"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	assertEqual(t, client.SendLogs(ctx, jsonlinelargedata), nil, "SendLogs should not generate error")


	t.Log("\ntesting flushall\n======================")
	var multiplelargedata = [][]byte{
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
	}
	err = client.FlushAll(multiplelargedata)
	assertEqual(t, strings.HasPrefix(err.Error(), "FlushAll - Errors during chunk creation: 0, Errors during flushing to S3"), true, "FlushAll should generate error")

	//Todo mock S3 client to improve below tests

	t.Log("\ntesting report data conversion\n================")
	var reportLogs = []byte(`[{"record":{"metrics":{"billedDurationMs":120000,"durationMs":122066.85,"maxMemoryUsedMB":74,"memorySizeMB":128},"requestId":"fcea12d9-e0b4-43b2-a9a2-04d04519539f"},"time":"2020-11-02T20:33:16.536Z","type":"platform.report"}]`)
	assertEqual(t, client.SendLogs(ctx, reportLogs), nil, "SendLogs should not generate error")

	t.Log("\ntesting SendAllLogs\n======================")
	assertEqual(t, client.SendAllLogs(ctx, multiplelargedata), nil, "SendAllLogs should not generate error")
	//Todo remove this function from sumologic-extension
	// t.Log("\ntesting sumo if no s3 failover\n=================")
	// config.EnableFailover = false
	// dataQueue <- largedata
	// flushData(ctx, 10*1000)

	config.SumoHTTPEndpoint = throttlingEndpointServer.URL
	t.Log("\nretry scenario + failover\n======================")
	err = client.SendLogs(ctx, logs)
	assertEqual(t, strings.HasPrefix(err.Error(), "SendLogs - errors during postToSumo: 1"), true, "SendLogs should generate error")

}
