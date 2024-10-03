package kubelet

import (
	"fmt"
	"log"

	"gopkg.in/yaml.v3"

	"github.com/google/uuid"
	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kube_containerd "github.com/jonatan5524/own-kubernetes/pkg/kubelet/containerd"
)

const (
	defaultNamespace          = "default"
	defaultPodLoggingLocation = "/home/user/kubernetes/log/pod/"
)

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
		containerID, err := kube_containerd.CreateContainer(&container, fmt.Sprintf("%s/%s/%s.log", defaultPodLoggingLocation, pod.Metadata.Name, container.Name))
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
