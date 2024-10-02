package kubelet

import (
	"fmt"
	"log"

	"gopkg.in/yaml.v3"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kube_containerd "github.com/jonatan5524/own-kubernetes/pkg/kubelet/containerd"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const defaultNamespace = "default"

func CreatePod(manifest []byte) (*kubeapi_rest.Pod, error) {
	var pod kubeapi_rest.Pod

	err := yaml.Unmarshal(manifest, &pod)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pod manifest, %v", err)
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = defaultNamespace
	}

	err = kube_containerd.CreateContainer(&pod.Spec.Containers[0], pod.Metadata.Namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to create container %v", err)
	}

	pod.Metadata.UID = utils.GenerateNewID(pod.Metadata.Name)

	log.Printf("Pod container created and started: %s %s", pod.Metadata.UID, pod.Metadata.Name)

	return &pod, nil
}
