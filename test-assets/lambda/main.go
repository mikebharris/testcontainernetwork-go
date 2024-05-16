package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	lambda.Start(func() error {
		message := getMessageFromEndpoint(os.Getenv("API_ENDPOINT"))
		message.sendToSqsQueue(os.Getenv("SQS_QUEUE_NAME"))
		message.sendToSnsTopic(os.Getenv("SNS_TOPIC_ARN"))
		message.writeToDynamoDbTable(os.Getenv("DYNAMODB_TABLE_NAME"))
		return nil
	})
}

type message string

func (m message) sendToSqsQueue(queue string) {
	sqsClient := sqsClient()
	queueUrl, err := sqsClient.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(queue),
	})
	if err != nil {
		log.Fatalf("getting queue url: %v", err)
	}
	if _, err := sqsClient.SendMessage(context.Background(), &sqs.SendMessageInput{
		MessageBody: aws.String(string(m)),
		QueueUrl:    aws.String(*queueUrl.QueueUrl),
	}); err != nil {
		log.Fatalf("sending message: %v", err)
	}
}

func (m message) sendToSnsTopic(topic string) {
	if _, err := snsClient().Publish(context.Background(), &sns.PublishInput{
		Message:  aws.String(string(m)),
		TopicArn: aws.String(topic),
	}); err != nil {
		log.Fatalf("publishing message: %v", err)
	}
}

func (m message) writeToDynamoDbTable(table string) {
	if _, err := dynamoDbClient().PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item: map[string]types.AttributeValue{
			"Message": &types.AttributeValueMemberS{Value: string(m)},
		},
	}); err != nil {
		log.Fatalf("writing to dynamodb: %v", err)
	}
}

func getMessageFromEndpoint(endpoint string) message {
	resp, err := http.Get(endpoint)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("expected status %v, got status %v", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("reading response body: %v", err)
	}

	var returnedJson struct {
		Message string `json:"message"`
	}
	if err = json.Unmarshal(body, &returnedJson); err != nil {
		log.Fatalf("unmarshalling %s: %v", body, err)
	}

	log.Printf("Wiremock returned a message of %s", returnedJson.Message)

	msg, err := json.Marshal(returnedJson)
	if err != nil {
		log.Fatalf("marshalling %s: %v", returnedJson, err)
	}
	return message(msg)
}

func sqsClient() *sqs.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	return sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("SQS_ENDPOINT"))
	})
}

func snsClient() *sns.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	client := sns.NewFromConfig(cfg, func(o *sns.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("SNS_ENDPOINT"))
	})
	return client
}

func dynamoDbClient() *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("DYNAMODB_ENDPOINT"))
	})
}
