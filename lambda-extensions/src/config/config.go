package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"strconv"
)

type LambdaExtensionConfig struct {
	SumoHTTPEndpoint    string
	EnableFailover      bool
	S3BucketName        string
	MaxRetry            int64
	AWSLambdaRuntimeAPI string
}

func GetConfig() (*LambdaExtensionConfig, error) {

	config := &LambdaExtensionConfig{
		SumoHTTPEndpoint: os.Getenv("SUMO_HTTP_ENDPOINT"),
		S3BucketName: os.Getenv("S3_BUCKET_NAME"),
		AWSLambdaRuntimeAPI: os.Getenv("AWS_LAMBDA_RUNTIME_API"),
	}

	err := (*config).SetDefaults()

	err = (*config).ValidateConfig()
	fmt.Println(err, err == nil)
	if err == nil {
		return config, nil
	} else {
		return nil, err
	}
}
func (cfg *LambdaExtensionConfig) SetDefaults()  error {
	maxRetry := os.Getenv("MAX_RETRY")
	enableFailover := os.Getenv("ENABLE_FAILOVER")
	var err  error
	if maxRetry == "" {
		cfg.MaxRetry = 3
	} else {
		cfg.MaxRetry,  err  = strconv.ParseInt(maxRetry, 10,  32)
	}
	if enableFailover == "" {
		cfg.EnableFailover = false
	} else {
		cfg.EnableFailover,  err  = strconv.ParseBool(enableFailover)
	}
	if cfg.AWSLambdaRuntimeAPI == "" {
		cfg.AWSLambdaRuntimeAPI = "127.0.0.1:9001"
	}
	return err
}

func (cfg *LambdaExtensionConfig) ValidateConfig() error {
	var allErrors []string
	var err error
	if cfg.SumoHTTPEndpoint == "" {
		allErrors = append(allErrors, "SUMO_HTTP_ENDPOINT not set in environment variable")
	}

	// test url valid

	if cfg.EnableFailover == true && cfg.S3BucketName == "" {
		allErrors = append(allErrors, "S3_BUCKET_NAME not set in environment variable")
	}
	if allErrors != nil {
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
