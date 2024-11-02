package kubeproxy

import (
	"log"
)

type KubeProxy interface {
	Run() error
	Setup() error
	Stop() error
}

type KubeProxyApp struct{}

func NewKubeProxy() KubeProxy {
	app := &KubeProxyApp{}

	return app
}

func (app *KubeProxyApp) Setup() error {
	log.Println("KubeProxy setup")

	return nil
}

func (app *KubeProxyApp) Run() error {
	return nil
}

func (app *KubeProxyApp) Stop() error {
	return nil
}
