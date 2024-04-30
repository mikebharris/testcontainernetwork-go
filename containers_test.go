package testcontainernetwork

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

const wiremockHostname = "wiremock"

func TestDockerContainerNetwork(t *testing.T) {
	var steps steps
	steps.t = t

	suite := godog.TestSuite{
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			ctx.BeforeSuite(steps.startContainerNetwork)
			ctx.AfterSuite(steps.stopContainerNetwork)
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^the Lambda is triggered$`, steps.theLambdaIsTriggered)
			ctx.Step(`^the Wiremock endpoint is hit`, steps.theWiremockEndpointIsHit)
			ctx.Step(`^the Lambda writes the message to the log`, steps.theLambdaWritesTheMessageToTheLog)
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
	t                         *testing.T
}

func (s *steps) startContainerNetwork() {
	s.lambdaContainer = LambdaDockerContainer{
		Config: LambdaDockerContainerConfig{
			Executable:  "test-assets/lambda/main",
			Environment: map[string]string{"API_ENDPOINT": fmt.Sprintf("http://%s:8080", wiremockHostname)},
			Hostname:    "lambda",
		},
	}
	s.wiremockContainer = WiremockDockerContainer{
		Config: WiremockDockerContainerConfig{
			JsonMappings: "test-assets/wiremock/mappings",
			Hostname:     wiremockHostname,
		},
	}
	s.networkOfDockerContainers =
		NetworkOfDockerContainers{}.
			WithDockerContainer(&s.lambdaContainer).
			WithDockerContainer(&s.wiremockContainer).
			StartWithDelay(2 * time.Second)
}

func (s *steps) stopContainerNetwork() {
	if err := s.networkOfDockerContainers.Stop(); err != nil {
		log.Fatalf("stopping docker containers: %v", err)
	}
}

func (s *steps) theLambdaIsTriggered() {
	localLambdaInvocationPort := s.lambdaContainer.MappedPort()
	url := fmt.Sprintf("http://localhost:%d/2015-03-31/functions/myfunction/invocations", localLambdaInvocationPort)

	request := events.APIGatewayProxyRequest{Path: fmt.Sprintf("/api-gateway-stage")}
	requestJsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("marshalling lambda request %v", err)
	}
	response, err := http.Post(url, "application/json", bytes.NewReader(requestJsonBytes))
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
	adminStatus := s.wiremockContainer.GetAdminStatus()
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
	buffer := s.lambdaContainer.Log()
	matched, err := regexp.Match("Wiremock returned a message of Hello World!", buffer.Bytes())
	if matched != true || err != nil {
		s.t.Errorf("Lambda log did not contain expected value. Expected: \"Wiremock returned a message of Hello World!\", Got: %s", buffer.String())
	}
}
