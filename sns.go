package testcontainernetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

type SnsDockerContainerConfig struct {
	Hostname   string
	ConfigFile string
	Port       int
}

type SnsDockerContainer struct {
	DockerContainer
	Config SnsDockerContainerConfig
}

func (c *SnsDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	c.internalServicePort = c.Config.Port
	req := testcontainers.ContainerRequest{
		Image:        "warrenseine/sns",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Networks:     []string{dockerNetwork.Name},
		Mounts:       testcontainers.Mounts(),
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
		},
	}
	var err error
	if c.dockerContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	}); err != nil {
		return fmt.Errorf("creating container: %w", err)
	}

	if err = c.dockerContainer.CopyFileToContainer(ctx, c.Config.ConfigFile, "/etc/sns/db.json", 365); err != nil {
		return fmt.Errorf("copying config file to docker container: %v", err)
	}

	if err := c.dockerContainer.Start(ctx); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	return nil
}

func (c *SnsDockerContainer) GetMessage() (string, error) {
	snsLog, err := c.dockerContainer.CopyFileFromContainer(context.Background(), "/tmp/sns.log")
	if err != nil {
		return "", fmt.Errorf("copying log file from docker container: %w", err)
	}

	type SNSMessage struct {
		MessageId string `json:"MessageId"`
		Message   string `json:"Message"`
		Type      string `json:"Type"`
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(snsLog)
	if err != nil {
		return "", fmt.Errorf("reading log file: %w", err)
	}
	result := buf.String()

	var snsMessage SNSMessage
	err = json.Unmarshal([]byte(result), &snsMessage)
	if err != nil {
		return "", fmt.Errorf("unmarshalling json: %w", err)
	}

	return snsMessage.Message, nil
}
