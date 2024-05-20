package clients

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.ScanOutput), args.Error(1)
}

func (m *MockDynamoDBClient) CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.CreateTableOutput), args.Error(1)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func TestDynamoDbClient_GetItemsInTable(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	table := "testTable"
	items := []map[string]types.AttributeValue{{"Key": &types.AttributeValueMemberS{Value: "Value"}}}

	mockClient.On("Scan", mock.Anything, &dynamodb.ScanInput{TableName: &table}).Return(&dynamodb.ScanOutput{Items: items}, nil)

	result, err := dynamoDbClient.GetItemsInTable(table)

	assert.NoError(t, err)
	assert.Equal(t, items, result)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_GetItemsInTable_Error(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	table := "testTable"

	mockClient.On("Scan", mock.Anything, &dynamodb.ScanInput{TableName: &table}).Return(&dynamodb.ScanOutput{}, errors.New("error"))

	result, err := dynamoDbClient.GetItemsInTable(table)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_CreateTable(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.CreateTableInput{TableName: aws.String("testTable")}

	mockClient.On("CreateTable", mock.Anything, input).Return(&dynamodb.CreateTableOutput{}, nil)

	err := dynamoDbClient.CreateTable(input)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_CreateTable_Error(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.CreateTableInput{TableName: aws.String("testTable")}

	mockClient.On("CreateTable", mock.Anything, input).Return(&dynamodb.CreateTableOutput{}, errors.New("error"))

	err := dynamoDbClient.CreateTable(input)

	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_PutItem(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.PutItemInput{TableName: aws.String("testTable")}

	mockClient.On("PutItem", mock.Anything, input).Return(&dynamodb.PutItemOutput{}, nil)

	err := dynamoDbClient.PutItem(input)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_PutItem_Error(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.PutItemInput{TableName: aws.String("testTable")}

	mockClient.On("PutItem", mock.Anything, input).Return(&dynamodb.PutItemOutput{}, errors.New("error"))

	err := dynamoDbClient.PutItem(input)

	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_PutObject(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.PutItemInput{TableName: aws.String("testTable"), Item: map[string]types.AttributeValue{"Key": &types.AttributeValueMemberS{Value: "Value"}}}

	mockClient.On("PutItem", mock.Anything, input).Return(&dynamodb.PutItemOutput{}, nil)

	err := dynamoDbClient.PutObject("testTable", struct{ Key string }{Key: "Value"})

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDynamoDbClient_PutObject_Error(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	dynamoDbClient := DynamoDbClient{handle: mockClient}
	input := &dynamodb.PutItemInput{TableName: aws.String("testTable"), Item: map[string]types.AttributeValue{"Key": &types.AttributeValueMemberS{Value: "Value"}}}

	mockClient.On("PutItem", mock.Anything, input).Return(&dynamodb.PutItemOutput{}, errors.New("error"))

	err := dynamoDbClient.PutObject("testTable", struct{ Key string }{Key: "Value"})

	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}
