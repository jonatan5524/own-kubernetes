package kubeapi

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

type KubeAPI interface {
	Run()
	Setup()
	Stop()
}

type KubeAPIApp struct {
	server        *http.Server
	Host          string
	restEndpoints []rest.Rest
	Port          int
}

const (
	defaultPort    = 8080
	defaultHost    = "0.0.0.0"
	defaultTimeout = 3 * time.Second
)

func NewKubeAPI(restEndpoints []rest.Rest) KubeAPI {
	app := &KubeAPIApp{}

	app.restEndpoints = restEndpoints
	app.Port = defaultPort
	app.Host = defaultHost

	app.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", app.Host, app.Port),
		ReadHeaderTimeout: defaultTimeout,
	}

	return app
}

func (app *KubeAPIApp) Setup() {
	for _, restEndpoint := range app.restEndpoints {
		restEndpoint.Register()
	}
}

func (app *KubeAPIApp) Run() {
	log.Printf("Kube api listening on %s:%d", app.Host, app.Port)
	err := app.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (app *KubeAPIApp) Stop() {
}
