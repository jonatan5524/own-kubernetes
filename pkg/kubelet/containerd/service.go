package containerd

import (
	"context"
	"fmt"
	"log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

const (
	containerdSocketPath      = "/run/containerd/containerd.sock"
	defaultPodLoggingLocation = "/home/user/kubernetes/log/pod/"
)

func containerdConnection(namespace string) (*containerd.Client, context.Context, error) {
	client, err := containerd.New(containerdSocketPath)
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), namespace)

	return client, ctx, nil
}

func CreateContainer(container *kubeapi_rest.Container, namespace string) error {
	log.Printf("creating container %s in namespace %s with containerd", container.Name, namespace)

	client, ctx, err := containerdConnection(namespace)
	if err != nil {
		return err
	}
	defer client.Close()

	log.Printf("pulling image %s ", container.Image)
	imageRef, err := client.Pull(ctx, container.Image, containerd.WithPullUnpack)
	if err != nil {
		return err
	}

	var containerRef containerd.Container
	if len(container.Command) == 0 {
		if containerRef, err = client.NewContainer(
			ctx,
			container.Name,
			containerd.WithNewSnapshot(container.Name+"-snapshot", imageRef),
			containerd.WithNewSpec(
				oci.WithImageConfig(imageRef),
				oci.WithEnv(convertEnvToStringSlice(container)),
				oci.WithHostNamespace(specs.NetworkNamespace), // for now on host network
			),
		); err != nil {
			return err
		}
	} else {
		if containerRef, err = client.NewContainer(
			ctx,
			container.Name,
			containerd.WithNewSnapshot(container.Name+"-snapshot", imageRef),
			containerd.WithNewSpec(
				oci.WithImageConfig(imageRef),
				oci.WithEnv(convertEnvToStringSlice(container)),
				oci.WithProcessArgs(append(container.Command, container.Args...)...),
				oci.WithHostNamespace(specs.NetworkNamespace), // for now on host network
			),
		); err != nil {
			return err
		}
	}

	return startContainer(containerRef, ctx, container.Name)
}

func convertEnvToStringSlice(container *kubeapi_rest.Container) []string {
	env := make([]string, len(container.Env))

	for index, envStruct := range container.Env {
		env[index] = fmt.Sprintf("%s=%s", envStruct.Name, envStruct.Value)
	}

	return env
}

func startContainer(container containerd.Container, ctx context.Context, podUID string) error {
	log.Printf("starting container %s", podUID)

	task, err := container.NewTask(ctx, cio.LogFile(fmt.Sprintf("%s/%s", defaultPodLoggingLocation, podUID)))
	if err != nil {
		return err
	}

	return task.Start(ctx)
}
