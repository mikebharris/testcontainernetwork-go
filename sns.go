package testcontainernetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"log"
)

type SnsDockerContainerConfig struct {
	Hostname       string
	ConfigFilePath string
}

type SnsDockerContainer struct {
	DockerContainer
	Config SnsDockerContainerConfig
}

func (c *SnsDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	if c.Config.Hostname == "" {
		c.Config.Hostname = "sns"
	}
	c.internalServicePort = 9911
	req := testcontainers.ContainerRequest{
		Image:        "warrenseine/sns",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Networks:     []string{dockerNetwork.Name},
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

	if err = c.dockerContainer.CopyFileToContainer(ctx, c.Config.ConfigFilePath, "/etc/sns/sns.json", 365); err != nil {
		log.Fatalf("copying config file %s to docker container: %v", c.Config.ConfigFilePath, err)
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
	if _, err := buf.ReadFrom(snsLog); err != nil {
		return "", fmt.Errorf("reading log file: %w", err)
	}
	var snsMessage SNSMessage
	if err := json.Unmarshal([]byte(buf.String()), &snsMessage); err != nil {
		return "", fmt.Errorf("unmarshalling log file: %w", err)
	}
	return snsMessage.Message, nil
}
