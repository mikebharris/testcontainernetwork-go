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

type DockerContainer struct {
	dockerContainer     testcontainers.Container
	internalServicePort int
}

func (c *DockerContainer) MappedPort() int {
	mappedPort, err := c.dockerContainer.MappedPort(context.Background(), nat.Port(fmt.Sprintf("%d/tcp", c.internalServicePort)))
	if err != nil {
		log.Fatalf("getting mapped port: %v", err)
	}
	return mappedPort.Int()
}

func (c *DockerContainer) Stop(ctx context.Context) error {
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

// StartWithDelay has intentional mixed use of pointer and value receivers because this method
// has side effects and thus this fits better with a functional programming paradigm
func (n *NetworkOfDockerContainers) StartWithDelay(delay time.Duration) error {
	ctx := context.Background()
	var err error
	if n.dockerNetwork, err = network.New(ctx); err != nil {
		return fmt.Errorf("creating network: %s", err)
	}
	for _, dockerContainer := range n.dockerContainers {
		if err := dockerContainer.StartUsing(ctx, n.dockerNetwork); err != nil {
			return fmt.Errorf("starting docker container: %s", err)
		}
	}
	if delay > 0 {
		fmt.Printf("Sleeping for %s while containers start\n", delay)
		time.Sleep(delay)
	}
	return nil
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
