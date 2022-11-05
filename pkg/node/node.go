package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jonatan5524/own-kubernetes/pkg"
	"github.com/jonatan5524/own-kubernetes/pkg/agent/api"
)

type Node struct {
	Id   string
	Port string
}

func NewNode(cli *client.Client, ctx context.Context) (*Node, error) {
	err := createNetwork(cli, ctx)
	if err != nil {
		return nil, err
	}

	exists, err := isNodeImageExists(cli, ctx)
	if err != nil {
		return nil, err
	} else if exists == false {
		return nil, fmt.Errorf("node image: %s not exists locally, need to build the image", NODE_IMAGE)
	}

	id := pkg.GenerateNewID(NODE_NAME)
	config := &container.Config{
		Image: NODE_IMAGE,
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%s/tcp", api.PORT)): []nat.PortBinding{
				{
					HostIP:   NODE_PORT_HOST_IP,
					HostPort: "0",
				},
			},
		},
		NetworkMode: NODE_DOCKER_NETWORK_NAME,
		Resources: container.Resources{
			Memory:    MEMORY_LIMIT,
			CPUShares: CPU_LIMIT,
		},
		Privileged: true,
	}

	_, err = cli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, id)
	if err != nil {
		return nil, err
	}

	return &Node{Id: id}, nil
}

func createNetwork(cli *client.Client, ctx context.Context) error {
	_, err := cli.NetworkInspect(ctx, NODE_DOCKER_NETWORK_NAME, types.NetworkInspectOptions{})
	if err == nil {
		return nil
	}

	newNetwork := types.NetworkCreate{IPAM: &network.IPAM{
		Driver: "default",
		Config: []network.IPAMConfig{{
			Subnet: NODE_CIDR,
		}},
	}}

	_, err = cli.NetworkCreate(context.Background(), NODE_DOCKER_NETWORK_NAME, newNetwork)
	if err != nil {
		return err
	}

	return nil
}

func isNodeImageExists(cli *client.Client, ctx context.Context) (bool, error) {
	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for _, image := range images {
		if strings.Contains(image.RepoTags[0], NODE_IMAGE) {
			return true, nil
		}
	}

	return false, nil
}
