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

const namespaceEtcdKey = "/namespaces"

var (
	setupNamespaces         = [...]string{"default", "kube-system"}
	etcdServiceAppNamespace etcd.EtcdService
)

type Namespace struct {
	Metadata ResourceMetadata `json:"metadata" yaml:"metadata"`

	Kind string `json:"kind" yaml:"kind"`
}

func (namespace *Namespace) Register(etcdServersEndpoints string) {
	log.Println("rest api namespace register")

	etcdServiceAppNamespace = etcd.NewEtcdService(etcdServersEndpoints)

	ws := new(restful.WebService)

	ws.Filter(kubeapi_logger.LoggerMiddleware)

	ws.Path("/namespaces").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/").To(namespace.getNamespaces))

	ws.Route(ws.POST("/").To(namespace.createNamespace).
		Param(ws.BodyParameter("Namespace", "a Namespace resource (JSON)").DataType("rest.Namespace")))

	ws.Route(ws.GET("/{namespace}/pods").To(namespace.getPods).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))

	ws.Route(ws.GET("/{namespace}/services").To(namespace.getServices).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.QueryParameter("watch", "boolean for watching resource").DataType("bool").DefaultValue("false")).
		Param(ws.QueryParameter("fieldSelector", "field selector for resource").DataType("string").DefaultValue("")))

	ws.Route(ws.GET("/{namespace}/pods/{name}").To(namespace.getPod).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")))

	ws.Route(ws.GET("/{namespace}/services/{name}").To(namespace.getService).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.PathParameter("name", "name of the service").DataType("string")))

	ws.Route(ws.POST("/{namespace}/pods").To(namespace.createPod).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.BodyParameter("Pod", "a Pod resource (JSON)").DataType("rest.Pod")))

	ws.Route(ws.POST("/{namespace}/services").To(namespace.createService).Filter(validateNamespaceExists).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.BodyParameter("Service", "a Service resource (JSON)").DataType("rest.Service")))

	ws.Route(ws.PATCH("/{namespace}/pods/{name}/status").To(namespace.updateStatus).Filter(validateNamespaceExists).
		Param(ws.PathParameter("name", "name of the pod").DataType("string")).
		Param(ws.PathParameter("namespace", "namespace").DataType("string")).
		Param(ws.BodyParameter("PodStatus", "a Pod status resource (JSON)").DataType("rest.PodStatus")))

	restful.Add(ws)

	setupDefaultNamespaces()
}

func setupDefaultNamespaces() {
	log.Printf("creating setup namespaces %v", setupNamespaces)

	for _, namespaceName := range setupNamespaces {
		namespace := Namespace{
			Kind: "Namespace",
			Metadata: ResourceMetadata{
				CreationTimestamp: time.Now().Format(time.RFC3339),
				Name:              namespaceName,
				UID:               uuid.NewString(),
			},
		}

		namespaceBytes, err := json.Marshal(namespace)
		if err != nil {
			panic("unable to create setup namespaces")
		}

		err = etcdServiceAppNamespace.PutResource(fmt.Sprintf("%s/%s", namespaceEtcdKey, namespaceName), string(namespaceBytes))
		if err != nil {
			panic("unable to create setup namespaces")
		}
	}
}

func validateNamespaceExists(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	namespaceQuery := req.PathParameter("namespace")
	log.Printf("validating namespace exists %s", namespaceQuery)

	_, err := etcdServiceAppNamespace.GetResource(fmt.Sprintf("%s/%s", namespaceEtcdKey, namespaceQuery))
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			err = resp.WriteErrorString(http.StatusBadRequest, "namespace not exists")
			if err != nil {
				log.Printf("Error while sending error message: %v", err)
			}

			return
		}

		log.Printf("Error while getting namespace from etcd: %v", err)

		return
	}

	chain.ProcessFilter(req, resp)
}

func (namespace *Namespace) getAllResourceInNamespace(req *restful.Request, resp *restful.Response, etcdKey string) {
	watchQuery := req.QueryParameter("watch")

	if watchQuery == "true" {
		namespace.watcher(req, resp, etcdKey)

		return
	}

	namespaceQuery := req.PathParameter("namespace")
	resArr, err := etcdServiceAppNamespace.GetAllFromResource(fmt.Sprintf("%s/%s", etcdKey, namespaceQuery))
	if err != nil {
		if strings.Contains(err.Error(), "key not found") {
			err = resp.WriteEntity([]Pod{})
			if err != nil {
				fmt.Printf("error while sending error: %v", err)
			}

			return
		}

		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	resourcesArr := make([]interface{}, len(resArr))
	for index, res := range resArr {
		var resource interface{}
		if err = json.Unmarshal(res, &resource); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				fmt.Printf("error while sending error: %v", err)
			}

			return
		}

		resourcesArr[index] = resource
	}

	err = resp.WriteEntity(resourcesArr)
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) getSingleResourceInNamespace(req *restful.Request, resp *restful.Response, etcdKey string) {
	name := req.PathParameter("name")
	namespaceQuery := req.PathParameter("namespace")

	res, err := etcdServiceAppNamespace.GetResource(fmt.Sprintf("%s/%s/%s", etcdKey, namespaceQuery, name))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	var resource interface{}
	if err = json.Unmarshal(res, &resource); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity(resource)
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) watcher(req *restful.Request, resp *restful.Response, etcdKey string) {
	fieldSelector := req.QueryParameter("fieldSelector")
	namespaceQuery := req.PathParameter("namespace")

	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache")
	resp.Header().Set("Connection", "keep-alive")

	watchChan, closeChanFunc, err := etcdServiceAppNamespace.GetWatchChannel(fmt.Sprintf("%s/%s", etcdKey, namespaceQuery))
	if err != nil {
		err = resp.WriteErrorString(http.StatusBadRequest, err.Error())
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
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
				fmt.Printf("error while sending error: %v", err)
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

func (namespace *Namespace) createResourceInNamespace(req *restful.Request, resp *restful.Response, etcdKey string, resource interface{}) {
	namespaceQuery := req.PathParameter("namespace")

	type GenericResource struct {
		Metadata ResourceMetadata
	}

	resourceStruct, ok := resource.(GenericResource)
	if !ok {
		err := resp.WriteError(http.StatusBadRequest, fmt.Errorf("error converting resource"))
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	if resourceStruct.Metadata.Namespace == "" {
		resourceStruct.Metadata.Namespace = namespaceQuery
	}

	resourceStruct.Metadata.CreationTimestamp = time.Now().Format(time.RFC3339)

	if resourceStruct.Metadata.UID == "" {
		resourceStruct.Metadata.UID = uuid.NewString()
	}

	podBytes, err := json.Marshal(resourceStruct)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = etcdServiceAppNamespace.PutResource(fmt.Sprintf("%s/%s/%s", podEtcdKey, resourceStruct.Metadata.Namespace, resourceStruct.Metadata.Name), string(podBytes))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) getPods(req *restful.Request, resp *restful.Response) {
	namespace.getAllResourceInNamespace(req, resp, podEtcdKey)
}

func (namespace *Namespace) getPod(req *restful.Request, resp *restful.Response) {
	namespace.getSingleResourceInNamespace(req, resp, podEtcdKey)
}

func (namespace *Namespace) createPod(req *restful.Request, resp *restful.Response) {
	newPod := new(Pod)
	err := req.ReadEntity(newPod)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	if err = newPod.initLastAppliedConfigurations(); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	namespace.createResourceInNamespace(req, resp, podEtcdKey, newPod)
}

func (namespace *Namespace) updateStatus(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	namespaceQuery := req.PathParameter("namespace")

	res, err := etcdServiceAppNamespace.GetResource(fmt.Sprintf("%s/%s/%s", podEtcdKey, namespaceQuery, name))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	var podRes Pod
	if err = json.Unmarshal(res, &podRes); err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	newPodStatus := new(PodStatus)
	err = req.ReadEntity(newPodStatus)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	podRes.Status = *newPodStatus

	podResBytes, err := json.Marshal(podRes)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = etcdServiceAppNamespace.PutResource(fmt.Sprintf("%s/%s/%s", podEtcdKey, namespaceQuery, name), string(podResBytes))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) getNamespaces(_ *restful.Request, resp *restful.Response) {
	resArr, err := etcdServiceAppNamespace.GetAllFromResource(namespaceEtcdKey)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	namespacesResArr := make([]Namespace, len(resArr))
	for index, res := range resArr {
		var namespaceRes Namespace
		if err = json.Unmarshal(res, &namespaceRes); err != nil {
			err = resp.WriteError(http.StatusBadRequest, err)
			if err != nil {
				fmt.Printf("error while sending error: %v", err)
			}

			return
		}

		namespacesResArr[index] = namespaceRes
	}

	err = resp.WriteEntity(namespacesResArr)
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) createNamespace(req *restful.Request, resp *restful.Response) {
	newNamespace := new(Namespace)
	err := req.ReadEntity(newNamespace)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	newNamespace.Metadata.CreationTimestamp = time.Now().Format(time.RFC3339)

	if newNamespace.Metadata.UID == "" {
		newNamespace.Metadata.UID = uuid.NewString()
	}

	namespaceBytes, err := json.Marshal(newNamespace)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = etcdServiceAppNamespace.PutResource(fmt.Sprintf("%s/%s", namespaceEtcdKey, newNamespace.Metadata.Name), string(namespaceBytes))
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	err = resp.WriteEntity("success")
	if err != nil {
		fmt.Printf("error while sending error: %v", err)
	}
}

func (namespace *Namespace) createService(req *restful.Request, resp *restful.Response) {
	newService := new(Service)
	err := req.ReadEntity(newService)
	if err != nil {
		err = resp.WriteError(http.StatusBadRequest, err)
		if err != nil {
			fmt.Printf("error while sending error: %v", err)
		}

		return
	}

	namespace.createResourceInNamespace(req, resp, serviceEtcdKey, newService)
}

func (namespace *Namespace) getServices(req *restful.Request, resp *restful.Response) {
	namespace.getAllResourceInNamespace(req, resp, serviceEtcdKey)
}

func (namespace *Namespace) getService(req *restful.Request, resp *restful.Response) {
	namespace.getSingleResourceInNamespace(req, resp, serviceEtcdKey)
}