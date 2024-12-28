package kubeapi

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
)

type KubeAPI interface {
	Run() error
	Setup() error
	Stop() error
}

type Rest interface {
	Register(etcdServers string)
}

type KubeAPIApp struct {
	server        *http.Server
	Host          string
	restEndpoints []Rest
	Port          int
	EtcdServers   string
}

const (
	defaultPort    = 8080
	defaultHost    = "0.0.0.0"
	defaultTimeout = 3 * time.Second
)

func NewKubeAPI(etcdServers string, restEndpoints []Rest) KubeAPI {
	app := &KubeAPIApp{}

	app.restEndpoints = restEndpoints
	app.Port = defaultPort
	app.Host = defaultHost

	app.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", app.Host, app.Port),
		ReadHeaderTimeout: defaultTimeout,
	}

	app.EtcdServers = etcdServers

	return app
}

func setupHealth() {
	log.Println("setup health check endpoint")

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/health").To(func(_ *restful.Request, res *restful.Response) {
		err := res.WriteEntity("all good")
		if err != nil {
			res.WriteError(http.StatusInternalServerError, err)
		}
	}))

	restful.Add(ws)
}

func (app *KubeAPIApp) Setup() error {
	log.Println("KubeApi setup")

	setupHealth()

	for _, restEndpoint := range app.restEndpoints {
		restEndpoint.Register(app.EtcdServers)
	}

	return nil
}

func (app *KubeAPIApp) Run() error {
	log.Printf("Kube api listening on %s:%d", app.Host, app.Port)

	err := app.server.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (app *KubeAPIApp) Stop() error {
	return nil
}
