package utils

import (
	"context"
	"io"
	"os"
    "fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var uploader *manager.Uploader

func init() {
	awsRegion, found := os.LookupEnv("SUMO_S3_BUCKET_REGION")
	if !found {
		awsRegion = os.Getenv("AWS_REGION")
	}

	// Load AWS default config with region
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		panic("unable to load AWS SDK config, " + err.Error())
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Create uploader
	uploader = manager.NewUploader(s3Client)

	// Create STS client to get account ID
    stsClient := sts.NewFromConfig(cfg)
    // Get caller identity
    identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
    if err != nil {
        panic("unable to get AWS caller identity: " + err.Error())
    }

    // Print complete identity
    fmt.Println("AWS Caller Identity:")
    fmt.Println("Account ID:", *identity.Account)
    fmt.Println("ARN       :", *identity.Arn)
    fmt.Println("UserID    :", *identity.UserId)
}

// UploadToS3 sends data to S3
func UploadToS3(bucketName *string, keyName *string, data io.Reader) error {
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: bucketName,
		Key:    keyName,
		Body:   data,
	})
	return err
}
