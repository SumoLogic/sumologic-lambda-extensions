package utils

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var uploader *s3manager.Uploader
var sess *session.Session

func init() {
	os.Setenv("AWS_PROFILE", "prod")
	os.Setenv("AWS_REGION", "us-east-1")

	var awsRegion, found = os.LookupEnv("S3_BUCKET_REGION")

	if !found {
		awsRegion = os.Getenv("AWS_REGION")
	}

	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)}))

	// Create an uploader with the session and default options
	uploader = s3manager.NewUploader(sess)

}

// UploadToS3 send data to S3
func UploadToS3(bucketName *string, keyName *string, data io.Reader) error {

	upParams := &s3manager.UploadInput{
		Bucket: bucketName,
		Key:    keyName,
		Body:   data,
	}
	_, err := uploader.Upload(upParams)

	return err
}
