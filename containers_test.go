package testcontainernetwork

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	_ "github.com/lib/pq"
	"github.com/mikebharris/testcontainernetwork-go/clients"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

const (
	wiremockHostname  = "wiremock"
	wiremockPort      = 8080
	ssmHostname       = "ssm"
	ssmPort           = 8080
	snsHostname       = "sns"
	snsPort           = 9911
	sqsHostname       = "sqs"
	sqsPort           = 9324
	sqsQueueName      = "sqs-queue"
	dynamoDbHostname  = "dynamodb"
	dynamoDbPort      = 8000
	dynamoDbTableName = "table"
	auroraHostname    = "aurora"
	auroraPort        = 5432
)

func TestDockerContainerNetwork(t *testing.T) {
	var steps steps
	steps.t = t

	suite := godog.TestSuite{
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			ctx.BeforeSuite(steps.startContainerNetwork)
			ctx.BeforeSuite(steps.initialiseDynamoDb)
			ctx.AfterSuite(steps.stopContainerNetwork)
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^the Lambda is triggered$`, steps.theLambdaIsTriggered)
			ctx.Step(`^the Wiremock endpoint is hit`, steps.theWiremockEndpointIsHit)
			ctx.Step(`^the Lambda writes the message to the log`, steps.theLambdaWritesTheMessageToTheLog)
			ctx.Step(`^the Lambda writes a message to the SQS queue`, steps.theLambdaWritesTheMessageToTheSqsQueue)
			ctx.Step(`^the Lambda sends a notification to the SNS topic`, steps.theLambdaSendsANotificationToTheSnsTopic)
			ctx.Step(`^the Lambda writes the message to DynamoDB$`, steps.theLambdaWritesTheMessageToDynamoDB)
			ctx.Step(`^the Lambda writes the message to the Aurora Postgres database$`, steps.theLambdaWritesTheMessageToTheRDSDatabase)
		},
		Options: &godog.Options{
			StopOnFailure: true,
			Strict:        true,
			Format:        "pretty",
			Paths:         []string{"features"},
			TestingT:      t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

type steps struct {
	networkOfDockerContainers NetworkOfDockerContainers
	lambdaContainer           LambdaDockerContainer
	wiremockContainer         WiremockDockerContainer
	ssmContainer              WiremockDockerContainer
	sqsContainer              SqsDockerContainer
	snsContainer              SnsDockerContainer
	dynamoDbContainer         DynamoDbDockerContainer
	auroraContainer           PostgresDockerContainer
	flywayContainer           FlywayDockerContainer
	t                         *testing.T
}

func (s *steps) startContainerNetwork() {
	s.wiremockContainer = WiremockDockerContainer{
		Config: WiremockDockerContainerConfig{
			Hostname:     wiremockHostname,
			Port:         wiremockPort,
			JsonMappings: "test-assets/wiremock/mappings",
		},
	}

	s.ssmContainer = WiremockDockerContainer{
		Config: WiremockDockerContainerConfig{
			Hostname:     ssmHostname,
			Port:         ssmPort,
			JsonMappings: "test-assets/ssm/mappings",
		},
	}

	wd, _ := os.Getwd()
	s.sqsContainer = SqsDockerContainer{
		Config: SqsDockerContainerConfig{
			Hostname:       sqsHostname,
			Port:           sqsPort,
			ConfigFilePath: path.Join(wd, "test-assets/sqs/elasticmq.conf"),
		},
	}
	s.snsContainer = SnsDockerContainer{
		Config: SnsDockerContainerConfig{
			Hostname:   snsHostname,
			Port:       snsPort,
			ConfigFile: path.Join(wd, "test-assets/sns/sns.json"),
		},
	}

	s.dynamoDbContainer = DynamoDbDockerContainer{
		Config: DynamoDbDockerContainerConfig{
			Hostname: dynamoDbHostname,
			Port:     dynamoDbPort,
		},
	}

	s.auroraContainer = PostgresDockerContainer{
		Config: PostgresDockerContainerConfig{
			Hostname: auroraHostname,
			Port:     auroraPort,
			Environment: map[string]string{
				"POSTGRES_PASSWORD": "password",
				"POSTGRES_USER":     "user",
				"POSTGRES_DB":       "database",
			},
		},
	}

	s.flywayContainer = FlywayDockerContainer{
		Config: FlywayDockerContainerConfig{
			Hostname:       "flyway",
			ConfigFilePath: path.Join(wd, "test-assets/postgres/flyway/conf"),
			SqlFilePath:    path.Join(wd, "test-assets/postgres/flyway/sql"),
		},
	}

	s.lambdaContainer = LambdaDockerContainer{
		Config: LambdaDockerContainerConfig{
			Hostname:   "lambda",
			Executable: "test-assets/lambda/main",
			Environment: map[string]string{
				"API_ENDPOINT":        fmt.Sprintf("http://%s:%d", wiremockHostname, wiremockPort),
				"SQS_ENDPOINT":        fmt.Sprintf("http://%s:%d", sqsHostname, sqsPort),
				"SQS_QUEUE_NAME":      sqsQueueName,
				"SNS_ENDPOINT":        fmt.Sprintf("http://%s:%d", snsHostname, snsPort),
				"SNS_TOPIC_ARN":       "arn:aws:sns:eu-west-1:12345678999:sns-topic",
				"DYNAMODB_HOSTNAME":   dynamoDbHostname,
				"DYNAMODB_PORT":       strconv.Itoa(dynamoDbPort),
				"DYNAMODB_TABLE_NAME": dynamoDbTableName,
				"SSM_ENDPOINT":        fmt.Sprintf("http://%s:%d", ssmHostname, ssmPort),
			},
		},
	}

	s.networkOfDockerContainers =
		NetworkOfDockerContainers{}.
			WithDockerContainer(&s.lambdaContainer).
			WithDockerContainer(&s.ssmContainer).
			WithDockerContainer(&s.wiremockContainer).
			WithDockerContainer(&s.sqsContainer).
			WithDockerContainer(&s.snsContainer).
			WithDockerContainer(&s.dynamoDbContainer).
			WithDockerContainer(&s.auroraContainer).
			WithDockerContainer(&s.flywayContainer)
	_ = s.networkOfDockerContainers.StartWithDelay(5 * time.Second)
}

func (s *steps) stopContainerNetwork() {
	buffer, _ := s.lambdaContainer.Log()
	log.Println("lambda log:", buffer.String())
	if err := s.networkOfDockerContainers.Stop(); err != nil {
		log.Fatalf("stopping docker containers: %v", err)
	}
}

func (s *steps) initialiseDynamoDb() {
	dynamoDbClient, err := clients.DynamoDbClient{}.New("localhost", s.dynamoDbContainer.MappedPort())
	if err != nil {
		log.Fatalf("creating DynamoDB client: %v", err)
	}

	i := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("Message"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("Message"),
				KeyType:       types.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(dynamoDbTableName),
	}
	if err = dynamoDbClient.CreateTable(i); err != nil {
		log.Fatalf("creating table: %v", err)
	}
}

func (s *steps) theLambdaIsTriggered() {
	request := events.APIGatewayProxyRequest{Path: fmt.Sprintf("/api-gateway-stage")}
	requestJsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("marshalling lambda request %v", err)
	}
	response, err := http.Post(s.lambdaContainer.InvocationUrl(), "application/json", bytes.NewReader(requestJsonBytes))
	if err != nil {
		log.Fatalf("triggering lambda: %v", err)
	}

	if response.StatusCode != 200 {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(response.Body); err != nil {
			panic(err)
		}
		body := buf.String()
		log.Fatalf("invoking Lambda: %d %s", response.StatusCode, body)
	}
}

func (s *steps) theWiremockEndpointIsHit() {
	adminStatus, _ := s.wiremockContainer.GetAdminStatus()
	var req WiremockAdminRequest
	for _, request := range adminStatus.Requests {
		if request.Request.AbsoluteUrl == fmt.Sprintf("http://%s:8080/", wiremockHostname) {
			req = request
			break
		}
	}
	if req.Request.AbsoluteUrl == "" {
		s.t.Errorf("unable to find matching call to the endpoint")
		s.t.Fail()
	}

	assert.Equal(s.t, http.StatusOK, req.ResponseDefinition.Status)
	assert.Equal(s.t, "GET", req.Request.Method)
}

func (s *steps) theLambdaWritesTheMessageToTheLog() {
	buffer, _ := s.lambdaContainer.Log()
	matched, err := regexp.Match("Wiremock returned a message of Hello World!", buffer.Bytes())
	if matched != true || err != nil {
		s.t.Errorf("Lambda log did not contain expected value. Expected: \"Wiremock returned a message of Hello World!\", Got: %s", buffer.String())
	}
}

func (s *steps) theLambdaWritesTheMessageToTheSqsQueue() {
	sqsClient, err := clients.SqsClient{}.New(s.sqsContainer.MappedPort())
	if err != nil {
		log.Fatalf("creating SQS client: %v", err)
	}

	messagesOnQueue, err := sqsClient.GetMessagesFrom(sqsQueueName)
	if err != nil {
		log.Fatalf("getting messages from SQS: %v", err)
	}
	assert.Equal(s.t, 1, len(messagesOnQueue))
	assert.Equal(s.t, "{\"message\":\"Hello World!\"}", *messagesOnQueue[0].Body)
}

func (s *steps) theLambdaSendsANotificationToTheSnsTopic() {
	message, err := s.snsContainer.GetMessage()
	if err != nil {
		log.Fatalf("getting message from SNS: %v", err)
	}
	assert.Equal(s.t, "{\"message\":\"Hello World!\"}", message)
}

func (s *steps) theLambdaWritesTheMessageToDynamoDB() {
	dynamoDbClient, err := clients.DynamoDbClient{}.New("localhost", s.dynamoDbContainer.MappedPort())
	if err != nil {
		log.Fatalf("creating DynamoDB client: %v", err)
	}

	items, err := dynamoDbClient.GetItemsInTable(dynamoDbTableName)
	if err != nil {
		log.Fatalf("scanning DynamoDB: %v", err)
	}

	assert.Equal(s.t, 1, len(items))

	type Message struct {
		Message string `json:"message"`
	}
	var message Message
	if err = json.Unmarshal([]byte(items[0]["Message"].(*types.AttributeValueMemberS).Value), &message); err != nil {
		log.Fatalf("unmarshalling message: %v", err)
	}

	assert.Equal(s.t, "Hello World!", message.Message)
}

func (s *steps) theLambdaWritesTheMessageToTheRDSDatabase() {
	dbConx, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s search_path=%s sslmode=disable",
		"localhost", s.auroraContainer.MappedPort(), "user", "password", "database", "database",
	))
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}

	rows, err := dbConx.Query("SELECT message FROM database.messages")
	if err != nil {
		log.Fatalf("querying database: %v", err)
	}
	if !rows.Next() {
		log.Fatalf("no rows returned from database")
	}

	var message string
	if err = rows.Scan(&message); err != nil {
		log.Fatalf("scanning rows: %v", err)
	}

	assert.Equal(s.t, "Hello World!", message)

}
