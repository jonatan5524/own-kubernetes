package pod

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
	kubelet_net "github.com/jonatan5524/own-kubernetes/pkg/kubelet/net"
)

const (
	defaultNamespace                     = "default"
	defaultPodLoggingLocation            = "/home/user/kubernetes/log/containers/%s.log"
	defaultPodResolvConfLocation         = "/home/user/kubernetes/kubelet/pod/%s/resolv.conf"
	defaultPodContainerHostnameLocation  = "/home/user/kubernetes/kubelet/pod/%s/%s/hostname"
	defaultPodContainerEtcdHostsLocation = "/home/user/kubernetes/kubelet/pod/%s/etc-hosts"
	defaultPauseContainerImage           = "docker.io/jonatan5524/own-kubernetes:pause"
	defaultNetNamespacePath              = "/proc/%d/ns/net"
	defaultIPCNamespacePath              = "/proc/%d/ns/ipc"
)

func ListenForPodCreation(kubeAPIEndpoint string, hostname string, podCIDR string, podBridgeName string) error {
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

		pod, err := CreatePod([]byte(line), podCIDR, podBridgeName)
		if err != nil {
			return err
		}

		log.Printf("Pod %s is created and started", pod.Metadata.UID)
	}
}

func CreatePod(manifest []byte, podCIDR string, podBridgeName string) (*kubeapi_rest.Pod, error) {
	var pod kubeapi_rest.Pod

	err := yaml.Unmarshal(manifest, &pod)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pod manifest, %v", err)
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = defaultNamespace
	}

	pod.Metadata.UID = uuid.New().String()

	pauseContainerPID, err := createPauseContainer(
		pod.Metadata.UID,
		pod.Metadata.Name,
		pod.Metadata.Namespace,
		pod.Spec.HostNetwork,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create pause container, %v", err)
	}

	for _, container := range pod.Spec.Containers {
		container.Name = fmt.Sprintf("own_k8s_%s_%s_%s_%s", container.Name, pod.Metadata.Name, pod.Metadata.Namespace, pod.Metadata.UID)

		containerID, err := kube_containerd.CreateContainer(
			&container,
			kube_containerd.CreateContainerSpec{
				LogLocation:          fmt.Sprintf(defaultPodLoggingLocation, container.Name),
				ResolvConfLocation:   fmt.Sprintf(defaultPodResolvConfLocation, pod.Metadata.UID),
				HostnameLocation:     fmt.Sprintf(defaultPodContainerHostnameLocation, pod.Metadata.UID, container.Name),
				EtcHostsLocation:     fmt.Sprintf(defaultPodContainerEtcdHostsLocation, pod.Metadata.UID),
				HostNetwork:          pod.Spec.HostNetwork,
				NetworkNamespacePath: fmt.Sprintf(defaultNetNamespacePath, pauseContainerPID),
				IPCNamespacePath:     fmt.Sprintf(defaultIPCNamespacePath, pauseContainerPID),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to create and start container %v", err)
		}

		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, kubeapi_rest.ContainerStatus{
			ContainerID: containerID,
			Image:       container.Image,
			Name:        container.Name,
		})

		log.Printf("Pod %s container %s created and started", pod.Metadata.Name, container.Name)
	}

	ip, err := kubelet_net.ConfigurePodNetwork(
		pod.Metadata.UID,
		podBridgeName,
		podCIDR,
		fmt.Sprintf(defaultNetNamespacePath, pauseContainerPID),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to configure pod network %v", err)
	}

	pod.Status.PodIP = ip

	return &pod, nil
}

func createPauseContainer(podID string, podName string, namespace string, isHostNetwork bool) (uint32, error) {
	log.Printf("starting pause container for %s", podID)

	cotnainerID, err := kube_containerd.CreateContainer(
		&kubeapi_rest.Container{
			Name:  fmt.Sprintf("own_k8s_POD_%s_%s_%s", podName, namespace, podID),
			Image: defaultPauseContainerImage,
		},
		kube_containerd.CreateContainerSpec{
			ContainerID:        podID,
			LogLocation:        fmt.Sprintf(defaultPodLoggingLocation, fmt.Sprintf("own_k8s_POD_%s_%s_%s", podName, namespace, podID)),
			ResolvConfLocation: fmt.Sprintf(defaultPodResolvConfLocation, podID),
			HostnameLocation:   fmt.Sprintf(defaultPodContainerHostnameLocation, podID, "pause"),
			EtcHostsLocation:   fmt.Sprintf(defaultPodContainerEtcdHostsLocation, podID),
			HostNetwork:        isHostNetwork,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("unable to create and start container %v", err)
	}

	log.Printf("pause container for pod %s  created and started", podID)

	return kube_containerd.GetContainerPID(cotnainerID)
}
