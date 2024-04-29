package testcontainernetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

type WiremockDockerContainer struct {
	GenericDockerContainer
	Config WiremockDockerContainerConfig
}

func (c *WiremockDockerContainer) Stop(ctx context.Context) error {
	return c.dockerContainer.Terminate(ctx)
}

func (c *WiremockDockerContainer) StartUsing(ctx context.Context, dockerNetwork *testcontainers.DockerNetwork) error {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if c.Config.Hostname == "" {
		c.Config.Hostname = "wiremock"
	}
	c.internalServicePort = 8080

	req := testcontainers.ContainerRequest{
		Image:        "wiremock/wiremock",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", c.internalServicePort)},
		Name:         c.Config.Hostname,
		Hostname:     c.Config.Hostname,
		Networks:     []string{dockerNetwork.Name},
		HostConfigModifier: func(config *container.HostConfig) {
			config.NetworkMode = container.NetworkMode(dockerNetwork.Name)
			config.Mounts = []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   path.Join(wd, c.Config.JsonMappings),
					Target:   "/home/wiremock/mappings/",
					ReadOnly: true,
				},
			}
		},
	}
	c.dockerContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		panic(err)
	}

	if err := c.dockerContainer.Start(ctx); err != nil {
		panic(err)
	}
	return nil
}

func (c *WiremockDockerContainer) GetAdminStatus() WiremockAdminStatus {
	wireMockAdminUri := fmt.Sprintf("http://localhost:%d/__admin/requests", c.MappedPort())
	req, _ := http.NewRequest(http.MethodGet, wireMockAdminUri, nil)

	var client = http.Client{
		Timeout: time.Second * 30,
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Fatalf("making http request: %v", getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatalf("reading body: %v", readErr)
	}

	var wiremockAdminStatus WiremockAdminStatus
	if err := json.Unmarshal(body, &wiremockAdminStatus); err != nil {
		log.Fatalf("Unmarshalling body: %v", err)
	}

	return wiremockAdminStatus
}

type WiremockAdminStatus struct {
	Requests               []WiremockAdminRequest `json:"requests"`
	Meta                   WiremockAdminMeta      `json:"meta"`
	RequestJournalDisabled bool                   `json:"requestJournalDisabled"`
}

type WiremockAdminMeta struct {
	total int
}

type WiremockAdminRequest struct {
	Id      string `json:"id"`
	Request struct {
		Url         string `json:"url"`
		AbsoluteUrl string `json:"absoluteUrl"`
		Method      string `json:"method"`
		ClientIp    string `json:"clientIp"`
		Headers     struct {
			Connection string `json:"Connection"`
			UserAgent  string `json:"User-Agent"`
			Host       string `json:"Host"`
		} `json:"headers"`
		Cookies struct {
		} `json:"cookies"`
		BrowserProxyRequest bool      `json:"browserProxyRequest"`
		LoggedDate          int64     `json:"loggedDate"`
		BodyAsBase64        string    `json:"bodyAsBase64"`
		Body                string    `json:"body"`
		LoggedDateString    time.Time `json:"loggedDateString"`
	} `json:"request"`
	ResponseDefinition struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	} `json:"responseDefinition"`
	WasMatched bool `json:"wasMatched"`
}

type WiremockDockerContainerConfig struct {
	JsonMappings string
	Hostname     string
}
