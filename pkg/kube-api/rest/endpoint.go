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
	endpointEtcdKey = "/services/endpoints"
)

var etcdServiceAppEndpoint etcd.EtcdService

type Endpoint struct {
	Metadata ResourceMetadata `json:"metadata" yaml:"metadata"`

	Kind string `json:"kind" yaml:"kind"`

	Subsets []EndpointSubset `json:"subsets" yaml:"subsets"`
}

type EndpointSubset struct {
	Addresses []EndpointAddress `json:"addresses" yaml:"addresses"`
	Ports     []ServicePorts    `json:"ports" yaml:"ports"`
}

type EndpointAddress struct {
	IP        string    `json:"ip" yaml:"ip"`
	NodeName  string    `json:"nodeName" yaml:"nodeName"`
	TargetRef TargetRef `json:"targetRef" yaml:"targetRef"`
}

func (endpoint *Endpoint) Register(etcdServersEndpoints string) {
	log.Println("rest api endpoint register")

	etcdServiceAppEndpoint = etcd.NewEtcdService(etcdServersEndpoints)

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/endpoints").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(endpoint.getAll).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))
	restful.Add(ws)
}

func (endpoint *Endpoint) getAll(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery == "true" {
		endpoint.watcher(req, resp)

		return
	}

	resArr, err := etcdServiceAppEndpoint.GetAllFromResource(endpointEtcdKey)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	endpointsRes := make([]Endpoint, len(resArr))
	for index, res := range resArr {
		var endpoint Endpoint
		if err = json.Unmarshal(res, &endpoint); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			return
		}

		endpointsRes[index] = endpoint
	}

	err = resp.WriteEntity(endpointsRes)
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}

func (endpoint *Endpoint) watcher(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")
	fieldSelector := req.QueryParameter("fieldSelector")

	if watchQuery != "true" {
		return
	}

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")

	watchChan, closeChanFunc, err := etcdServiceAppEndpoint.GetWatchChannel(endpointEtcdKey)
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}
	defer closeChanFunc()
	defer resp.CloseNotify()

	log.Println("Client endpoint watcher started")

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
