package ownkubectl

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

const (
	OutputFormatWide = "wide"
	OutputFormatYAML = "yaml"
	OutputFormatJSON = "json"
)

func PrintPodsInTableFormat(pods []rest.Pod, outputFormat string) {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)
	if outputFormat == "" {
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")
	} else if outputFormat == OutputFormatWide {
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	}

	for _, pod := range pods {
		if outputFormat == "" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				pod.Metadata.Name,
				"Not yet supported",
				pod.Status.Phase,
				"Not yet supported",
				getAge(pod.Metadata.CreationTimestamp),
			)
		} else if outputFormat == OutputFormatWide {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				pod.Metadata.Name,
				"Not yet supported",
				pod.Status.Phase,
				"Not yet supported",
				getAge(pod.Metadata.CreationTimestamp),
				pod.Status.PodIP,
				pod.Spec.NodeName,
			)
		}
	}

	w.Flush()
}

func PrintServicesInTableFormat(services []rest.Service, outputFormat string) {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)
	if outputFormat == "" {
		fmt.Fprintln(w, "NAME\tTYPE\tCLUSTER-IP\tEXTERNAL-IP\tPORT(S)\tAGE")
	} else if outputFormat == OutputFormatWide {
		fmt.Fprintln(w, "NAME\tTYPE\tCLUSTER-IP\tEXTERNAL-IP\tPORT(S)\tAGE\tSELECTOR")
	}

	for _, service := range services {
		if outputFormat == "" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				service.Metadata.Name,
				service.Spec.Type,
				service.Spec.ClusterIP,
				"Not yet supported",
				getFormattedPorts(service),
				getAge(service.Metadata.CreationTimestamp),
			)
		} else if outputFormat == OutputFormatWide {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				service.Metadata.Name,
				service.Spec.Type,
				service.Spec.ClusterIP,
				"Not yet supported",
				getFormattedPorts(service),
				getAge(service.Metadata.CreationTimestamp),
				"Not yet supported",
			)
		}
	}

	w.Flush()
}

func PrintEndpointsInTableFormat(endpoints []rest.Endpoint) {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)

	fmt.Fprintln(w, "NAME\tENDPOINTS\tAGE")

	for _, endpoint := range endpoints {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			endpoint.Metadata.Name,
			getFormattedAddresses(endpoint),
			getAge(endpoint.Metadata.CreationTimestamp),
		)
	}

	w.Flush()
}

func PrintNamespacesInTableFormat(namespaces []rest.Namespace) {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAGE")

	for _, namespace := range namespaces {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			namespace.Metadata.Name,
			"Not yet supported",
			getAge(namespace.Metadata.CreationTimestamp),
		)
	}

	w.Flush()
}

func getFormattedAddresses(endpoint rest.Endpoint) string {
	endpoints := ""

	for _, subset := range endpoint.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				endpoints += fmt.Sprintf("%s:%d,", address.IP, port.Port)
			}
		}
	}

	return endpoints[:len(endpoints)-1]
}

func getFormattedPorts(service rest.Service) string {
	ports := ""

	for _, portSpec := range service.Spec.Ports {
		ports += fmt.Sprintf("%d/%s,", portSpec.Port, portSpec.Protocol)
	}

	return ports[:len(ports)-1]
}

func getAge(timeStampStr string) string {
	now := time.Now()
	timeStamp, err := time.Parse(time.RFC3339, timeStampStr)
	if err != nil {
		return ""
	}

	duration := now.Sub(timeStamp)

	if duration.Hours() > 24 {
		days := roundTime(duration.Seconds() / 86400)
		hours := roundTime(duration.Hours()) - 24*days

		if hours >= 1 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}

		return fmt.Sprintf("%dd", days)
	}

	return fmt.Sprintf("%dh", roundTime(duration.Hours()))
}

func roundTime(input float64) int {
	var result float64
	if input < 0 {
		result = math.Ceil(input - 0.5)
	} else {
		result = math.Floor(input + 0.5)
	}
	// only interested in integer, ignore fractional
	i, _ := math.Modf(result)
	return int(i)
}

func getResource(path string) ([]byte, error) {
	var resources []byte

	resp, err := http.Get(path)
	if err != nil {
		return resources, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resources, fmt.Errorf("error reading response body: %v", err)
		}

		if strings.Contains(string(body), "key not found") {
			return resources, nil
		}

		return resources, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resources, fmt.Errorf("error reading response body: %v", err)
	}

	return body, nil
}

func GetPods(namespace string) ([]rest.Pod, error) {
	resources, err := getResource(
		fmt.Sprintf("%s/namespaces/%s/pods", os.Getenv("KUBE_API_ENDPOINT"), namespace),
	)
	if err != nil {
		return nil, err
	}

	var pods []rest.Pod
	err = json.Unmarshal(resources, &pods)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return pods, nil
}

func GetNamespaces() ([]rest.Namespace, error) {
	resources, err := getResource(
		fmt.Sprintf("%s/namespaces", os.Getenv("KUBE_API_ENDPOINT")),
	)
	if err != nil {
		return nil, err
	}

	var namespaces []rest.Namespace
	err = json.Unmarshal(resources, &namespaces)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return namespaces, nil
}

func GetServices(namespace string) ([]rest.Service, error) {
	resources, err := getResource(
		fmt.Sprintf("%s/namespaces/%s/services", os.Getenv("KUBE_API_ENDPOINT"), namespace),
	)
	if err != nil {
		return nil, err
	}

	var services []rest.Service
	err = json.Unmarshal(resources, &services)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return services, nil
}

func GetEndpoints(namespace string) ([]rest.Endpoint, error) {
	resources, err := getResource(
		fmt.Sprintf("%s/namespaces/%s/endpoints", os.Getenv("KUBE_API_ENDPOINT"), namespace),
	)
	if err != nil {
		return nil, err
	}

	var endpoints []rest.Endpoint
	err = json.Unmarshal(resources, &endpoints)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return endpoints, nil
}
