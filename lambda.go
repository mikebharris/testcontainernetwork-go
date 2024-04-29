package testcontainernetwork

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"log"
)

type LambdaDockerContainer struct {
	GenericDockerContainer
	Config LambdaDockerContainerConfig
}

func (c *LambdaDockerContainer) Stop(ctx context.Context) error {
	return c.dockerContainer.Terminate(ctx)
}

func (c *LambdaDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	if c.Config.Hostname == "" {
		c.Config.Hostname = "lambda"
	}
	c.internalServicePort = 9001
	req := testcontainers.ContainerRequest{
		Image:        "lambci/lambda:go1.x",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Env:          c.setupEnvironment(),
		Networks:     []string{dockerNetwork.Name},
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
		},
	}
	var err error
	c.dockerContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		panic(err)
	}
	if err = c.dockerContainer.CopyFileToContainer(ctx, c.Config.Executable, "/var/task/handler", 365); err != nil {
		log.Fatalf("copying binary to docker container: %v", err)
	}
	if err := c.dockerContainer.Start(ctx); err != nil {
		panic(err)
	}
	return nil
}

func (c *LambdaDockerContainer) setupEnvironment() map[string]string {
	env := map[string]string{
		"ENVIRONMENT":             "dev",
		"AWS_REGION":              "eu-west-1",
		"DOCKER_LAMBDA_STAY_OPEN": "1",
		"AWS_ACCESS_KEY_ID":       "x",
		"AWS_SECRET_ACCESS_KEY":   "x",
	}
	for k, v := range c.Config.Environment {
		env[k] = v
	}
	return env
}

func (c *LambdaDockerContainer) Log() *bytes.Buffer {
	logs, err := c.dockerContainer.Logs(context.Background())
	if err != nil {
		log.Fatalf("getting Lambda logs: %s", err)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(logs); err != nil {
		log.Fatalf("Reading log from Lambda container %s", err)
	}
	return buf
}

func (c *LambdaDockerContainer) InvocationUrl() string {
	return fmt.Sprintf("http://localhost:%d/2015-03-31/functions/myfunction/invocations", c.MappedPort())
}

type LambdaDockerContainerConfig struct {
	Executable  string
	Environment map[string]string
	Hostname    string
}
