package nodeport

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/iptables"
)

const (
	nodePortRangeStart = 30000
	nodePortRangeEnd   = 32767
)

func CreateNodePort(namespace string, serviceName string, portName string, kubeAPIEndpoint string) (int, error) {
	log.Printf("creating iptables nodeport")

	// TODO: not working
	port, err := getNextAvailableNodePort(kubeAPIEndpoint)
	if err != nil {
		return 0, err
	}

	if port == 0 {
		return 0, fmt.Errorf("not node port available left")
	}

	if err := iptables.NewNodePortService(namespace, serviceName, port, portName); err != nil {
		return 0, err
	}

	return port, nil
}

func CheckIfNodePortServiceExists(namespace string, name string, portName string) bool {
	return iptables.CheckIfNodePortServiceExists(namespace, name, portName)
}

// TODO: update node port to api
func getNextAvailableNodePort(kubeAPIEndpoint string) (int, error) {
	log.Printf("getting available node port")

	services, err := getAllServices(kubeAPIEndpoint)
	if err != nil {
		return 0, err
	}

	port := nodePortRangeStart
	for port <= nodePortRangeEnd {
		found := false
		for _, service := range services {
			for _, portService := range service.Spec.Ports {
				if portService.NodePort == port {
					found = true

					break
				}
			}
		}

		if !found {
			return port, nil
		}

		port++
	}

	return 0, err
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
