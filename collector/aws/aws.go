package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	// Region is the default aws region
	Region = "us-east-2"
	// MaxRetries is the number of retries when connecting to aws
	MaxRetries = 5
)

// Session is a global session handle
var Session *session.Session

// NewSession creates a new aws session
func NewSession(accessKeyID, secretAccessKey string) *session.Session {
	return session.New(&aws.Config{
		Region:      aws.String(Region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		MaxRetries:  aws.Int(MaxRetries),
	})
}
