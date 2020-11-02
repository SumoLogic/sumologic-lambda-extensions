package main

import (
	cfg "config"
	"context"
	"fmt"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	sumocli "sumoclient"
	"syscall"
	"time"
	"workers"

	"github.com/sirupsen/logrus"
)

var (
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	logger          = logrus.New().WithField("Name", extensionName)
)
var producer workers.TaskProducer
var consumer workers.TaskConsumer
var config *cfg.LambdaExtensionConfig
var dataQueue chan []byte
var quitQueue chan bool

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {

Loop:
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Flush Queue")
			consumer.FlushDataQueue()
			return
		case <-quitQueue:
			logger.Debug("Quit Queue")
			break Loop
		default:
			consumer.DrainQueue(ctx)
			logger.Info("Calling Next Event ...")
			time.Sleep(5 * time.Second)
			break
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		logger.Info("Received", s)
	}()
	logger.Logger.SetOutput(os.Stdout)

	// Creating config and performing validation

	os.Setenv("NUM_RETRIES", "3")
	os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
	os.Setenv("S3_BUCKET_NAME", "test-angad")
	os.Setenv("S3_BUCKET_REGION", "test-angad")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
	os.Setenv("ENABLE_FAILOVER", "true")
	os.Setenv("LOG_LEVEL", "5")
	os.Setenv("MAX_DATAQUEUE_LENGTH", "10")
	os.Setenv("MAX_CONCURRENT_REQUESTS", "3")

	fmt.Println("\nvalidating config\n======================")
	config, err := cfg.GetConfig()
	fmt.Println(config, err)

	logger.Logger.SetLevel(config.LogLevel)
	logger.Debug(config)
	dataQueue = make(chan []byte, config.MaxDataQueueLength)

	// Start HTTP Server before subscription in a goRoutine
	// producer = workers.NewTaskProducer(dataQueue, logger)
	// go producer.Start()

	quitQueue = make(chan bool, 1)

	// Creating SumoTaskConsumer
	consumer = workers.NewTaskConsumer(dataQueue, config, logger)

	go func() {
		numDataGenerated := 10
		largedata := []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
		enddata := []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.end","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
		for i := 0; i < numDataGenerated-1; i++ {
			logger.Debugf("Producing data into dataQueue: %d", i+1)
			dataQueue <- largedata
			sleepTime := i%5 + 1
			logger.Debugf("Waiting for data for %d seconds", sleepTime)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
		dataQueue <- enddata
		close(dataQueue)
		quitQueue <- true
		return
	}()

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)

	fmt.Println("Testing SumoClient")
	fmt.Println("\nsuccess scenario\n======================")
	client := sumocli.NewLogSenderClient(logger, config)
	var logs = []byte("[{\"key\": \"value\"}]")
	fmt.Println(client.SendLogs(ctx, logs))

	config.MaxDataPayloadSize = 500
	fmt.Println("\nchunking large data\n======================")
	var largedata = []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
	fmt.Println(client.SendLogs(ctx, largedata))

	fmt.Println("\ntesting flushall\n======================")
	var multiplelargedata = [][]byte{
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
		[]byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`),
	}
	fmt.Println(client.FlushAll(multiplelargedata))

	fmt.Println("\nretry scenario + failover\n======================")
	config.SumoHTTPEndpoint = "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw="
	fmt.Println(client.SendLogs(ctx, logs))
}
