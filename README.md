# Test Container Network for Go

![AI-generated test pyramid for gophers](doc/ai-test-pyramid.png)

A _library_ of helper types and methods to abstract out boilerplate code
for writing integration- and system-level tests for AWS services using [testcontainers in Go](https://github.com/testcontainers/testcontainers-go).  The tests are implemented using the Gherkin syntax and the [Godog](https://github.com/cucumber/godog) test runner. See the Cucumber feature file in [features](features) directory.

Currently implemented are:
 
* __DynamoDbDockerContainer__ - a container for a DynamoDB data store
* __FlywayDockerContainer__ - a Flyway container for provisioning database containers
* __LambdaDockerContainer__ - a container that runs a Lambda function
* __PostgresDockerContainer__ - a container for an Postgres Database in AWS RDS
* __SnsDockerContainer__ - a container that runs an AWS Simple Notification Service server
* __SqsDockerContainer__ - a container that runs an AWS Simple Queue Service server
* __WiremockDockerContainer__ - a container that runs a Wiremock server, great for simulating external APIs and AWS's Systems Manager Parameter Store

I presented this library to the [Golang Oxford MeetUp](https://www.meetup.com/Golang-Oxford) group on [5th June 2024](https://www.meetup.com/golang-oxford/events/300287783) to pretty positive feedback from the attendees.  One attendee did query why I created this rather than just using Dockerfiles.  The following is my rationale.

## Rationale

I wanted to have a test-driven way of working with the containers and writing the code and this is why I created this library.  I also wanted a little Go project to get my teeth into and show my coding style and I was keen for this project to have a real use, a real purpose.  But it was a good point that the attendee made, but there are many ways to skin the proverbial cat, and this is one of them.  It may not be subjectively the _best_ and certainly might be more complex than some others. 

I am going to willfully paraphrase from Ham Vocke, who sums up perfectly why I wanted to create this library in [The Practical Test Pyramid](https://martinfowler.com/articles/practical-test-pyramid.html#IntegrationTests):

_"All non-trivial applications [in this case an AWS Lambda] will integrate with some other parts (databases, filesystems, network calls to other applications). When writing unit tests these are usually the parts you leave out in order to come up with better isolation and faster tests. Still, your application will interact with other parts and this needs to be tested. Integration Tests are there to help. They test the integration of your application with all the parts that live outside of your application._

_For your automated tests this means you don't just need to run your own application but also the component you're integrating with. If you're testing the integration with a database you need to run a database when running your tests. For testing that you can read files from a disk you need to save a file to your disk and load it in your integration test."_

Referring to the test pyramid, integration tests sit in the middle, and the term _service tests_ is often used interchangeably:

![A typical test pyramid](doc/testpyramid.png)

I go with that. 

We are generally testing a service's integration between itself and other external services, some of which we may have also written ourselves.  In this example, we have a single AWS Lambda service that interacts with a number of other AWS services and an external API service (non-AWS).  A diagram of the Lambda service under test probably indicates this the best:

![The Lambda service under test](doc/service-under-test.svg)

## Testing

You will need to have some credentials in your AWS credentials file.  You can set them up using the AWS CLI tool thus (the values are relatively arbitrary for this):

```shell
aws configure
AWS Access Key ID [None]: X
AWS Secret Access Key [None]: Y
Default region name [None]: us-east-1
Default output format [None]: 
```

```shell
make test
```

The tests can also be run directly from within an IDE, such as GoLand, by running the func _TestDockerContainerNetwork_ in [containers_test.go](containers_test.go).

## Using

I have _playground_ project that makes extensive use of this library for running it's tests.  See [CLAMS](https://github.com/mikebharris/CLAMS) for some relatively real-world, realistic examples of its use.

```go
package main

import "github.com/mikebharris/testcontainernetwork-go"
import "time"

func main() {
	wiremockContainer := WiremockDockerContainer{
		Config: WiremockDockerContainerConfig{
			Hostname:     wiremockHostname,
			Port:         wiremockPort,
			JsonMappings: "test-assets/wiremock/mappings",
		},
	}

	lambdaContainer := LambdaDockerContainer{
		Config: LambdaDockerContainerConfig{
			Hostname:   "lambda",
			Executable: "path/to/lambda/bootable",
			Environment: map[string]string{
				"API_ENDPOINT":        fmt.Sprintf("http://%s:%d", wiremockHostname, wiremockPort),
			},
		},
	}

	networkOfDockerContainers :=
		NetworkOfDockerContainers{}.
			WithDockerContainer(&lambdaContainer).
			WithDockerContainer(&wiremockContainer)
	
	err := networkOfDockerContainers.StartWithDelay(5 * time.Second)
	if err != nil {
        log.Fatalf("starting network of Docker containers: %v", err)
    }
}
```

## Clients

There is a client for some container types that provides a simple way to interact with the container. For example, the SQS client provides methods to receive messages from the SQS server:

```go
package main

import "github.com/mikebharris/testcontainernetwork-go/clients"

func main() {
	sqsClient, _ := clients.SqsClient{}.New(s.sqsContainer.MappedPort())
	if err != nil {
		log.Fatalf("creating SQS client: %v", err)
	}

	messagesOnQueue, err := sqsClient.GetMessagesFrom(sqsQueueName)
	if err != nil {
		log.Fatalf("getting messages from SQS: %v", err)
	}
}
```

## Implementing a new container

The container should _promote_ the values and methods of _DockerContainer_ and implement the
_StartableDockerContainer_ interface, which gets you the internal reference to the testcontainer and the port for the
service, as well as the _MappedPort()_ method. For example:

```go

type MyDockerContainerConfig struct {
    Hostname   string
    ConfigFile string
    Port       int
}

type MyDockerContainer struct {
    DockerContainer
    Config MyDockerContainerConfig
}

func (c *MyDockerContainer) Stop(ctx context.Context) error {
    ....
}

func (c *LambdaDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
    ....
}

```

I recommend that container-specific configuration parameters be assigned as a struct to the Config field, thus keeping
the container struct itself clean and simple and the same across different container implementations.