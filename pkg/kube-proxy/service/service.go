package service

import (
	"bufio"
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

func ListenForServiceCreation(kubeAPIEndpoint string) error {
	log.Printf("started watch on services from kube API")

	resp, err := http.Get(fmt.Sprintf(
		"%s/services/?watch=true",
		kubeAPIEndpoint,
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

		var service kubeapi_rest.Service
		err = yaml.Unmarshal([]byte(line), &service)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		go createService(kubeAPIEndpoint, service)

	}
}

func createService(kubeAPIEndpoint string, service kubeapi_rest.Service) {
	log.Printf("creating service %s/%s", service.Metadata.Namespace, service.Metadata.Name)

	key, value := getSelectorKeyAndValue(service.Spec.Selector)
	pods, err := getSelectorPods(kubeAPIEndpoint, key, value, service.Metadata.Namespace)
	if err != nil {
		log.Printf("error in getting pods from selector: %v", err)

		return
	}

	createEndpoints(kubeAPIEndpoint, service, pods)

	log.Printf("Service %s is created ", service.Metadata.UID)
}

func getSelectorKeyAndValue(selector map[string]string) (string, string) {
	for key, value := range selector {
		return key, value
	}

	return "", ""
}

func getSelectorPods(kubeAPIEndpoint string, selectorKey string, selectorValue string, namespace string) ([]kubeapi_rest.Pod, error) {
	var pods []kubeapi_rest.Pod
	resp, err := http.Get(fmt.Sprintf(
		"%s/namespaces/%s/pods?fieldSelector=%s",
		kubeAPIEndpoint,
		namespace,
		url.QueryEscape(fmt.Sprintf("metadata.labels.%s=%s", selectorKey, selectorValue)),
	),
	)
	if err != nil {
		return pods, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return pods, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return pods, nil
		}

		return pods, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pods, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &pods)
	if err != nil {
		return pods, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return pods, nil
}
