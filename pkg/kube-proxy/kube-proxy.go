package kubeproxy

import (
	"fmt"
	"log"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/iptables"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service"
)

func Setup() error {
	log.Println("KubeProxy setup")

	if err := iptables.InitKubeServicesChain(); err != nil {
		return err
	}

	return nil
}

func Run(kubeAPIEndpoint string) error {
	log.Println("kube-proxy running")

	if err := service.ListenForServiceCreation(kubeAPIEndpoint); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func Stop() error {
	return nil
}
