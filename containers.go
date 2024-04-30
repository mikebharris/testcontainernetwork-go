package testcontainernetwork

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"log"
	"time"
)

type StartableDockerContainer interface {
	MappedPort() int
	StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error
	Stop(ctx context.Context) error
}

type GenericDockerContainer struct {
	dockerContainer     testcontainers.Container
	internalServicePort int
}

func (c *GenericDockerContainer) MappedPort() int {
	mappedPort, err := c.dockerContainer.MappedPort(context.Background(), nat.Port(fmt.Sprintf("%d/tcp", c.internalServicePort)))
	if err != nil {
		log.Fatalf("getting mapped port: %v", err)
	}
	return mappedPort.Int()
}

func (c *GenericDockerContainer) Stop(ctx context.Context) error {
	return c.dockerContainer.Terminate(ctx)
}

type NetworkOfDockerContainers struct {
	dockerNetwork    *testcontainers.DockerNetwork
	dockerContainers []StartableDockerContainer
}

func (n NetworkOfDockerContainers) WithDockerContainer(dockerContainer StartableDockerContainer) NetworkOfDockerContainers {
	n.dockerContainers = append(n.dockerContainers, dockerContainer)
	return n
}

func (n NetworkOfDockerContainers) StartWithDelay(delay time.Duration) NetworkOfDockerContainers {
	ctx := context.Background()
	var err error
	if n.dockerNetwork, err = network.New(ctx); err != nil {
		log.Fatalf("creating network: %s", err)
	}
	for _, dockerContainer := range n.dockerContainers {
		if err := dockerContainer.StartUsing(ctx, n.dockerNetwork); err != nil {
			log.Fatalf("starting docker container: %s", err)
		}
	}
	if delay > 0 {
		fmt.Printf("Sleeping for %s while containers start\n", delay)
		time.Sleep(delay)
	}
	return n
}

func (n NetworkOfDockerContainers) Stop() error {
	ctx := context.Background()
	for _, dockerContainer := range n.dockerContainers {
		if err := dockerContainer.Stop(ctx); err != nil {
			return fmt.Errorf("stopping docker container: %v", err)
		}
	}
	if err := n.dockerNetwork.Remove(ctx); err != nil {
		return fmt.Errorf("removing network: %v", err)
	}
	return nil
}
