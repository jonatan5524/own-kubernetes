package endpoint

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/iptables"
	clusterip "github.com/jonatan5524/own-kubernetes/pkg/kube-proxy/service/clusterIP"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
	"gopkg.in/yaml.v3"
)

func ListenForEndpoint(kubeAPIEndpoint string, hostname string) error {
	log.Printf("started watch on endpoints from kube API")

	resp, err := http.Get(fmt.Sprintf(
		"%s/endpoints/?watch=true",
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

		log.Printf("endpoint event for endpoints: %s %s", typeEvent, value)

		var endpoint kubeapi_rest.Endpoint
		err = yaml.Unmarshal([]byte(value), &endpoint)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		if typeEvent == "PUT" {
			found := false
			for _, subset := range endpoint.Subsets {
				for _, address := range subset.Addresses {
					if address.NodeName == hostname {
						found = true
						break
					}
				}

				if found {
					break
				}
			}

			if found {
				go createEndpointIPTable(kubeAPIEndpoint, endpoint)
			}
		}
	}
}

func createEndpointIPTable(kubeAPIEndpoint string, endpoint kubeapi_rest.Endpoint) {
	log.Printf("creating iptable rules using endpoint")

	service, err := getRelatableService(kubeAPIEndpoint, endpoint.Metadata.Name, endpoint.Metadata.Namespace)
	if err != nil {
		log.Printf("error in getting relatable service: %v", err)

		return
	}

	for _, subset := range endpoint.Subsets {
		for index, address := range subset.Addresses {
			for _, port := range subset.Ports {
				if !clusterip.CheckIfClusterIPServiceEndpointExists(service.Metadata.Namespace, address.TargetRef.Name, port.Name) {
					log.Printf("len: %d", len(subset.Addresses))
					log.Printf("index: %d", index)
					log.Printf("probability: %f", float32(len(subset.Addresses)-index)/float32(len(subset.Addresses)))
					err := iptables.CreateEndpointChain(
						service.Metadata.Namespace,
						service.Metadata.Name,
						address.TargetRef.Name,
						port.Name,
						address.IP,
						port.Port,
					)
					if err != nil {
						log.Printf("error creating endpoint: %v", err)

						return
					}

					err = clusterip.AddEndpointToClusterIP(
						service.Metadata.Namespace,
						service.Metadata.Name,
						address.TargetRef.Name,
						port.Name,
						address.IP,
						port.Port,
						float32(len(subset.Addresses)-index)/float32(len(subset.Addresses)),
					)
					if err != nil {
						log.Printf("error adding pod to clusterIP: %v", err)

						return
					}
				}
			}
		}
	}
}

func DeleteEndpointAddressIfExists(pod kubeapi_rest.Pod, kubeAPIEndpoint string) {
	log.Printf("check if endpoint for pod %s/%s need to delete", pod.Metadata.Namespace, pod.Metadata.Name)

	endpoints, err := getAllEndpointsInNamespace(pod.Metadata.Namespace, kubeAPIEndpoint)
	if err != nil {
		log.Printf("error getting endpoints from namespace %v", err)

		return
	}

	deleteEndpointIndex := math.MaxInt
	deleteSubsetIndex := math.MaxInt
	deleteAddressIndex := math.MaxInt
	for indexEndpoint, endpoint := range endpoints {
		for indexSubset, subset := range endpoint.Subsets {
			for indexAddress, address := range subset.Addresses {
				if address.TargetRef.UID == pod.Metadata.UID {
					deleteEndpointIndex = indexEndpoint
					deleteSubsetIndex = indexSubset
					deleteAddressIndex = indexAddress

					break
				}
			}
		}
	}

	if deleteEndpointIndex != math.MaxInt &&
		deleteSubsetIndex != math.MaxInt &&
		deleteAddressIndex != math.MaxInt {
		log.Printf(
			"found endpoint %s/%s to remove the address pod from",
			endpoints[deleteEndpointIndex].Metadata.Namespace,
			endpoints[deleteEndpointIndex].Metadata.Name,
		)

		service, err := getRelatableService(
			kubeAPIEndpoint,
			endpoints[deleteEndpointIndex].Metadata.Name,
			endpoints[deleteEndpointIndex].Metadata.Namespace,
		)
		if err != nil {
			log.Printf("error in getting relatable service: %v", err)

			return
		}

		for _, subset := range endpoints[deleteEndpointIndex].Subsets {
			for range subset.Addresses {
				for _, port := range subset.Ports {
					if err := iptables.ClearClusterIPServiceFromEndpoints(
						service.Metadata.Name,
						service.Metadata.Namespace,
						port.Name,
					); err != nil {
						log.Printf("error clearing clusterip chain: %v", err)
					}
				}
			}
		}

		endpoints[deleteEndpointIndex].Subsets[deleteSubsetIndex].Addresses = append(
			endpoints[deleteEndpointIndex].Subsets[deleteSubsetIndex].Addresses[:deleteAddressIndex],
			endpoints[deleteEndpointIndex].Subsets[deleteSubsetIndex].Addresses[deleteAddressIndex+1:]...,
		)

		for _, subset := range endpoints[deleteEndpointIndex].Subsets {
			for index, address := range subset.Addresses {
				for _, port := range subset.Ports {
					err = clusterip.AddEndpointToClusterIP(
						service.Metadata.Namespace,
						service.Metadata.Name,
						address.TargetRef.Name,
						port.Name,
						address.IP,
						port.Port,
						float32(len(subset.Addresses)-index)/float32(len(subset.Addresses)),
					)
					if err != nil {
						log.Printf("error adding pod to clusterIP: %v", err)

						return
					}
				}
			}
		}

		if err := updateEndpointToAPI(endpoints[deleteEndpointIndex], kubeAPIEndpoint); err != nil {
			log.Printf("error updating endpoint %v", err)

			return
		}
	}
}

func updateEndpointToAPI(endpoint kubeapi_rest.Endpoint, kubeAPIEndpoint string) error {
	log.Printf("update endpoint %s for api", endpoint.Metadata.Name)

	endpointBytes, err := json.Marshal(endpoint)
	if err != nil {
		return fmt.Errorf("error parsing endpoint: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/namespaces/%s/endpoints/%s", kubeAPIEndpoint, endpoint.Metadata.Namespace, endpoint.Metadata.Name),
		bytes.NewBuffer(endpointBytes),
	)
	if err != nil {
		return fmt.Errorf("error creating request for endpoint update: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending endpoint update: %v", err)
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

func CreateEndpoints(kubeAPIEndpoint string, service kubeapi_rest.Service, pods []kubeapi_rest.Pod) {
	log.Printf("creating endpoints for service %s/%s", service.Metadata.Namespace, service.Metadata.Name)

	var method string
	var url string
	endpoint, err := getEndpoint(kubeAPIEndpoint, service.Metadata.Name, service.Metadata.Namespace)
	if err != nil {
		if err.Error() == "endpoint not found" {
			method = http.MethodPost
			endpoint = createNewEndpoint(service, pods)
			url = fmt.Sprintf("%s/namespaces/%s/endpoints", kubeAPIEndpoint, endpoint.Metadata.Namespace)
		} else {
			log.Printf("error getting existing endpoint: %v", err)

			return
		}
	} else {
		method = http.MethodPatch
		url = fmt.Sprintf("%s/namespaces/%s/endpoints/%s", kubeAPIEndpoint, endpoint.Metadata.Namespace, endpoint.Metadata.Name)
	}

	for _, pod := range pods {
		if pod.Status.PodIP != "" {
			endpoint.Subsets[0].Addresses = append(endpoint.Subsets[0].Addresses,
				kubeapi_rest.EndpointAddress{
					IP:       pod.Status.PodIP,
					NodeName: pod.Spec.NodeName,
					TargetRef: kubeapi_rest.TargetRef{
						Kind:      pod.Kind,
						Name:      pod.Metadata.Name,
						Namespace: pod.Metadata.Namespace,
						UID:       pod.Metadata.UID,
					},
				},
			)
		}
	}

	data, err := json.Marshal(endpoint)
	if err != nil {
		log.Printf("error marshaling endpoints: %v", err)

		return
	}

	req, err := http.NewRequest(
		method,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		log.Printf("error creating request for service update: %v", err)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error sending service update: %v", err)

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

func createNewEndpoint(service kubeapi_rest.Service, pods []kubeapi_rest.Pod) kubeapi_rest.Endpoint {
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

	return endpoint
}

func getEndpoint(kubeAPIEndpoint string, name string, namespace string) (kubeapi_rest.Endpoint, error) {
	var endpoint kubeapi_rest.Endpoint
	resp, err := http.Get(fmt.Sprintf(
		"%s/namespaces/%s/endpoints/%s",
		kubeAPIEndpoint,
		namespace,
		name,
	),
	)
	if err != nil {
		return endpoint, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return endpoint, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return endpoint, fmt.Errorf("endpoint not found")
		}

		return endpoint, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return endpoint, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &endpoint)
	if err != nil {
		return endpoint, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return endpoint, nil
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

func getAllEndpointsInNamespace(namespace string, kubeAPIEndpoint string) ([]kubeapi_rest.Endpoint, error) {
	var endpoints []kubeapi_rest.Endpoint
	resp, err := http.Get(fmt.Sprintf(
		"%s/namespaces/%s/endpoints",
		kubeAPIEndpoint,
		namespace,
	),
	)
	if err != nil {
		return endpoints, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return endpoints, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return endpoints, nil
		}

		return endpoints, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return endpoints, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &endpoints)
	if err != nil {
		return endpoints, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return endpoints, nil
}
