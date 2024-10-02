package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/jonatan5524/own-kubernetes/pkg"
	etcdService "github.com/jonatan5524/own-kubernetes/pkg/kube-api/etcd"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
)

type Pod struct {
	Kind string `json:"kind" yaml:"kind"`

	Metadata struct {
		Name      string `json:"name" yaml:"name"`
		Namespace string `json:"namespace" yaml:"namespace"`
		UID       string `json:"uid" yaml:"uid"`
	} `json:"metadata" yaml:"metadata"`

	Spec struct {
		Containers []Container `json:"containers" yaml:"containers"`
	} `json:"spec" yaml:"spec"`
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

func (pod *Pod) Register() {
	log.Println("rest api pod register")

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/pod").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(pod.watcher).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool")))

	ws.Route(ws.GET("/{name}").To(pod.get).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.POST("/").To(pod.create).
		Param(ws.BodyParameter("Pod", "a Pod resource (JSON)").DataType("rest.Pod")))

	ws.Route(ws.DELETE("/{name}").To(pod.delete).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	restful.Add(ws)
}

func (pod *Pod) get(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	res, err := etcdService.GetResource(fmt.Sprintf("%s/%s", pkg.POD_ETCD_KEY, name))
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}

	var podRes Pod
	if err = json.Unmarshal(res, &podRes); err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity(podRes)
	if err != nil {
		log.Fatalf("err while sending response: %v", err)
	}
}

func (pod *Pod) watcher(req *restful.Request, resp *restful.Response) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery != "true" {
		return
	}

	log.Println("Client pod watcher started")

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")

	watchChan, closeChanFunc, err := etcdService.GetWatchChannel(pkg.POD_ETCD_KEY)
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}
	defer closeChanFunc()
	defer resp.CloseNotify()

	for {
		select {
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				log.Fatal("error watcher")
			}

			for _, event := range watchResp.Events {
				fmt.Fprintf(resp, "%s executed on %q with value %q\n", event.Type, event.Kv.Key, event.Kv.Value)
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
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}
	podBytes, err := json.Marshal(newPod)
	if err != nil {
		panic(err)
	}

	err = etcdService.PutResource(fmt.Sprintf("%s/%s", pkg.POD_ETCD_KEY, newPod.Metadata.Name), string(podBytes))
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		log.Fatalf("err while sending error: %v", err)
	}
}

func (pod *Pod) delete(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")

	err := etcdService.DeleteResource(fmt.Sprintf("%s/%s", pkg.POD_ETCD_KEY, name))
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			log.Fatalf("err while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		log.Fatalf("err while sending error: %v", err)
	}
}
