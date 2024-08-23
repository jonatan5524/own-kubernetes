package main

import (
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet"
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet/handlers/pod"
)

func main() {
	app := kubelet.NewKubelet([]kubelet.KubeletHandler{
		&pod.Pod{},
	})

	app.Setup()
	app.Run()
}
