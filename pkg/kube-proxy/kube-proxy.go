package kubeproxy

import (
	"fmt"
	"log"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/iptables"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service/endpoint"
)

const (
	serviceClusterIPCIDR = "10.96.0.0/16"
)

func Setup() error {
	log.Println("KubeProxy setup")

	if err := iptables.InitKubeServicesChain(); err != nil {
		return err
	}

	if err := iptables.InitNodePortChain(); err != nil {
		return err
	}

	return nil
}

func Run(kubeAPIEndpoint string, hostname string, podCIDR string) error {
	log.Println("kube-proxy running")

	go endpoint.ListenForEndpoint(kubeAPIEndpoint, hostname)
	go service.ListenForPodRunning(kubeAPIEndpoint, hostname)

	if err := service.ListenForService(kubeAPIEndpoint, serviceClusterIPCIDR, podCIDR); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func Stop() error {
	return nil
}
