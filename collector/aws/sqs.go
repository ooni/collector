package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	// QueueURL is the URL of the message queue
	queueURL = "https://sqs.us-east-2.amazonaws.com/082866812839/ooni-collector.fifo"
)

// SendMessage sends a message to the AWS SQS queue
func SendMessage(sess *session.Session, body string, groupID string) (*string, error) {
	if sess == nil {
		return nil, errors.New("invalid aws Session")
	}

	svc := sqs.New(sess)
	sendParams := &sqs.SendMessageInput{
		MessageBody:    aws.String(body),
		QueueUrl:       aws.String(queueURL),
		MessageGroupId: aws.String(groupID),
	}
	sendResp, err := svc.SendMessage(sendParams)

	if err != nil {
		return nil, err
	}

	return sendResp.MessageId, nil
}
