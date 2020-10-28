package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"utils"
)

type LambdaExtensionConfig struct {
	SumoHTTPEndpoint    string
	EnableFailover      bool
	S3BucketName        string
	S3BucketRegion      string
	MaxRetry            int64
	AWSLambdaRuntimeAPI string
	LogTypes            string
	FunctionName        string
	FunctionVersion     string
}

var validLogTypes = []string{"platform", "function"}

// GetConfig to get config instance
func GetConfig() (*LambdaExtensionConfig, error) {

	config := &LambdaExtensionConfig{
		SumoHTTPEndpoint:    os.Getenv("SUMO_HTTP_ENDPOINT"),
		S3BucketName:        os.Getenv("S3_BUCKET_NAME"),
		S3BucketRegion:      os.Getenv("S3_BUCKET_REGION"),
		AWSLambdaRuntimeAPI: os.Getenv("AWS_LAMBDA_RUNTIME_API"),
		FunctionName:        os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
		FunctionVersion:     os.Getenv("AWS_LAMBDA_FUNCTION_VERSION"),
	}

	err := (*config).setDefaults()

	err = (*config).validateConfig()
	fmt.Println(err, err == nil)
	if err == nil {
		return config, nil
	} else {
		return nil, err
	}
}
func (cfg *LambdaExtensionConfig) setDefaults() error {
	maxRetry := os.Getenv("MAX_RETRY")
	enableFailover := os.Getenv("ENABLE_FAILOVER")
	logTypes := os.Getenv("LOG_TYPES")
	var err error
	if maxRetry == "" {
		cfg.MaxRetry = 3
	}
	if enableFailover == "" {
		cfg.EnableFailover = false
	}
	if cfg.AWSLambdaRuntimeAPI == "" {
		cfg.AWSLambdaRuntimeAPI = "127.0.0.1:9001"
	}
	if logTypes == "" {
		cfg.LogTypes = strings.Join(validLogTypes, ",")
	}

	return err
}

func (cfg *LambdaExtensionConfig) validateConfig() error {
	maxRetry := os.Getenv("MAX_RETRY")
	enableFailover := os.Getenv("ENABLE_FAILOVER")
	var allErrors []string
	var err error

	if cfg.SumoHTTPEndpoint == "" {
		allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT not set in environment variable")
	}

	// Todo test url valid
	if cfg.SumoHTTPEndpoint != "" {
		_, err = url.ParseRequestURI("http://google.com/")
		if err != nil {
			allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT is not Valid")
		}
	}

	if enableFailover != "" {
		cfg.EnableFailover, err = strconv.ParseBool(enableFailover)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse EnableFailover: %v", err))
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

	if maxRetry != "" {
		cfg.MaxRetry, err = strconv.ParseInt(maxRetry, 10, 32)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Unable to parse MaxRetry: %v", err))
		}

	}

	// test valid log format type
	logTypes := strings.Split(cfg.LogTypes, ",")
	for _, logType := range logTypes {
		if !utils.StringInSlice(strings.TrimSpace(logType), validLogTypes) {
			allErrors = append(allErrors, fmt.Sprintf("logType %s is unsupported logtype", logType))
		}
	}

	if len(allErrors) > 0 {
		err = errors.New(strings.Join(allErrors, ", "))
	}

	return err
}

// func main() {
// 	os.Setenv("SUMO_HTTP_ENDPOINT", "http://sumo")
// 	os.Setenv("ENABLE_FAILOVER", "True")
// 	os.Setenv("S3_BUCKET_NAME", "blaha")
// 	cfg, _ := GetConfig()
// 	fmt.Println(cfg)
// }
