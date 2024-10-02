package kubelet

import (
	"log"
	"os"
)

type Kubelet interface {
	Run()
	Setup()
	Stop()
}

type KubeletApp struct {
	systemManifestPath string
	loggingLocation    string
	logFile            *os.File
}

const (
	defaultSystemManifestPath = "/home/user/kubernetes/manifests"
	defaultLoggingLocation    = "/home/user/kubernetes/log/kubelet.log"
)

func NewKubelet() Kubelet {
	return &KubeletApp{
		systemManifestPath: defaultSystemManifestPath,
		loggingLocation:    defaultLoggingLocation,
	}
}

func (app *KubeletApp) Setup() {
	log.Println("kubelet setup")

	file, err := os.OpenFile(app.loggingLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)
	app.logFile = file
}

func (app *KubeletApp) Run() {
	log.Println("kubelet running")

	if err := ReadAndStartSystemManifests(app.systemManifestPath); err != nil {
		log.Fatalf("%v", err)
	}
}

func (app *KubeletApp) Stop() {
	app.logFile.Close()
}
