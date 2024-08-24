package kubelet

import (
	"log"
)

type Kubelet interface {
	Run()
	Setup()
	Stop()
}

type KubeletApp struct{}

func NewKubelet() Kubelet {
	return &KubeletApp{}
}

func (app *KubeletApp) Setup() {
	log.Println("kubelet setup")
}

func (app *KubeletApp) Run() {
	log.Println("kubelet running")
}

func (app *KubeletApp) Stop() {
}
