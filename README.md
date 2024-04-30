# Test Container Network for Go

A set of helper types and methods to abstract out boilerplate code
for [testcontainers in Go](https://github.com/testcontainers/testcontainers-go). Currently supported are:

* LambdaDockerContainer - a container that runs a Lambda function
* WiremockDockerContainer - a container that runs a Wiremock server
* SqsDockerContainer - a container that runs an SQS server

## Testing

```shell
make test
```

## Usage

```go
package main

import "github.com/mikebharris/testcontainernetwork"
import "time"
import "fmt"

func main() {
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
```

## Implementing a new container

The container should _promote_ the values and methods of _GenericDockerContainer_ and implement the
_StartableDockerContainer_ interface, which gets you the internal reference to the testcontainer and the port for the
service, as well as the _MappedPort()_ method. For example:

```go
type MyDockerContainer struct {
GenericDockerContainer
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

```go