package pod

import (
	"context"
	"fmt"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
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

func (pod *Pod) Run() (*RunningPod, error) {
	task, err := (*pod.container).NewTask(*pod.ctx, cio.LogFile(fmt.Sprintf("%s/%s", LOGS_PATH, pod.Id)))
	if err != nil {
		return nil, err
	}

	exitStatusC, err := task.Wait(*pod.ctx)
	if err != nil {
		return nil, err
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
