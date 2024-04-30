package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	lambda.Start(func() error {
		resp, err := http.Get(os.Getenv("API_ENDPOINT"))
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("expected status %v, got status %v", http.StatusOK, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("reading respons body: %v", err)
		}

		var returnedJson struct {
			Message string `json:"message"`
		}
		if err = json.Unmarshal(body, &returnedJson); err != nil {
			log.Fatalf("unmarshalling %s: %v", body, err)
		}

		log.Printf("Wiremock returned a message of %s", returnedJson.Message)

		sqsClient := newSqsClient()
		marshal, err := json.Marshal(returnedJson)
		if err != nil {
			log.Fatalf("marshalling %s: %v", returnedJson, err)
		}
		if _, err = sqsClient.SendMessage(context.Background(), &sqs.SendMessageInput{
			MessageBody: aws.String(string(marshal)),
			QueueUrl:    aws.String(queueUrl(sqsClient)),
		}); err != nil {
			log.Fatalf("sending message: %v", err)
		}
		return nil
	})
}

func queueUrl(client *sqs.Client) string {
	queueUrl, err := client.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(os.Getenv("SQS_QUEUE_NAME")),
	})
	if err != nil {
		log.Fatalf("getting queue url: %v", err)
	}
	return *queueUrl.QueueUrl
}

func newSqsClient() *sqs.Client {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("SQS_ENDPOINT"))
	})
	return client
}
