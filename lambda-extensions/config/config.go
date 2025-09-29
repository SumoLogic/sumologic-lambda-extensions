package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SumoLogic/sumologic-lambda-extensions/lambda-extensions/utils"

	"github.com/sirupsen/logrus"
)

// LambdaExtensionConfig config for storing all configurable parameters
type LambdaExtensionConfig struct {
	SumoHTTPEndpoint       string
	KMSKeyId               string
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
	MaxRetryAttempts       int
	RetrySleepTime         time.Duration
	ConnectionTimeoutValue time.Duration
	MaxDataPayloadSize     int
	LambdaRegion           string
	SourceCategoryOverride string
	EnhanceJsonLogs        bool
	EnableSpanDrops        bool
	KmsCacheSeconds        int64
	TelemetryTimeoutMs     int
	TelemetryMaxBytes      int64
	TelemetryMaxItems      int
}

var defaultLogTypes = []string{"platform", "function"}
var validLogTypes = []string{"platform", "function", "extension"}

// GetConfig to get config instance
func GetConfig() (*LambdaExtensionConfig, error) {

	config := &LambdaExtensionConfig{
		SumoHTTPEndpoint:       os.Getenv("SUMO_HTTP_ENDPOINT"),
		KMSKeyId:               os.Getenv("KMS_KEY_ID"),
		S3BucketName:           os.Getenv("SUMO_S3_BUCKET_NAME"),
		S3BucketRegion:         os.Getenv("SUMO_S3_BUCKET_REGION"),
		AWSLambdaRuntimeAPI:    os.Getenv("AWS_LAMBDA_RUNTIME_API"),
		FunctionName:           os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
		FunctionVersion:        os.Getenv("AWS_LAMBDA_FUNCTION_VERSION"),
		LambdaRegion:           os.Getenv("AWS_REGION"),
		SourceCategoryOverride: os.Getenv("SOURCE_CATEGORY_OVERRIDE"),
		MaxRetryAttempts:       5,
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
	numRetry := os.Getenv("SUMO_NUM_RETRIES")
	retrySleepTime := os.Getenv("SUMO_RETRY_SLEEP_TIME_MS")
	logLevel := os.Getenv("SUMO_LOG_LEVEL")
	maxDataQueueLength := os.Getenv("SUMO_MAX_DATAQUEUE_LENGTH")
	maxConcurrentRequests := os.Getenv("SUMO_MAX_CONCURRENT_REQUESTS")
	enableFailover := os.Getenv("SUMO_ENABLE_FAILOVER")
	logTypes := os.Getenv("SUMO_LOG_TYPES")
	enhanceJsonLogs := os.Getenv("SUMO_ENHANCE_JSON_LOGS")
	enableSpanDrops := os.Getenv("SUMO_SPAN_DROP")
	kmsCacheSeconds := os.Getenv("KMS_CACHE_SECONDS")
	telemetryTimeoutMs := os.Getenv("TELEMETRY_TIMEOUT_MS")
	telemetryMaxBytes := os.Getenv("TELEMETRY_MAX_BYTES")
	telemetryMaxItems := os.Getenv("TELEMETRY_MAX_ITEMS")

	if telemetryTimeoutMs == "" {
		cfg.TelemetryTimeoutMs = 1000
	}

	if telemetryMaxBytes == "" {
		cfg.TelemetryMaxBytes = 262144
	}

	if telemetryMaxItems == "" {
		cfg.TelemetryMaxItems = 10000
	}

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
		cfg.LogTypes = defaultLogTypes
	} else {
		cfg.LogTypes = strings.Split(logTypes, ",")
	}

	if retrySleepTime == "" {
		cfg.RetrySleepTime = 300 * time.Millisecond
	}

	if enhanceJsonLogs == "" {
		cfg.EnhanceJsonLogs = true
	}

	if enableSpanDrops == "" {
		// by default, spans will not be dropped if user did not configure the env variable
		cfg.EnableSpanDrops = false
	}

	if kmsCacheSeconds == "" {
		cfg.KmsCacheSeconds = 5
	}
}

func (cfg *LambdaExtensionConfig) validateConfig() error {
	numRetry := os.Getenv("SUMO_NUM_RETRIES")
	logLevel := os.Getenv("SUMO_LOG_LEVEL")
	maxDataQueueLength := os.Getenv("SUMO_MAX_DATAQUEUE_LENGTH")
	maxConcurrentRequests := os.Getenv("SUMO_MAX_CONCURRENT_REQUESTS")
	enableFailover := os.Getenv("SUMO_ENABLE_FAILOVER")
	retrySleepTime := os.Getenv("SUMO_RETRY_SLEEP_TIME_MS")
	enhanceJsonLogs := os.Getenv("SUMO_ENHANCE_JSON_LOGS")
	enableSpanDrops := os.Getenv("SUMO_SPAN_DROP")
	kmsCacheSeconds := os.Getenv("KMS_CACHE_SECONDS")
	telemetryTimeoutMs := os.Getenv("TELEMETRY_TIMEOUT_MS")
	telemetryMaxBytes := os.Getenv("TELEMETRY_MAX_BYTES")
	telemetryMaxItems := os.Getenv("TELEMETRY_MAX_ITEMS")

	var allErrors []string
	var err error

	if cfg.SumoHTTPEndpoint == "" {
		allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT not set in environment variable")
	}

	// Todo test url valid
	if cfg.SumoHTTPEndpoint != "" && cfg.KMSKeyId == "" {
		_, err = url.ParseRequestURI(cfg.SumoHTTPEndpoint)
		if err != nil {
			allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT is not Valid")
		}
	}

	if enableFailover != "" {
		cfg.EnableFailover, err = strconv.ParseBool(enableFailover)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_ENABLE_FAILOVER: %v", err))
		}
	}

	if cfg.EnableFailover {
		if cfg.S3BucketName == "" {
			allErrors = append(allErrors, "SUMO_S3_BUCKET_NAME not set in environment variable")
		}
		if cfg.S3BucketRegion == "" {
			allErrors = append(allErrors, "SUMO_S3_BUCKET_REGION not set in environment variable")
		}
	}

	if numRetry != "" {
		customNumRetry, err := strconv.ParseInt(numRetry, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_NUM_RETRIES: %v", err))
		} else {
			cfg.NumRetry = int(customNumRetry)
		}
	}

	if retrySleepTime != "" {
		customRetrySleepTime, err := strconv.ParseInt(retrySleepTime, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_RETRY_SLEEP_TIME_MS: %v", err))
		} else {
			cfg.RetrySleepTime = time.Duration(customRetrySleepTime) * time.Millisecond
		}
	}

	if maxDataQueueLength != "" {
		customMaxDataQueueLength, err := strconv.ParseInt(maxDataQueueLength, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_MAX_DATAQUEUE_LENGTH: %v", err))
		} else {
			cfg.MaxDataQueueLength = int(customMaxDataQueueLength)
		}
	}

	if maxConcurrentRequests != "" {
		customMaxConcurrentRequests, err := strconv.ParseInt(maxConcurrentRequests, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_MAX_CONCURRENT_REQUESTS: %v", err))
		} else {
			cfg.MaxConcurrentRequests = int(customMaxConcurrentRequests)
		}
	}

	if logLevel != "" {
		customloglevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_LOG_LEVEL: %v", err))
		} else {
			cfg.LogLevel = customloglevel
		}
	}

	if enhanceJsonLogs != "" {
		cfg.EnhanceJsonLogs, err = strconv.ParseBool(enhanceJsonLogs)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_ENHANCE_JSON_LOGS: %v", err))
		}
	}

	if enableSpanDrops != "" {
		cfg.EnableSpanDrops, err = strconv.ParseBool(enableSpanDrops)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse SUMO_SPAN_DROP: %v", err))
		}
	}

	if kmsCacheSeconds != "" {
		cfg.KmsCacheSeconds, err = strconv.ParseInt(kmsCacheSeconds, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse KMS_CACHE_SECONDS: %v", err))
		}
	}

	if telemetryTimeoutMs != "" {
		telemetryTimeoutMs, err := strconv.ParseInt(telemetryTimeoutMs, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse TELEMETRY_TIMEOUT_MS: %v", err))
		} else {
			cfg.TelemetryTimeoutMs = int(telemetryTimeoutMs)
		}
		cfg.TelemetryTimeoutMs = max(cfg.TelemetryTimeoutMs, 25)
		cfg.TelemetryTimeoutMs = min(cfg.TelemetryTimeoutMs, 30000)
	}

	if telemetryMaxBytes != "" {
		cfg.TelemetryMaxBytes, err = strconv.ParseInt(telemetryMaxBytes, 10, 64)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse TELEMETRY_MAX_BYTES: %v", err))
		}
		cfg.TelemetryMaxBytes = max(cfg.TelemetryMaxBytes, 262144)
		cfg.TelemetryMaxBytes = min(cfg.TelemetryMaxBytes, 1048576)
	}

	if telemetryMaxItems != "" {
		telemetryMaxItems, err := strconv.ParseInt(telemetryMaxItems, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse TELEMETRY_MAX_ITEMS: %v", err))
		} else {
			cfg.TelemetryMaxItems = int(telemetryMaxItems)
		}
		cfg.TelemetryMaxItems = max(cfg.TelemetryMaxItems, 1000)
		cfg.TelemetryMaxItems = min(cfg.TelemetryMaxItems, 10000)
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
