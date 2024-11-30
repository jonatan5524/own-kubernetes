package kubelet

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	kubeproxy "github.com/jonatan5524/own-kubernetes/pkg/kube-proxy"
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet/pod"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

type Kubelet interface {
	Run() error
	Setup() error
	Stop() error
}

type KubeletApp struct {
	logFile            *os.File
	systemManifestPath string
	loggingLocation    string
	kubeAPIEndpoint    string
	hostname           string
}

const (
	defaultSystemManifestPath = "/home/user/kubernetes/manifests"
	defaultLoggingLocation    = "/home/user/kubernetes/log/kubelet.log"
	podCIDR                   = "10.244.0.0/16"
	podBridgeName             = "br0"
)

func NewKubelet(kubeAPIEndpoint string) Kubelet {
	return &KubeletApp{
		systemManifestPath: defaultSystemManifestPath,
		loggingLocation:    defaultLoggingLocation,
		kubeAPIEndpoint:    kubeAPIEndpoint,
	}
}

func (app *KubeletApp) Setup() error {
	log.Println("kubelet setup")

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("hostname not found")
	}
	app.hostname = hostname
	log.Printf("running on host %s", hostname)

	if err = utils.CreateDirectory(filepath.Dir(app.loggingLocation), 0o644); err != nil {
		return fmt.Errorf("unable to create log file location %v", err)
	}
	file, err := os.OpenFile(app.loggingLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	log.SetOutput(file)
	app.logFile = file

	if err := initCIDRPodNetwork(podCIDR, podBridgeName); err != nil {
		return err
	}

	if err := kubeproxy.Setup(); err != nil {
		return err
	}

	return nil
}

func (app *KubeletApp) Run() error {
	log.Println("kubelet running")

	pods, err := readAndStartSystemManifests(app.systemManifestPath, app.hostname)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	// TODO: switch later for wait for kube-api to be ready
	time.Sleep(5 * time.Second)

	if len(pods) > 0 {
		for _, podRes := range pods {
			err := pod.UpdatePod(app.kubeAPIEndpoint, *podRes)
			if err != nil {
				return fmt.Errorf("error sending system pods to api: %v", err)
			}
		}
	}

	go kubeproxy.Run(app.kubeAPIEndpoint, app.hostname, podCIDR)
	defer kubeproxy.Stop()

	if err := pod.ListenForPodCreation(app.kubeAPIEndpoint, app.hostname, podCIDR, podBridgeName); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func (app *KubeletApp) Stop() error {
	return app.logFile.Close()
}
