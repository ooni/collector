package aws

import (
	"os"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// UploadFile will upload the srcPath to the target bucket with the key
func UploadFile(sess *session.Session, srcPath string, bucket string, key string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return err
	}
	log.Debugf("uploaded %s to %s/%s", srcPath, bucket, key)
	return nil
}
