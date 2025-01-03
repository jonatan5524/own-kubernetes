package pod

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/google/uuid"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kube_containerd "github.com/jonatan5524/own-kubernetes/pkg/kubelet/containerd"
	kubelet_net "github.com/jonatan5524/own-kubernetes/pkg/kubelet/net"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const (
	defaultPodLoggingLocation            = "/home/user/kubernetes/log/containers/%s.log"
	defaultPodResolvConfLocation         = "/home/user/kubernetes/kubelet/pod/%s/resolv.conf"
	defaultPodContainerHostnameLocation  = "/home/user/kubernetes/kubelet/pod/%s/%s/hostname"
	defaultPodContainerEtcdHostsLocation = "/home/user/kubernetes/kubelet/pod/%s/etc-hosts"
	// defaultPauseContainerImage           = "docker.io/jonatan5524/own-kubernetes:pause"
	defaultPauseContainerImage = "registry.k8s.io/pause:3.9"
	defaultNetNamespacePath    = "/proc/%d/ns/net"
	defaultIPCNamespacePath    = "/proc/%d/ns/ipc"
	podRunningPhase            = "Running"
	podFailedPhase             = "Failed"
	podPendingPhase            = "Pending"
	podTerminatingPhase        = "Terminating"
	podUnknownPhase            = "Unknown"
	defaultReconcileTimeout    = 30
)

func UpdatePodStatus(kubeAPIEndpoint string, podName string, namespace string, podStatus kubeapi_rest.PodStatus) error {
	log.Printf("update pod %s status for api", podName)

	podStatusBytes, err := json.Marshal(podStatus)
	if err != nil {
		return fmt.Errorf("error parsing pod status: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/namespaces/%s/pods/%s/status", kubeAPIEndpoint, namespace, podName),
		bytes.NewBuffer(podStatusBytes),
	)
	if err != nil {
		return fmt.Errorf("error creating request for pod status update: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending pod status update: %v", err)
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

func deletePodAPI(kubeAPIEndpoint string, namespace string, name string) error {
	log.Printf("delete pod %s for api", name)

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/namespaces/%s/pods/%s", kubeAPIEndpoint, namespace, name),
		&bytes.Buffer{},
	)
	if err != nil {
		return fmt.Errorf("error creating request for pod delete: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending pod delete: %v", err)
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

func UpdatePod(kubeAPIEndpoint string, pod kubeapi_rest.Pod) error {
	log.Printf("update pod %s for api", pod.Metadata.Name)

	podBytes, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("error parsing pod: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/namespaces/%s/pods", kubeAPIEndpoint, pod.Metadata.Namespace),
		bytes.NewBuffer(podBytes),
	)
	if err != nil {
		return fmt.Errorf("error creating request for pod update: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending pod update: %v", err)
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

func ListenForPod(kubeAPIEndpoint string, hostname string, podCIDR string, podBridgeName string) error {
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

		log.Printf("pod  event for pods: %s %s", typeEvent, value)

		var pod kubeapi_rest.Pod
		err = yaml.Unmarshal([]byte(value), &pod)
		if err != nil {
			log.Printf("error parsing pod from event: %v", err)

			continue
		}

		if typeEvent == "PUT" {
			if pod.Status.Phase == podRunningPhase && strings.Contains(line, kubeapi_rest.LastAppliedConfigurationAnnotationKey) {
				equal, err := compareLastAppliedToCurrentPod(pod)
				if err != nil {
					log.Printf("error comparing last applied: %v", err)

					continue
				}

				if equal {
					log.Printf("Pod is not changed from last applied annotation")

					continue
				}

				log.Printf("Pod has changed starts creation")
			}

			if pod.Status.Phase == podPendingPhase {
				go createPod(pod, podCIDR, podBridgeName, kubeAPIEndpoint)
			} else if pod.Status.Phase == podTerminatingPhase {
				go deletePod(pod, kubeAPIEndpoint)
			}
		}
	}
}

func Reconcile(kubeAPIEndpoint string, hostname string) {
	log.Printf("starting pod reconciliation")

	for {
		pods, err := getPods(kubeAPIEndpoint, hostname)
		if err != nil {
			log.Printf("error getting pods from api %v", err)

			continue
		}

		for _, pod := range pods {
			newPhase, err := getStatus(pod)
			if err != nil {
				log.Printf("error figuring out pod status %v", err)
			}

			if pod.Status.Phase != podTerminatingPhase && newPhase != pod.Status.Phase {
				pod.Status.Phase = newPhase

				if err := UpdatePodStatus(kubeAPIEndpoint,
					pod.Metadata.Name,
					pod.Metadata.Namespace,
					pod.Status,
				); err != nil {
					log.Printf("error updating pod status to api %v", err)

					continue
				}
			}
		}

		time.Sleep(time.Second * defaultReconcileTimeout)
	}
}

func getStatus(pod rest.Pod) (string, error) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		isRunning, err := kube_containerd.IsContainerRunning(containerStatus.ContainerID)
		if err != nil {
			return podUnknownPhase, err
		}

		if !isRunning {
			return podFailedPhase, nil
		}
	}

	return podRunningPhase, nil
}

func getPods(kubeAPIEndpoint string, hostname string) ([]rest.Pod, error) {
	log.Printf("getting all pods from api")

	var pods []kubeapi_rest.Pod
	resp, err := http.Get(fmt.Sprintf(
		"%s/pods?fieldSelector=%s",
		kubeAPIEndpoint,
		url.QueryEscape(fmt.Sprintf("spec.nodeName=%s", hostname)),
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

func deletePod(pod kubeapi_rest.Pod, kubeAPIEndpoint string) {
	log.Printf("started deleting pod %s/%s", pod.Metadata.Namespace, pod.Metadata.Name)

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if err := kube_containerd.DeleteContainer(containerStatus.ContainerID); err != nil {
			log.Printf("error deleting container %v", err)

			return
		}

		// Pause container
		if err := kube_containerd.DeleteContainer(pod.Metadata.UID); err != nil {
			log.Printf("error deleting container %v", err)

			return
		}
	}

	if err := deletePodAPI(kubeAPIEndpoint, pod.Metadata.Namespace, pod.Metadata.Name); err != nil {
		log.Printf("error sending delete pod to api %v", err)

		return
	}
}

func createPod(pod kubeapi_rest.Pod, podCIDR string, podBridgeName string, kubeAPIEndpoint string) {
	log.Printf("started creating pods %s/%s", pod.Metadata.Namespace, pod.Metadata.Name)

	podRes, err := CreatePodContainers(pod, podCIDR, podBridgeName)
	if err != nil {
		log.Printf("error creating pod: %v", err)

		return
	}

	if err := UpdatePodStatus(kubeAPIEndpoint, podRes.Metadata.Name, pod.Metadata.Namespace, podRes.Status); err != nil {
		log.Printf("error updating pod status to api: %v", err)

		return
	}

	log.Printf("Pod %s is created and started", podRes.Metadata.UID)
}

func compareLastAppliedToCurrentPod(podRes kubeapi_rest.Pod) (bool, error) {
	lastAppliedManifest := podRes.Metadata.Annotations[kubeapi_rest.LastAppliedConfigurationAnnotationKey]

	podRes.Status = kubeapi_rest.PodStatus{}
	podRes.Metadata.Annotations = make(map[string]string)
	podRes.Metadata.CreationTimestamp = ""
	podRes.Metadata.UID = ""

	var lastAppliedPodRes kubeapi_rest.Pod
	if err := json.Unmarshal([]byte(lastAppliedManifest), &lastAppliedPodRes); err != nil {
		return false, fmt.Errorf("error parsing last applied pod: %v", err)
	}

	return reflect.DeepEqual(lastAppliedPodRes, podRes), nil
}

func CreatePodContainers(pod kubeapi_rest.Pod, podCIDR string, podBridgeName string) (*kubeapi_rest.Pod, error) {
	if pod.Metadata.UID == "" {
		pod.Metadata.UID = uuid.NewString()
	}

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
		containerStatusName := fmt.Sprintf("own_k8s_%s_%s_%s_%s", container.Name, pod.Metadata.Name, pod.Metadata.Namespace, pod.Metadata.UID)

		containerID, err := kube_containerd.CreateContainer(
			&container,
			kube_containerd.CreateContainerSpec{
				LogLocation:          fmt.Sprintf(defaultPodLoggingLocation, containerStatusName),
				ResolvConfLocation:   fmt.Sprintf(defaultPodResolvConfLocation, pod.Metadata.UID),
				HostnameLocation:     fmt.Sprintf(defaultPodContainerHostnameLocation, pod.Metadata.UID, containerStatusName),
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
			Name:        containerStatusName,
		})

		log.Printf("container %s created and started", containerStatusName)
	}

	ip, err := kubelet_net.ConfigurePodNetwork(
		pod.Metadata.UID,
		podBridgeName,
		podCIDR,
		fmt.Sprintf(defaultNetNamespacePath, pauseContainerPID),
		pod.Spec.HostNetwork,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to configure pod network %v", err)
	}

	pod.Status.PodIP = ip
	pod.Status.Phase = podRunningPhase

	return &pod, nil
}

func createPauseContainer(podID string, podName string, namespace string, isHostNetwork bool) (uint32, error) {
	log.Printf("starting pause container for %s", podName)

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
