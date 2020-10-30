package main

import (
	cfg "config"
	"context"
	"lambdaapi"
	"os"
	"os/signal"
	"path/filepath"
	sumocli "sumoclient"
	"sync"
	"syscall"
	"time"
	"utils"

	"github.com/sirupsen/logrus"
)

const (
	maxchannelLength       = 20
	maxConcurrentConsumers = 3
)

var (
	logger          = logrus.New().WithField("Name", extensionName)
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = lambdaapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"), extensionName)
	dataQueue       = make(chan []byte, 20)
	httpServer      = lambdaapi.NewHTTPServer(dataQueue, logger)
)

var sumoclient sumocli.LogSender
var config *cfg.LambdaExtensionConfig
var runningLocal = false

func init() {

	logger.Logger.SetOutput(os.Stdout)
	if !runningLocal {
		logger.Logger.SetLevel(logrus.InfoLevel)
		logger.Info("Initializing Extension.........")
		// Register early so Runtime could start in parallel
		logger.Info("Registering Extension to Run Time API Client..........")
		registerResponse, err := extensionClient.RegisterExtension(nil)
		if err != nil {
			extensionClient.InitError(nil, "Error during extension registration."+err.Error())
		}
		logger.Info("Succcessfully Registered with Run Time API Client: ", utils.PrettyPrint(registerResponse))
		// Start HTTP Server before subscription in a goRoutine
		go httpServer.HTTPServerStart()

		// Creating config and performing validation
		config, err = cfg.GetConfig()
		if err != nil {
			extensionClient.InitError(nil, "Error during Fetching Env Variables."+err.Error())
		}
		sumoclient = sumocli.NewLogSenderClient(logger, config)
		// Subscribe to Logs API
		logger.Info("Subscribing Extension to Logs API........")
		subscribeResponse, err := extensionClient.SubscribeToLogsAPI(nil, config.LogTypes)
		if err != nil {
			extensionClient.InitError(nil, "Error during Logs API Subscription."+err.Error())
		}
		logger.Info("Successfully subscribed to Logs API: ", utils.PrettyPrint(string(subscribeResponse)))
		logger.Info("Successfully Intialized Sumo Logic Extension.")
	}
}

func flushDataQueue() {
	rawMsgArr := make([][]byte, maxchannelLength)
	for {
		rawmsg, more := <-dataQueue
		if !more {
			err := sumoclient.FlushAll(rawMsgArr)
			if err != nil {
				logger.Debugln("Unable to flushing DataQueue", err.Error())
			}
			close(dataQueue)
			return
		}
		rawMsgArr = append(rawMsgArr, rawmsg)
	}

}
func consumer(ctx context.Context, wg *sync.WaitGroup, rawmsg []byte) {
	defer wg.Done()
	err := sumoclient.SendLogs(ctx, rawmsg)
	if err != nil {
		logger.Error("Error during Send Logs to Sumo Logic.", err.Error())
	}
	return
}

// processEvents is - Will block until shutdown event is received or cancelled via the context..
func processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			flushDataQueue()
			return
		default:
			wg := new(sync.WaitGroup)
			logger.Debug("Consuming data from dataQueue")
			counter := 0
			for i := 0; i < maxConcurrentConsumers; i++ {
				rawmsg := <-dataQueue // read from a closed channel will be the zero value
				if len(rawmsg) > 0 {
					counter++
					wg.Add(1)
					go consumer(ctx, wg, rawmsg)
				} else {
					break
				}
			}
			logger.Debugf("Waiting for %d consumer to finish their tasks", counter)
			wg.Wait()
			if !runningLocal {
				logger.Info("Waiting for Run Time API event...")
				nextResponse, err := extensionClient.NextEvent(ctx)
				if err != nil {
					logger.Error("Error:", err.Error())
					logger.Info("Exiting")
					return
				}
				// Exit if we receive a SHUTDOWN event
				if nextResponse.EventType == lambdaapi.Shutdown {
					logger.Info("Received SHUTDOWN event")
					logger.Info("Exiting")
					flushDataQueue()
					return
				} else if nextResponse.EventType == lambdaapi.Invoke {
					logger.Info("Received Invoke event.", utils.PrettyPrint(nextResponse))
				}
			} else {
				time.Sleep(3 * time.Second)
			}
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
	if runningLocal {
		// Testing
		logger.Logger.SetLevel(logrus.DebugLevel)
		os.Setenv("MAX_RETRY", "3")
		os.Setenv("SUMO_HTTP_ENDPOINT", "https://collectors.sumologic.com/receiver/v1/http/ZaVnC4dhaV2ZZls3q0ihtegxCvl_lvlDNWoNAvTS5BKSjpuXIOGYgu7QZZSd-hkZlub49iL_U0XyIXBJJjnAbl6QK_JX0fYVb_T4KLEUSbvZ6MUArRavYw==")
		os.Setenv("S3_BUCKET_NAME", "test-angad")
		os.Setenv("S3_BUCKET_REGION", "test-angad")
		os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "himlambda")
		os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "Latest$")
		os.Setenv("ENABLE_FAILOVER", "true")
		config, _ = cfg.GetConfig()
		sumoclient = sumocli.NewLogSenderClient(logger, config)

		go func() {
			numDataGenerated := 100
			largedata := []byte(`[{"time":"2020-10-27T15:36:14.133Z","type":"platform.start","record":{"requestId":"7313c951-e0bc-4818-879f-72d202e24727","version":"$LATEST"}},{"time":"2020-10-27T15:36:14.282Z","type":"platform.logsSubscription","record":{"name":"sumologic-extension","state":"Subscribed","types":["platform","function"]}},{"time":"2020-10-27T15:36:14.283Z","type":"function","record":"2020-10-27T15:36:14.281Z\tundefined\tINFO\tLoading function\n"},{"time":"2020-10-27T15:36:14.283Z","type":"platform.extension","record":{"name":"sumologic-extension","state":"Ready","events":["INVOKE"]}},{"time":"2020-10-27T15:36:14.301Z","type":"function","record":"2020-10-27T15:36:14.285Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue1 = value1\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue2 = value2\n"},{"time":"2020-10-27T15:36:14.302Z","type":"function","record":"2020-10-27T15:36:14.301Z\t7313c951-e0bc-4818-879f-72d202e24727\tINFO\tvalue3 = value3\n"}]`)
			for i := 0; i < numDataGenerated; i++ {
				logger.Debugf("Producing data into dataQueue: %d", i+1)
				dataQueue <- largedata
				sleepTime := i % 4
				time.Sleep(time.Duration(sleepTime) * time.Second)
			}
			close(dataQueue)
			return
		}()

	}
	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx)
}
