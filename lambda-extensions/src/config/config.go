package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"utils"

	"github.com/sirupsen/logrus"
)

// LambdaExtensionConfig config for storing all configurable parameters
type LambdaExtensionConfig struct {
	SumoHTTPEndpoint       string
	EnableFailover         bool
	S3BucketName           string
	S3BucketRegion         string
	NumRetry               int
	AWSLambdaRuntimeAPI    string
	LogTypes               []string
	FunctionName           string
	FunctionVersion        string
	LogLevel               logrus.Level
	MaxDataQueueLength     int
	MaxConcurrentRequests  int
	ProcessingSleepTime    time.Duration
	MaxRetryAttempts       int
	RetrySleepTime         time.Duration
	ConnectionTimeoutValue time.Duration
	MaxDataPayloadSize     int
}

var validLogTypes = []string{"platform", "function", "extension"}

// GetConfig to get config instance
func GetConfig() (*LambdaExtensionConfig, error) {

	config := &LambdaExtensionConfig{
		SumoHTTPEndpoint:       os.Getenv("SUMO_HTTP_ENDPOINT"),
		S3BucketName:           os.Getenv("S3_BUCKET_NAME"),
		S3BucketRegion:         os.Getenv("S3_BUCKET_REGION"),
		AWSLambdaRuntimeAPI:    os.Getenv("AWS_LAMBDA_RUNTIME_API"),
		FunctionName:           os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
		FunctionVersion:        os.Getenv("AWS_LAMBDA_FUNCTION_VERSION"),
		MaxRetryAttempts:       5,
		RetrySleepTime:         300 * time.Millisecond,
		ConnectionTimeoutValue: 10000 * time.Millisecond,
		MaxDataPayloadSize:     1024 * 1024, // 1 MB
	}

	(*config).setDefaults()

	err := (*config).validateConfig()

	if err != nil {
		return config, err
	}
	return config, nil
}
func (cfg *LambdaExtensionConfig) setDefaults() {
	numRetry := os.Getenv("NUM_RETRIES")
	processingSleepTime := os.Getenv("PROCESSING_SLEEP_TIME_MS")
	logLevel := os.Getenv("LOG_LEVEL")
	maxDataQueueLength := os.Getenv("MAX_DATAQUEUE_LENGTH")
	maxConcurrentRequests := os.Getenv("MAX_CONCURRENT_REQUESTS")
	enableFailover := os.Getenv("ENABLE_FAILOVER")
	logTypes := os.Getenv("LOG_TYPES")
	if numRetry == "" {
		cfg.NumRetry = 3
	}
	if logLevel == "" {
		cfg.LogLevel = logrus.InfoLevel
	}
	if maxDataQueueLength == "" {
		cfg.MaxDataQueueLength = 20
	}
	if maxConcurrentRequests == "" {
		cfg.MaxConcurrentRequests = 3
	}

	if enableFailover == "" {
		cfg.EnableFailover = false
	}
	if cfg.AWSLambdaRuntimeAPI == "" {
		cfg.AWSLambdaRuntimeAPI = "127.0.0.1:9001"
	}
	if logTypes == "" {
		cfg.LogTypes = validLogTypes
	} else {
		cfg.LogTypes = strings.Split(logTypes, ",")
	}
	if processingSleepTime == "" {
		cfg.ProcessingSleepTime = 15000 * time.Millisecond
	}

}

func (cfg *LambdaExtensionConfig) validateConfig() error {
	numRetry := os.Getenv("NUM_RETRIES")
	logLevel := os.Getenv("LOG_LEVEL")
	maxDataQueueLength := os.Getenv("MAX_DATAQUEUE_LENGTH")
	maxConcurrentRequests := os.Getenv("MAX_CONCURRENT_REQUESTS")
	enableFailover := os.Getenv("ENABLE_FAILOVER")
	processingSleepTime := os.Getenv("PROCESSING_SLEEP_TIME_MS")
	var allErrors []string
	var err error

	if cfg.SumoHTTPEndpoint == "" {
		allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT not set in environment variable")
	}

	// Todo test url valid
	if cfg.SumoHTTPEndpoint != "" {
		_, err = url.ParseRequestURI(cfg.SumoHTTPEndpoint)
		if err != nil {
			allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT is not Valid")
		}
	}

	if enableFailover != "" {
		cfg.EnableFailover, err = strconv.ParseBool(enableFailover)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse ENABLE_FAILOVER: %v", err))
		}
	}

	if cfg.EnableFailover == true {
		if cfg.S3BucketName == "" {
			allErrors = append(allErrors, "S3_BUCKET_NAME not set in environment variable")
		}
		if cfg.S3BucketRegion == "" {
			allErrors = append(allErrors, "S3_BUCKET_REGION not set in environment variable")
		}
	}

	if numRetry != "" {
		customNumRetry, err := strconv.ParseInt(numRetry, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse NUM_RETRIES: %v", err))
		} else {
			cfg.NumRetry = int(customNumRetry)
		}
	}

	if processingSleepTime != "" {
		customProcessingSleepTime, err := strconv.ParseInt(processingSleepTime, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse PROCESSING_SLEEP_TIME_MS: %v", err))
		} else {
			cfg.ProcessingSleepTime = time.Duration(customProcessingSleepTime) * time.Millisecond
		}
	}

	if maxDataQueueLength != "" {
		customMaxDataQueueLength, err := strconv.ParseInt(maxDataQueueLength, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse MAX_DATAQUEUE_LENGTH: %v", err))
		} else {
			cfg.MaxDataQueueLength = int(customMaxDataQueueLength)
		}

	}
	if maxConcurrentRequests != "" {
		customMaxConcurrentRequests, err := strconv.ParseInt(maxConcurrentRequests, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse MAX_CONCURRENT_REQUESTS: %v", err))
		} else {
			cfg.MaxConcurrentRequests = int(customMaxConcurrentRequests)
		}

	}
	if logLevel != "" {
		customloglevel, err := strconv.ParseInt(logLevel, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse LOG_LEVEL: %v", err))
		} else {
			cfg.LogLevel = logrus.Level(customloglevel)
		}

	}

	// test valid log format type
	for _, logType := range cfg.LogTypes {
		if !utils.StringInSlice(strings.TrimSpace(logType), validLogTypes) {
			allErrors = append(allErrors, fmt.Sprintf("logType %s is unsupported", logType))
		}
	}

	if len(allErrors) > 0 {
		err = errors.New(strings.Join(allErrors, ", "))
	}

	return err
}
