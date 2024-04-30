package clients

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"os"
)

type SqsClient struct {
	handle *sqs.Client
}

func (s SqsClient) New(port int) (SqsClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return SqsClient{}, fmt.Errorf("loading config: %v", err)
	}
	s.handle = sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%d", "localhost", port))
	})
	return s, nil
}

func (s SqsClient) GetMessagesFrom(queue string) ([]types.Message, error) {
	queueUrlOutput, err := s.handle.GetQueueUrl(
		context.Background(),
		&sqs.GetQueueUrlInput{
			QueueName: aws.String(queue),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("getting queue url: %v", err)
	}

	receiveMessageOutput, err := s.handle.ReceiveMessage(
		context.Background(),
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueUrlOutput.QueueUrl,
			MaxNumberOfMessages: 10,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("receiving messages: %v", err)
	}
	return receiveMessageOutput.Messages, nil
}
