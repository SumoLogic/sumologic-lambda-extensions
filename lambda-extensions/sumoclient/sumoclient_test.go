package sumoclient

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"

	cfg "github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/config"

	"github.com/sirupsen/logrus"
)

func setupEnv() {

	os.Setenv("SUMO_NUM_RETRIES", "3")
	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
	os.Setenv("SUMO_S3_BUCKET_NAME", "test-angad")
	os.Setenv("SUMO_S3_BUCKET_REGION", "test-angad")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
	os.Setenv("AWS_LAMBDA_LOG_GROUP_NAME", "/aws/lambda/testfunctionpython")
	os.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "2020/11/03/[$LATEST]e5ef8fe91380465fab7da53f5bac50f6")
	os.Setenv("SUMO_ENABLE_FAILOVER", "true")
	os.Setenv("SUMO_LOG_LEVEL", "5")
	os.Setenv("SUMO_MAX_DATAQUEUE_LENGTH", "10")
	os.Setenv("SUMO_MAX_CONCURRENT_REQUESTS", "3")
	os.Setenv("SUMO_LOG_LEVEL", "DEBUG")
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
	var largedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	assertEqual(t, client.SendLogs(ctx, largedata), nil, "SendLogs should not generate error")

	t.Log("\ntesting flushall\n======================")
	var multiplelargedata = [][]byte{
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
	}
	assertEqual(t, client.FlushAll(multiplelargedata), nil, "FlushAll should not generate error")

	t.Log("\ntesting report data conversion\n================")
	var reportLogs = []byte(`[{"record":{"metrics":{"billedDurationMs":120000,"durationMs":122066.85,"maxMemoryUsedMB":74,"memorySizeMB":128},"requestId":"fcea12d9-e0b4-43b2-a9a2-04d04519539f"},"time":"2020-11-02T20:33:16.536Z","type":"platform.report"}]`)
	assertEqual(t, client.SendLogs(ctx, reportLogs), nil, "SendLogs should not generate error")

	// t.Log("\ntesting sumo if no s3 failover\n=================")
	// config.EnableFailover = false
	// dataQueue <- largedata
	// flushData(ctx, 10*1000)

	t.Log("\nretry scenario + failover\n======================")
	config.SumoHTTPEndpoint = "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw="
	assertEqual(t, client.SendLogs(ctx, logs), nil, "SendLogs should not generate error")

}
