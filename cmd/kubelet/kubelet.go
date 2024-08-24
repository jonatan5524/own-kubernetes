package main

import (
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet"
)

func main() {
	app := kubelet.NewKubelet()

	app.Setup()
	app.Run()
}
