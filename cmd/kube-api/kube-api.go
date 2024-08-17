package main

import (
	kubeapi "github.com/jonatan5524/own-kubernetes/pkg/kube-api"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

func main() {
	app := kubeapi.NewKubeAPI([]rest.Rest{
		&rest.Pod{},
	})

	app.Setup()
	app.Run()
}
