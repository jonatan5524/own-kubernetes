package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	etcdService "github.com/jonatan5524/own-kubernetes/pkg/kube-api/etcd"
	kubeapi_logger "github.com/jonatan5524/own-kubernetes/pkg/kube-api/logger"
)

type Pod struct {
	Name string
}

func (pod *Pod) Register() {
	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/pod").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/{name}").To(pod.get).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.PUT("/{name}").To(pod.put).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.DELETE("/{name}").To(pod.delete).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	restful.Add(ws)
}

func (pod *Pod) get(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	res, err := etcdService.GetResource(fmt.Sprintf("/pod/%s", name))
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

func (pod *Pod) put(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	podBytes, err := json.Marshal(&Pod{Name: name})
	if err != nil {
		panic(err)
	}

	err = etcdService.PutResource(fmt.Sprintf("/pod/%s", name), string(podBytes))
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

	err := etcdService.DeleteResource(fmt.Sprintf("/pod/%s", name))
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
