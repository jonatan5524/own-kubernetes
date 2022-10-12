package node

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jonatan5524/own-kubernetes/pkg/agent/api"
)

func initDockerConnection() (*client.Client, context.Context, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, err
	}

	return cli, ctx, nil
}

func NewNodeAndRun() (*Node, error) {
	cli, ctx, err := initDockerConnection()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	node, err := NewNode(cli, ctx)
	if err != nil {
		return nil, err
	}

	if err := cli.ContainerStart(ctx, node.Id, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	log.Printf("node created: %s\n", node.Id)
	log.Printf("starting node\n")

	container, err := cli.ContainerInspect(ctx, node.Id)
	if err != nil {
		return nil, err
	}

	// get linux generated port
	node.Port = container.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%s/tcp", api.PORT))][0].HostPort

	log.Printf("node assign port: %s\n", node.Port)

	return node, nil
}

func ListRunningNodes() ([]string, error) {
	cli, ctx, err := initDockerConnection()
	if err != nil {
		return []string{}, err
	}
	defer cli.Close()

	runningNodes := []string{}

	filter := filters.NewArgs(filters.KeyValuePair{Key: "ancestor", Value: NODE_IMAGE})
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: filter})
	if err != nil {
		return runningNodes, err
	}

	for _, container := range containers {
		runningNodes = append(runningNodes, container.Names[0])
	}

	return runningNodes, nil
}

func KillNode(name string) (string, error) {
	cli, ctx, err := initDockerConnection()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	if err := cli.ContainerStop(ctx, name, nil); err != nil {
		return "", err
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, name, removeOptions); err != nil {
		return "", err
	}

	return name, nil
}
