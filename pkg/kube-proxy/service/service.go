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
	clusterip "github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service/clusterIP"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service/endpoint"
	nodeport "github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service/nodePort"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
	"gopkg.in/yaml.v3"
)

func ListenForService(kubeAPIEndpoint string, clusterIPCIDR string, podCIDR string) error {
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

		typeEvent, value, err := utils.GetTypeAndValueFromEvent(line)
		if err != nil {
			log.Printf("error getting type and value from event: %v", err)

			continue
		}

		log.Printf("service event for services: %s %s", typeEvent, value)

		var service kubeapi_rest.Service
		err = yaml.Unmarshal([]byte(value), &service)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		if typeEvent == "PUT" {
			go createService(kubeAPIEndpoint, service, clusterIPCIDR, podCIDR)
		} else {
			go deleteService(service)
		}
	}
}

func ListenForPodRunning(kubeAPIEndpoint string, hostname string) error {
	log.Printf("started watch on pod from kube API")

	resp, err := http.Get(fmt.Sprintf(
		"%s/pods/?watch=true&fieldSelector=%s",
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

		typeEvent, value, err := utils.GetTypeAndValueFromEvent(line)
		if err != nil {
			log.Printf("error getting type and value from event: %v", err)

			continue
		}

		log.Printf("service event for pods: %s %s", typeEvent, value)

		var pod kubeapi_rest.Pod
		err = yaml.Unmarshal([]byte(value), &pod)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		if typeEvent == "PUT" {
			if pod.Status.Phase == "Running" {
				go conditionalCreateEndpoints(pod, kubeAPIEndpoint)
			} else {
				go endpoint.DeleteEndpointAddressIfExists(pod, kubeAPIEndpoint)
			}
		} else {
			go endpoint.DeleteEndpointAddressIfExists(pod, kubeAPIEndpoint)
		}
	}
}

func conditionalCreateEndpoints(pod kubeapi_rest.Pod, kubeAPIEndpoint string) {
	log.Printf("checking if new created pod need to create an endpoint")

	services, err := getAllServices(kubeAPIEndpoint)
	if err != nil {
		log.Printf("error getting all services: %v", err)

		return
	}

	for _, service := range services {
		key, value := getSelectorKeyAndValue(service.Spec.Selector)
		val, ok := pod.Metadata.Labels[key]

		if ok && val == value {
			endpoint.CreateEndpoints(kubeAPIEndpoint, service, []kubeapi_rest.Pod{pod})
		}
	}
}

func deleteService(service kubeapi_rest.Service) {
	log.Printf("deleteing service %s/%s", service.Metadata.Namespace, service.Metadata.Name)

	for _, port := range service.Spec.Ports {
		if err := clusterip.DeleteClusterIPService(
			service.Metadata.Name,
			service.Metadata.Namespace,
			port.Name,
		); err != nil {
			log.Printf("error deleting service: %v", err)

			return
		}
	}
}

func createService(kubeAPIEndpoint string, service kubeapi_rest.Service, clusterIPCIDR string, podCIDR string) {
	log.Printf("creating service %s/%s", service.Metadata.Namespace, service.Metadata.Name)

	key, value := getSelectorKeyAndValue(service.Spec.Selector)
	pods, err := getSelectorPods(kubeAPIEndpoint, key, value, service.Metadata.Namespace)
	if err != nil {
		log.Printf("error in getting pods from selector: %v", err)

		return
	}

	updated := false
	for index, port := range service.Spec.Ports {
		if !clusterip.CheckIfClusterIPServiceExists(service.Metadata.Namespace, service.Metadata.Name, port.Name) {
			updated = true
			if service.Spec.ClusterIP == "" {
				var clusterIP string
				clusterIP, err = clusterip.CreateNewClusterIP(
					clusterIPCIDR,
					podCIDR,
					service.Metadata.Namespace,
					service.Metadata.Name,
					port.Port,
					port.Name,
					kubeAPIEndpoint,
				)
				if err == nil {
					service.Spec.ClusterIP = clusterIP
				}
			} else {
				err = clusterip.CreateClusterIP(
					clusterIPCIDR,
					podCIDR,
					service.Metadata.Namespace,
					service.Metadata.Name,
					port.Port,
					port.Name,
					service.Spec.ClusterIP,
				)
			}
			if err != nil {
				log.Printf("error creating clusterIP: %v", err)

				return
			}

			if service.Spec.Type == "NodePort" &&
				!nodeport.CheckIfNodePortServiceExists(service.Metadata.Namespace, service.Metadata.Name, port.Name) {

				nodeport, err := nodeport.CreateNodePort(
					service.Metadata.Namespace,
					service.Metadata.Name,
					port.Name,
					kubeAPIEndpoint,
				)
				if err != nil {
					log.Printf("error creating NodePort: %v", err)

					return
				}

				service.Spec.Ports[index].NodePort = nodeport
			}
		}
	}

	if updated {
		if err := updateService(kubeAPIEndpoint, service); err != nil {
			log.Printf("error sending service update to api: %v", err)

			return
		}
		
		log.Printf("Service created")
		
		endpoint.CreateEndpoints(kubeAPIEndpoint, service, pods)
		
		log.Printf("Service %s is created ", service.Metadata.UID)
	}
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

func getAllServices(kubeAPIEndpoint string) ([]kubeapi_rest.Service, error) {
	var services []kubeapi_rest.Service
	resp, err := http.Get(fmt.Sprintf(
		"%s/services",
		kubeAPIEndpoint,
	),
	)
	if err != nil {
		return services, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return services, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return services, nil
		}

		return services, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return services, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &services)
	if err != nil {
		return services, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return services, nil
}

func updateService(kubeAPIEndpoint string, service kubeapi_rest.Service) error {
	log.Printf("update service %s for api", service.Metadata.Name)

	serviceBytes, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("error parsing service: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/namespaces/%s/services/%s", kubeAPIEndpoint, service.Metadata.Namespace, service.Metadata.Name),
		bytes.NewBuffer(serviceBytes),
	)
	if err != nil {
		return fmt.Errorf("error creating request for service update: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending service update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}

		return fmt.Errorf("request failed with status code: %d %s", resp.StatusCode, string(body))
	}

	return nil
}
