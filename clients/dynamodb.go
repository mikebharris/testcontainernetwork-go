package clients

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"os"
)

type IDynamoDbClient interface {
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type DynamoDbClient struct {
	handle IDynamoDbClient
}

func (c DynamoDbClient) New(hostname string, port int) (DynamoDbClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return DynamoDbClient{}, fmt.Errorf("loading config: %v", err)
	}
	c.handle = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%d", hostname, port))
	})
	return c, nil
}

func (c DynamoDbClient) GetItemsInTable(table string) ([]map[string]types.AttributeValue, error) {
	scanOutput, err := c.handle.Scan(context.Background(), &dynamodb.ScanInput{
		TableName: aws.String(table),
	})
	if err != nil {
		return nil, fmt.Errorf("scanning DynamoDB: %v", err)
	}
	return scanOutput.Items, nil
}

func (c DynamoDbClient) CreateTable(input *dynamodb.CreateTableInput) error {
	_, err := c.handle.CreateTable(context.Background(), input)
	return err
}

func (c DynamoDbClient) PutItem(input *dynamodb.PutItemInput) error {
	_, err := c.handle.PutItem(context.Background(), input)
	return err
}

func (c DynamoDbClient) PutObject(table string, object interface{}) error {
	m, err := attributevalue.MarshalMap(object)
	if err != nil {
		return err
	}

	if err := c.PutItem(&dynamodb.PutItemInput{
		Item:      m,
		TableName: aws.String(table),
	}); err != nil {
		return err
	}
	return nil
}
