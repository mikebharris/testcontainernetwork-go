package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	_ "github.com/lib/pq"
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
		message.writeToDatabase(getDbUrl())
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

func (m message) writeToDatabase(dbUrl string) {
	statement := `insert into database.messages(message) values($1)`
	log.Println("Writing to database using statement: ", statement)

	var message struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(m), &message); err != nil {
		log.Fatalf("unmarshalling %s: %v", m, err)
	}
	if _, err := databaseClient(dbUrl).Exec(statement, message.Message); err != nil {
		log.Fatalf("writing to database: %v", err)
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
		o.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%s", os.Getenv("DYNAMODB_HOSTNAME"), os.Getenv("DYNAMODB_PORT")))
	})
}

func databaseClient(dbUrl string) *sql.DB {
	dbConx, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("opening database connection: %v", err)
	}
	return dbConx
}

func getDbUrl() string {
	response, err := parameterStoreClient().GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           aws.String("/db-url"),
		WithDecryption: aws.Bool(true)},
	)
	if err != nil {
		panic(fmt.Errorf("getting SSM parameter %s: %v", "/db-url", err))
	}

	dbUrl := *response.Parameter.Value
	return dbUrl
}

func parameterStoreClient() *ssm.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	return ssm.NewFromConfig(cfg, func(o *ssm.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("SSM_ENDPOINT"))
	})
}
