package utils

import (
	"context"
	"io"
	"os"
    "fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var uploader *manager.Uploader

func init() {
	awsRegion, found := os.LookupEnv("SUMO_S3_BUCKET_REGION")
	if !found {
		awsRegion = os.Getenv("AWS_REGION")
	}
    fmt.Println("awsRegion:-->", awsRegion)

	// Load AWS default config with region
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		panic("unable to load AWS SDK config, " + err.Error())
	}
    fmt.Printf("CFG --> %+v\n", cfg)

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Create uploader
	uploader = manager.NewUploader(s3Client)
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
