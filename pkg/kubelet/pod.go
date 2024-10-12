package kubelet

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/google/uuid"
	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kube_containerd "github.com/jonatan5524/own-kubernetes/pkg/kubelet/containerd"
)

const (
	defaultNamespace                     = "default"
	defaultPodLoggingLocation            = "/home/user/kubernetes/log/pod/%s/%s.log"
	defaultPodResolvConfLocation         = "/home/user/kubernetes/kubelet/pod/%s/resolv.conf"
	defaultPodContainerHostnameLocation  = "/home/user/kubernetes/kubelet/pod/%s/%s/hostname"
	defaultPodContainerEtcdHostsLocation = "/home/user/kubernetes/kubelet/pod/%s/etc-hosts"
)

func ListenForPodCreation(kubeAPIEndpoint string, hostname string) error {
	log.Printf("started watch on pod from kube API")

	resp, err := http.Get(fmt.Sprintf(
		"%s/pods/?watch=true&fieldSelector=spec.nodeName=%s",
		kubeAPIEndpoint,
		hostname,
	),
	)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not success from kube api server")
	}

	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error sending request: %v", err)
		}

		line = strings.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		log.Printf("event: %s", line)

		pod, err := CreatePod([]byte(line))
		if err != nil {
			return err
		}

		log.Printf("Pod %s is created and started", pod.Metadata.UID)
	}
}

func CreatePod(manifest []byte) (*kubeapi_rest.Pod, error) {
	var pod kubeapi_rest.Pod

	err := yaml.Unmarshal(manifest, &pod)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pod manifest, %v", err)
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = defaultNamespace
	}

	pod.Metadata.UID = uuid.New().String()

	for _, container := range pod.Spec.Containers {
		containerID, err := kube_containerd.CreateContainer(
			&container,
			kube_containerd.CreateContainerSpec{
				LogLocation:        fmt.Sprintf(defaultPodLoggingLocation, pod.Metadata.UID, container.Name),
				ResolvConfLocation: fmt.Sprintf(defaultPodResolvConfLocation, pod.Metadata.UID),
				HostnameLocation:   fmt.Sprintf(defaultPodContainerHostnameLocation, pod.Metadata.UID, container.Name),
				EtcHostsLocation:   fmt.Sprintf(defaultPodContainerEtcdHostsLocation, pod.Metadata.UID),
				HostNetwork:        pod.Spec.HostNetwork,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to create container %v", err)
		}

		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, kubeapi_rest.ContainerStatus{
			ContainerID: containerID,
			Image:       container.Image,
			Name:        container.Name,
		})

		log.Printf("Pod %s container %s created and started", pod.Metadata.Name, container.Name)
	}

	return &pod, nil
}
