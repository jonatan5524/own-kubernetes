package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"gopkg.in/yaml.v3"
)

func ListenForEndpointCreation(kubeAPIEndpoint string, hostname string) error {
	log.Printf("started watch on endpoints from kube API")

	resp, err := http.Get(fmt.Sprintf(
		"%s/endpoints/?watch=true&fieldSelector=%s",
		kubeAPIEndpoint,
		url.QueryEscape(fmt.Sprintf("spec.nodeName=%s", hostname)),
	),
	)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}

		return fmt.Errorf("request failed with status code: %d %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("error parsing response: %v", err)

			continue
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		log.Printf("event: %s", line)

		var endpoint kubeapi_rest.Endpoint
		err = yaml.Unmarshal([]byte(line), &endpoint)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		go createEndpointIPTable(kubeAPIEndpoint, endpoint)
	}
}

func createEndpointIPTable(kubeAPIEndpoint string, endpoint kubeapi_rest.Endpoint) {
	log.Printf("creating iptable rules using endpoint")

	service, err := getRelatableService(kubeAPIEndpoint, endpoint.Metadata.Name, endpoint.Metadata.Namespace)
	if err != nil {
		log.Printf("error in getting relatable service: %v", err)

		return
	}

	if service.Spec.Type == "ClusterIP" {
	} else if service.Spec.Type == "NodePort" {
	}
}

func createEndpoints(kubeAPIEndpoint string, service kubeapi_rest.Service, pods []kubeapi_rest.Pod) {
	log.Printf("creating endpoints for service %s/%s", service.Metadata.Namespace, service.Metadata.Name)

	endpoint := kubeapi_rest.Endpoint{
		Metadata: kubeapi_rest.ResourceMetadata{
			Name:      service.Metadata.Name,
			Namespace: service.Metadata.Namespace,
		},
		Kind: "Endpoint",
		Subsets: []kubeapi_rest.EndpointSubset{
			{
				Ports: service.Spec.Ports,
			},
		},
	}

	endpoint.Subsets[0].Addresses = make([]kubeapi_rest.EndpointAddress, len(pods))
	for index, pod := range pods {
		endpoint.Subsets[0].Addresses[index] = kubeapi_rest.EndpointAddress{
			IP:       pod.Status.PodIP,
			NodeName: pod.Spec.NodeName,
			TargetRef: kubeapi_rest.TargetRef{
				Kind:      pod.Kind,
				Name:      pod.Metadata.Name,
				Namespace: pod.Metadata.Namespace,
				UID:       pod.Metadata.UID,
			},
		}
	}

	data, err := json.Marshal(endpoint)
	if err != nil {
		log.Printf("error marshaling endpoints: %v", err)

		return
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/namespaces/%s/endpoints", kubeAPIEndpoint, endpoint.Metadata.Namespace),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		log.Printf("error post endpoint to api: %v", err)

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("error from api: %s %s", resp.Status, string(body))

		return
	}
}

func getRelatableService(kubeAPIEndpoint string, name string, namespace string) (kubeapi_rest.Service, error) {
	var service kubeapi_rest.Service
	resp, err := http.Get(fmt.Sprintf(
		"%s/namespaces/%s/services/%s",
		kubeAPIEndpoint,
		namespace,
		name,
	),
	)
	if err != nil {
		return service, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return service, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return service, nil
		}

		return service, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return service, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &service)
	if err != nil {
		return service, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return service, nil
}
