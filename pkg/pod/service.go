package pod

import (
	"context"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/uuid"
)

const (
	SOCKET_PATH = "/run/containerd/containerd.sock"
	NAMESPACE   = "own-kubernetes"
)

func initContainerdConnection() (*containerd.Client, context.Context, error) {
	client, err := containerd.New(SOCKET_PATH)
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), NAMESPACE)

	return client, ctx, nil
}

func NewPod(registryImage string, name string) (*Pod, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return nil, err
	}

	image, err := client.Pull(ctx, registryImage, containerd.WithPullUnpack)
	if err != nil {
		return nil, err
	}

	id := generateNewID(name)

	container, err := client.NewContainer(
		ctx,
		id,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(id+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return nil, err
	}

	return &Pod{
		Id:        id,
		container: &container,
		ctx:       &ctx,
		client:    client,
	}, nil
}

func generateNewID(name string) string {
	id := uuid.New()

	return fmt.Sprintf("%s-%s", name, id)
}

func ListRunningPods() ([]string, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return nil, err
	}

	runningPods := []string{}

	containers, err := client.Containers(ctx)
	if err != nil {
		return runningPods, err
	}

	for _, container := range containers {
		_, err = container.Task(ctx, cio.Load)

		if err == nil {
			runningPods = append(runningPods, container.ID())
		}
	}

	return runningPods, nil
}

func KillPod(name string) (string, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return "", err
	}

	container, err := client.LoadContainer(ctx, name)
	if err != nil {
		return "", err
	}

	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		return "", err
	}

	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return "", err
	}

	runningPod := RunningPod{
		task: &task,
		Pod: &Pod{
			client:    client,
			ctx:       &ctx,
			container: &container,
			Id:        name,
		},
		exitStatusC: exitStatusC,
	}

	_, err = runningPod.Kill()
	if err != nil {
		return "", err
	}

	runningPod.Pod.Delete()

	return name, nil
}
