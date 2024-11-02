package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/etcd"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
	"github.com/tidwall/gjson"
)

const (
	serviceEtcdKey = "/services/specs"
)

var etcdServiceAppService etcd.EtcdService

type Service struct {
	Metadata ServiceMetadata `json:"metadata" yaml:"metadata"`

	Kind string `json:"kind" yaml:"kind"`

	Spec struct {
		Type      string `json:"type" yaml:"type"`
		ClusterIP string `json:"clusterIP" yaml:"clusterIP"`
		Ports     []struct {
			Name       string `json:"name" yaml:"name"`
			Protocol   string `json:"protocol" yaml:"protocol"`
			Port       int    `json:"port" yaml:"port"`
			TargetPort int    `json:"targetPort" yaml:"targetPort"`
		} `json:"ports" yaml:"ports"`
	} `json:"spec" yaml:"spec"`
}

type ServiceMetadata struct {
	Annotations       map[string]string `json:"annotations" yaml:"annotations"`
	Name              string            `json:"name" yaml:"name"`
	Namespace         string            `json:"namespace" yaml:"namespace"`
	CreationTimestamp string            `json:"creationTimestamp" yaml:"creationTimestamp"`
	UID               string            `json:"uid" yaml:"uid"`
}

func (service *Service) Register(etcdServersEndpoints string) {
	log.Println("rest api service register")

	etcdServiceAppService = etcd.NewEtcdService(etcdServersEndpoints)

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/services").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(service.getAll).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))
	restful.Add(ws)
}

func (service *Service) getAll(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery == "true" {
		service.watcher(req, resp)

		return
	}

	resArr, err := etcdServiceAppService.GetAllFromResource(serviceEtcdKey)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	servicesRes := make([]Service, len(resArr))
	for index, res := range resArr {
		var service Service
		if err = json.Unmarshal(res, &service); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			return
		}

		servicesRes[index] = service
	}

	err = resp.WriteEntity(servicesRes)
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}

func (service *Service) watcher(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")
	fieldSelector := req.QueryParameter("fieldSelector")

	if watchQuery != "true" {
		return
	}

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")

	watchChan, closeChanFunc, err := etcdServiceAppService.GetWatchChannel(serviceEtcdKey)
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}
	defer closeChanFunc()
	defer resp.CloseNotify()

	log.Println("Client service watcher started")

	for {
		select {
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			for _, event := range watchResp.Events {
				log.Printf("watch: %s executed on %s with value %s\n", event.Type, string(event.Kv.Key), string(event.Kv.Value))

				if fieldSelector == "" {
					fmt.Fprintf(resp, "%s\n", string(event.Kv.Value))
				} else {
					splitedFieldSelector := strings.Split(fieldSelector, "=")
					resGJSON := gjson.Get(string(event.Kv.Value), splitedFieldSelector[0])

					if resGJSON.Exists() && resGJSON.Value() == splitedFieldSelector[1] {
						fmt.Fprintf(resp, "%s\n", string(event.Kv.Value))
					}
				}

				resp.Flush()
			}

		case <-req.Request.Context().Done():
			log.Println("Connection closed")
			return
		}
	}
}
