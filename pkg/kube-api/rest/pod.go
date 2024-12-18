package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/etcd"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
)

const (
	podEtcdKey                            = "/pods"
	defaultNamespace                      = "default"
	LastAppliedConfigurationAnnotationKey = "last-applied-configuration"
)

var etcdServiceAppPod etcd.EtcdService

type Pod struct {
	Metadata ResourceMetadata `json:"metadata" yaml:"metadata"`

	Kind string `json:"kind" yaml:"kind"`

	Status PodStatus `json:"status" yaml:"status"`

	Spec struct {
		NodeName    string      `json:"nodeName" yaml:"nodeName"`
		Containers  []Container `json:"containers" yaml:"containers"`
		HostNetwork bool        `json:"hostNetwork" yaml:"hostNetwork"`
	} `json:"spec" yaml:"spec"`
}

type PodStatus struct {
	PodIP             string            `json:"podIP" yaml:"podIP"`
	Phase             string            `json:"phase" yaml:"phase"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses" yaml:"containerStatuses"`
}

type ContainerStatus struct {
	ContainerID string `json:"containerID" yaml:"containerID"`
	Image       string `json:"image" yaml:"image"`
	Name        string `json:"name" yaml:"name"`
}

type Container struct {
	Name  string `json:"name" yaml:"name"`
	Image string `json:"image" yaml:"image"`

	Command []string `json:"command" yaml:"command"`
	Args    []string `json:"args" yaml:"args"`

	Ports []struct {
		ContainerPort int `json:"containerPort" yaml:"containerPort"`
	} `json:"ports" yaml:"ports"`

	Env []struct {
		Name  string `json:"name" yaml:"name"`
		Value string `json:"value" yaml:"value"`
	} `json:"env" yaml:"env"`

	SecurityContext struct {
		Privileged bool `json:"privileged" yaml:"privileged"`
	} `json:"securityContext" yaml:"securityContext"`
}

func (pod *Pod) Register(etcdServersEndpoints string) {
	log.Println("rest api pod register")

	etcdServiceAppPod = etcd.NewEtcdService(etcdServersEndpoints)

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/pods").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(pod.getAll).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))

	restful.Add(ws)
}

func (pod *Pod) initLastAppliedConfigurations() error {
	podWithoutStatus := *pod
	podWithoutStatus.Status = PodStatus{}
	podWithoutStatus.Metadata.Annotations = make(map[string]string)
	podWithoutStatus.Metadata.CreationTimestamp = ""
	podWithoutStatus.Metadata.UID = ""

	podBytes, err := json.Marshal(podWithoutStatus)
	if err != nil {
		return err
	}

	if pod.Metadata.Annotations == nil {
		pod.Metadata.Annotations = make(map[string]string)
	}

	pod.Metadata.Annotations[LastAppliedConfigurationAnnotationKey] = string(podBytes)

	return nil
}

func (pod *Pod) getAll(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery == "true" {
		pod.watcher(req, resp)

		return
	}

	resArr, err := etcdServiceAppPod.GetAllFromResource(podEtcdKey)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	podsRes := make([]Pod, len(resArr))
	for index, res := range resArr {
		var pod Pod
		if err = json.Unmarshal(res, &pod); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			return
		}

		podsRes[index] = pod
	}

	err = resp.WriteEntity(podsRes)
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}

func (pod *Pod) watcher(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")
	fieldSelector := req.QueryParameter("fieldSelector")

	if watchQuery != "true" {
		return
	}

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")

	watchChan, closeChanFunc, err := etcdServiceAppPod.GetWatchChannel(podEtcdKey)
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}
	defer closeChanFunc()
	defer resp.CloseNotify()

	log.Println("Client pod watcher started")

	for {
		select {
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			for _, event := range watchResp.Events {
				log.Printf("watch: %s executed on %s with value %s\n", event.Type, string(event.Kv.Key), string(event.Kv.Value))

				if fieldSelector == "" {
					fmt.Fprintf(resp, "Type: %s Value: %s\n", event.Type, string(event.Kv.Value))
					} else if validateFieldSelector(fieldSelector, string(event.Kv.Value)) {
						fmt.Fprintf(resp, "Type: %s Value: %s\n", event.Type, string(event.Kv.Value))
				}

				resp.Flush()
			}

		case <-req.Request.Context().Done():
			log.Println("Connection closed")
			return
		}
	}
}
