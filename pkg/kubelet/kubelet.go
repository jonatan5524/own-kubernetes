package kubelet

import (
	"log"
	"sync"
)

type Kubelet interface {
	Run()
	Setup()
	Stop()
}

type KubeletHandler interface {
	Register() error
	StartWatch(*sync.WaitGroup)
}

type KubeletApp struct {
	watchEndpoints []KubeletHandler
}

func NewKubelet(watchEndpoints []KubeletHandler) Kubelet {
	return &KubeletApp{
		watchEndpoints: watchEndpoints,
	}
}

func (app *KubeletApp) Setup() {
	log.Println("kubelet setup")

	for _, watchEndpoint := range app.watchEndpoints {
		if err := watchEndpoint.Register(); err != nil {
			log.Fatalf("error while setup: %v", err)
		}
	}
}

func (app *KubeletApp) Run() {
	log.Println("kubelet running")
	var wg sync.WaitGroup

	for _, watchEndpoint := range app.watchEndpoints {
		wg.Add(1)
		go watchEndpoint.StartWatch(&wg)
	}

	wg.Wait()
}

func (app *KubeletApp) Stop() {
}
