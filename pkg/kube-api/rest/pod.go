package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/etcd"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
	"github.com/tidwall/gjson"
)

const (
	podEtcdKey                            = "/pod"
	defaultNamespace                      = "default"
	LastAppliedConfigurationAnnotationKey = "last-applied-configuration"
)

var etcdServiceApp etcd.EtcdService

type Pod struct {
	Metadata PodMetadata `json:"metadata" yaml:"metadata"`

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

type PodMetadata struct {
	Annotations       map[string]string `json:"annotations" yaml:"annotations"`
	Name              string            `json:"name" yaml:"name"`
	Namespace         string            `json:"namespace" yaml:"namespace"`
	CreationTimestamp string            `json:"creationTimestamp" yaml:"creationTimestamp"`
	UID               string            `json:"uid" yaml:"uid"`
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
}

func (pod *Pod) Register(etcdServersEndpoints string) {
	log.Println("rest api pod register")

	etcdServiceApp = etcd.NewEtcdService(etcdServersEndpoints)

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/pods").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(pod.getAll).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))

	ws.Route(ws.GET("/{name}").To(pod.get).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.POST("/").To(pod.create).
		Param(ws.BodyParameter("Pod", "a Pod resource (JSON)").DataType("rest.Pod")))

	ws.Route(ws.PUT("/status/{name}").To(pod.updateStatus).
		Param(ws.BodyParameter("PodStatus", "a Pod status resource (JSON)").DataType("rest.PodStatus")))

	ws.Route(ws.DELETE("/{name}").To(pod.delete).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	restful.Add(ws)
}

func (pod *Pod) get(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	res, err := etcdServiceApp.GetResource(fmt.Sprintf("%s/%s", podEtcdKey, name))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	var podRes Pod
	if err = json.Unmarshal(res, &podRes); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = resp.WriteEntity(podRes)
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}

func (pod *Pod) getAll(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery == "true" {
		pod.watcher(req, resp)

		return
	}

	resArr, err := etcdServiceApp.GetAllFromResource(podEtcdKey)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	var podsRes []Pod
	for _, res := range resArr {
		var pod Pod
		if err = json.Unmarshal(res, &pod); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				resp.WriteError(http.StatusInternalServerError, err)
			}

			return
		}

		podsRes = append(podsRes, pod)
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

	watchChan, closeChanFunc, err := etcdServiceApp.GetWatchChannel(podEtcdKey)
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

func (pod *Pod) create(req *restful.Request, resp *restful.Response) {
	newPod := new(Pod)
	err := req.ReadEntity(newPod)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = defaultNamespace
	}

	if err = newPod.initLastAppliedConfigurations(); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}
	newPod.Metadata.CreationTimestamp = time.Now().Format(time.RFC3339)

	if newPod.Metadata.UID == "" {
		newPod.Metadata.UID = uuid.NewString()
	}

	podBytes, err := json.Marshal(newPod)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = etcdServiceApp.PutResource(fmt.Sprintf("%s/%s", podEtcdKey, newPod.Metadata.Name), string(podBytes))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
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

func (pod *Pod) delete(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")

	err := etcdServiceApp.DeleteResource(fmt.Sprintf("%s/%s", podEtcdKey, name))
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}

func (pod *Pod) updateStatus(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")

	res, err := etcdServiceApp.GetResource(fmt.Sprintf("%s/%s", podEtcdKey, name))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	var podRes Pod
	if err = json.Unmarshal(res, &podRes); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	newPodStatus := new(PodStatus)
	err = req.ReadEntity(newPodStatus)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	podRes.Status = *newPodStatus

	podResBytes, err := json.Marshal(podRes)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = etcdServiceApp.PutResource(fmt.Sprintf("%s/%s", podEtcdKey, name), string(podResBytes))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			resp.WriteError(http.StatusInternalServerError, err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
	}
}
