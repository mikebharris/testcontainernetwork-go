package testcontainernetwork

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

type DynamoDbDockerContainerConfig struct {
	Hostname string
	Port     int
}

type DynamoDbDockerContainer struct {
	DockerContainer
	Config DynamoDbDockerContainerConfig
}

func (c *DynamoDbDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	c.internalServicePort = c.Config.Port
	req := testcontainers.ContainerRequest{
		Image:        "amazon/dynamodb-local",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Networks:     []string{dockerNetwork.Name},
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
		},
		Entrypoint: []string{"java", "-jar", "DynamoDBLocal.jar", "-inMemory", "-sharedDb"},
	}
	var err error
	c.testContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	if err := c.testContainer.Start(ctx); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	return nil
}
