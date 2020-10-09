package client

import (
	"context"
	"os"

	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/bindings"
	"github.com/containers/podman/v2/pkg/bindings/containers"
	"github.com/containers/podman/v2/pkg/bindings/images"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/specgen"
)

type Client struct {
	context context.Context
}

func (c *Client) Connect() error {
	// Get Podman socket location
	sockDir := os.Getenv("XDG_RUNTIME_DIR")
	socket := "unix:" + sockDir + "/podman/podman.sock"

	// Connect to Podman socket
	connText, err := bindings.NewConnection(context.Background(), socket)
	if err != nil {
		return err
	}
	c.context = connText
	return nil
}

func (c *Client) PullImage(rawImage string) error {
	_, err := images.Pull(c.context, rawImage, entities.ImagePullOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateContainer(s *specgen.SpecGenerator) (string, error) {
	s.Terminal = true
	r, err := containers.CreateWithSpec(c.context, s)
	if err != nil {
		return "", err
	}
	return r.ID, c.StartContainer(r.ID)
}

func (c *Client) StartContainer(containerId string) error {
	err := containers.Start(c.context, containerId, nil)
	if err != nil {
		return err
	}
	return c.WaitContainer(containerId)
}

func (c *Client) WaitContainer(containerId string) error {
	running := define.ContainerStateRunning
	_, err := containers.Wait(c.context, containerId, &running)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StopContainer(containerId string) error {
	return containers.Stop(c.context, containerId, nil)
}

func (c *Client) RemoveContainer(containerId string) error {
	return containers.Remove(c.context, containerId, newTrue(), newTrue())
}
