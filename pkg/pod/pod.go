package pod

import (
	"context"
	"fmt"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/uuid"
)

type Pod struct {
	Id        string
	client    *containerd.Client
	ctx       *context.Context
	container *containerd.Container
}

type RunningPod struct {
	Pod         *Pod
	task        *containerd.Task
	exitStatusC <-chan containerd.ExitStatus
}

func NewPod(registryImage string, name string) (*Pod, error) {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), "own-kubernetes")

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

func (pod *Pod) Run() (*RunningPod, error) {
	task, err := (*pod.container).NewTask(*pod.ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, err
	}

	exitStatusC, err := task.Wait(*pod.ctx)
	if err != nil {
		fmt.Println(err)
	}

	if err := task.Start(*pod.ctx); err != nil {
		return nil, err
	}

	return &RunningPod{
		Pod:         pod,
		task:        &task,
		exitStatusC: exitStatusC,
	}, nil
}

func (pod *RunningPod) Kill() (uint32, error) {
	if err := (*pod.task).Kill(*pod.Pod.ctx, syscall.SIGTERM); err != nil {
		return 0, err
	}

	status := <-pod.exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return 0, err
	}

	(*pod.task).Delete(*pod.Pod.ctx)

	return code, nil
}

func (pod *Pod) Delete() {
	(*pod.container).Delete(*pod.ctx, containerd.WithSnapshotCleanup)
	pod.client.Close()
}

func ListRunningPods() ([]string, error) {
	runningPods := []string{}

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return runningPods, err
	}

	ctx := namespaces.WithNamespace(context.Background(), "own-kubernetes")

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
