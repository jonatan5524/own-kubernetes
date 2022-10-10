package pod

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/uuid"
)

func initContainerdConnection() (*containerd.Client, context.Context, error) {
	client, err := containerd.New(SOCKET_PATH)
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), NAMESPACE)

	return client, ctx, nil
}

func NewPodAndRun(imageRegistry string, name string) (string, error) {
	pod, err := NewPod(imageRegistry, name)
	if err != nil {
		return "", err
	}

	log.Printf("pod created: %s\n", pod.Id)
	log.Printf("starting pod\n")

	_, err = pod.Run()
	if err != nil {
		return "", err
	}

	return pod.Id, nil
}

func NewPod(imageRegistry string, name string) (*Pod, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return nil, err
	}

	image, err := client.Pull(ctx, imageRegistry, containerd.WithPullUnpack)
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

func LogPod(id string) (string, error) {
	logs, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", LOGS_PATH, id))
	log.Println(fmt.Sprintf("%s/%s", LOGS_PATH, id))

	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func KillPod(id string) (string, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return "", err
	}

	container, err := client.LoadContainer(ctx, id)
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
			Id:        id,
		},
		exitStatusC: exitStatusC,
	}

	_, err = runningPod.Kill()
	if err != nil {
		return "", err
	}

	runningPod.Pod.Delete()

	return id, nil
}
