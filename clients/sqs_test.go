package clients

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*sqs.GetQueueUrlOutput), args.Error(1)
}

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func TestSqsClient_GetMessagesFrom(t *testing.T) {
	mockClient := new(MockSQSClient)
	sqsClient := SqsClient{handle: mockClient}
	queue := "testQueue"
	messages := []types.Message{{}}

	mockClient.On("GetQueueUrl", mock.Anything, &sqs.GetQueueUrlInput{QueueName: &queue}).Return(&sqs.GetQueueUrlOutput{QueueUrl: &queue}, nil)
	mockClient.On("ReceiveMessage", mock.Anything, &sqs.ReceiveMessageInput{QueueUrl: &queue, MaxNumberOfMessages: 10}).Return(&sqs.ReceiveMessageOutput{Messages: messages}, nil)

	result, err := sqsClient.GetMessagesFrom(queue)

	assert.NoError(t, err)
	assert.Equal(t, messages, result)
	mockClient.AssertExpectations(t)
}

func TestSqsClient_GetMessagesFrom_GetQueueUrlReturnsError(t *testing.T) {
	mockClient := new(MockSQSClient)
	sqsClient := SqsClient{handle: mockClient}
	queue := "testQueue"

	mockClient.On("GetQueueUrl", mock.Anything, &sqs.GetQueueUrlInput{QueueName: &queue}).Return(&sqs.GetQueueUrlOutput{}, errors.New("error"))

	result, err := sqsClient.GetMessagesFrom(queue)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

func TestSqsClient_GetMessagesFrom_ReceiveMessagesReturnsError(t *testing.T) {
	mockClient := new(MockSQSClient)
	sqsClient := SqsClient{handle: mockClient}
	queue := "testQueue"

	mockClient.On("GetQueueUrl", mock.Anything, &sqs.GetQueueUrlInput{QueueName: &queue}).Return(&sqs.GetQueueUrlOutput{QueueUrl: &queue}, nil)
	mockClient.On("ReceiveMessage", mock.Anything, &sqs.ReceiveMessageInput{QueueUrl: &queue, MaxNumberOfMessages: 10}).Return(&sqs.ReceiveMessageOutput{}, errors.New("error"))

	result, err := sqsClient.GetMessagesFrom(queue)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}
