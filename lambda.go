package testcontainernetwork

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

type LambdaDockerContainerConfig struct {
	Executable  string
	Hostname    string
	Environment map[string]string
}

type LambdaDockerContainer struct {
	DockerContainer
	Config LambdaDockerContainerConfig
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
	c.testContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	if err = c.testContainer.CopyFileToContainer(ctx, c.Config.Executable, "/var/task/handler", 365); err != nil {
		return fmt.Errorf("copying binary to docker container: %v", err)
	}
	if err := c.testContainer.Start(ctx); err != nil {
		return fmt.Errorf("starting container: %w", err)
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

func (c *LambdaDockerContainer) Log() (*bytes.Buffer, error) {
	logs, err := c.testContainer.Logs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("getting Lambda logs: %w", err)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(logs); err != nil {
		return nil, fmt.Errorf("reading Lambda logs: %w", err)
	}
	return buf, nil
}

func (c *LambdaDockerContainer) InvocationUrl() string {
	return fmt.Sprintf("http://localhost:%d/2015-03-31/functions/myfunction/invocations", c.MappedPort())
}
