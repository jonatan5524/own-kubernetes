package containerd

import (
	"context"
	"fmt"
	"log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

const (
	containerdSocketPath = "/run/containerd/containerd.sock"
	defaultNamespace     = "own-kube"
)

func containerdConnection() (*containerd.Client, context.Context, error) {
	client, err := containerd.New(containerdSocketPath)
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	return client, ctx, nil
}

func CreateContainer(container *kubeapi_rest.Container, logLocation string) (string, error) {
	log.Printf("creating container %s with containerd", container.Name)

	client, ctx, err := containerdConnection()
	if err != nil {
		return "", err
	}
	defer client.Close()

	log.Printf("pulling image %s ", container.Image)
	imageRef, err := client.Pull(ctx, container.Image, containerd.WithPullUnpack)
	if err != nil {
		return "", err
	}

	var containerRef containerd.Container
	if len(container.Command) == 0 {
		if containerRef, err = client.NewContainer(
			ctx,
			uuid.New().String(),
			containerd.WithNewSnapshot(container.Name+"-snapshot", imageRef),
			containerd.WithNewSpec(
				oci.WithImageConfig(imageRef),
				oci.WithEnv(convertEnvToStringSlice(container)),
				oci.WithHostNamespace(specs.NetworkNamespace), // for now on host network
			),
		); err != nil {
			return "", err
		}
	} else {
		if containerRef, err = client.NewContainer(
			ctx,
			uuid.New().String(),
			containerd.WithNewSnapshot(container.Name+"-snapshot", imageRef),
			containerd.WithNewSpec(
				oci.WithImageConfig(imageRef),
				oci.WithEnv(convertEnvToStringSlice(container)),
				oci.WithProcessArgs(append(container.Command, container.Args...)...),
				oci.WithHostNamespace(specs.NetworkNamespace), // for now on host network
			),
		); err != nil {
			return "", err
		}
	}

	err = startContainer(ctx, containerRef, logLocation)
	if err != nil {
		return "", err
	}

	return containerRef.ID(), nil
}

func convertEnvToStringSlice(container *kubeapi_rest.Container) []string {
	env := make([]string, len(container.Env))

	for index, envStruct := range container.Env {
		env[index] = fmt.Sprintf("%s=%s", envStruct.Name, envStruct.Value)
	}

	return env
}

func startContainer(ctx context.Context, container containerd.Container, logLocation string) error {
	log.Printf("starting container %s", container.ID())

	task, err := container.NewTask(ctx, cio.LogFile(logLocation))
	if err != nil {
		return err
	}

	return task.Start(ctx)
}
