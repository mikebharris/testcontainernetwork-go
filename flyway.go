package testcontainernetwork

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/testcontainers/testcontainers-go"
)

type FlywayDockerContainerConfig struct {
	Hostname       string
	Port           int
	ConfigFilePath string
	SqlFilePath    string
}

type FlywayDockerContainer struct {
	DockerContainer
	Config FlywayDockerContainerConfig
}

func (c *FlywayDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	c.internalServicePort = c.Config.Port
	req := testcontainers.ContainerRequest{
		Image:    "flyway/flyway",
		Name:     c.Config.Hostname,
		Hostname: c.Config.Hostname,
		Networks: []string{dockerNetwork.Name},
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
			config.Mounts = []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   c.Config.SqlFilePath,
					Target:   "/flyway/sql",
					ReadOnly: true,
				},
				{
					Type:     mount.TypeBind,
					Source:   c.Config.ConfigFilePath,
					Target:   "/flyway/conf",
					ReadOnly: true,
				},
			}
		},
		Entrypoint: []string{"flyway", "migrate"},
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
