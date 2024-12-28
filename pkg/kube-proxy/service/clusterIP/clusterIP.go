package clusterip

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/iptables"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

func CreateNewClusterIP(clusterIPCIDR string, podCIDR string, namespace string, serviceName string, servicePort int, portName string, kubeAPIEndpoint string) (string, error) {
	log.Printf("creating iptables clusterIP")

	clusterIP, err := getNextAvailableClusterIP(kubeAPIEndpoint, clusterIPCIDR)
	if err != nil {
		return "", err
	}

	log.Printf("configuring iptables clusterIP %s", clusterIP)
	if err := iptables.NewClusterIPService(clusterIP, podCIDR, namespace, serviceName, servicePort, portName); err != nil {
		return "", err
	}

	return clusterIP, nil
}

func CreateClusterIP(clusterIPCIDR string, podCIDR string, namespace string, serviceName string, servicePort int, portName string, clusterIP string) error {
	log.Printf("creating iptables clusterIP")

	log.Printf("configuring iptables clusterIP %s", clusterIP)
	if err := iptables.NewClusterIPService(clusterIP, podCIDR, namespace, serviceName, servicePort, portName); err != nil {
		return err
	}

	return nil
}

func CheckIfClusterIPServiceExists(namespace string, name string, portName string) bool {
	return iptables.CheckIfClusterIPServiceExists(namespace, name, portName)
}

func CheckIfClusterIPServiceEndpointExists(namespace string, podName string, portName string) bool {
	return iptables.CheckIfClusterIPServiceEndpointExists(namespace, podName, portName)
}

func AddEndpointToClusterIP(namespace string, serviceName string, podName string, portName string, podIP string, podPort int, probability float32) error {
	log.Printf("adding pod addres %s:%d to service %s/%s", podIP, podPort, namespace, serviceName)

	return iptables.AddEndpointToServiceChain(namespace, serviceName, podName, portName, podIP, podPort, probability)
}

func getNextAvailableClusterIP(kubeAPIEndpoint string, clusterIPCIDR string) (string, error) {
	log.Printf("getting available cluster ip")

	services, err := getAllServices(kubeAPIEndpoint)
	if err != nil {
		return "", err
	}

	hosts, err := utils.HostsFromCIDR(clusterIPCIDR)
	if err != nil {
		return "", err
	}

	for _, ip := range hosts {
		found := false

		for _, service := range services {
			if service.Spec.ClusterIP == ip {
				found = true

				break
			}
		}

		if !found {
			return ip, nil
		}
	}

	return "", err
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

func DeleteClusterIPService(serviceName string, namespace string, portName string) error {
	log.Printf("deleting iptables clusterIP")

	if err := iptables.DeleteServiceClusterIP(serviceName, namespace, portName); err != nil {
		return err
	}

	return nil
}
