package testcontainernetwork

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

type PostgresDockerContainerConfig struct {
	Hostname    string
	Port        int
	Environment map[string]string
}

type PostgresDockerContainer struct {
	DockerContainer
	Config PostgresDockerContainerConfig
}

func (c *PostgresDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	c.internalServicePort = c.Config.Port
	req := testcontainers.ContainerRequest{
		Image:        "postgres:13",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Networks:     []string{dockerNetwork.Name},
		Mounts:       testcontainers.Mounts(),
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
		},
		Env: c.Config.Environment,
	}

	var err error
	if c.dockerContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	}); err != nil {
		return fmt.Errorf("creating container: %w", err)
	}

	if err := c.dockerContainer.Start(ctx); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	return nil
}
