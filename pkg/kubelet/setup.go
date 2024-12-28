package kubelet

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	kubelet_net "github.com/jonatan5524/own-kubernetes/pkg/kubelet/net"
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet/pod"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
	"gopkg.in/yaml.v3"
)

func readAndStartSystemManifests(systemManifestPath string, hostname string) ([]*rest.Pod, error) {
	log.Printf("Reading manifest in system %s", systemManifestPath)

	files, err := os.ReadDir(systemManifestPath)
	if err != nil {
		log.Fatal(err)
	}

	var pods []*rest.Pod

	for _, file := range files {
		log.Printf("Reading file %s", file.Name())

		data, kind, _, err := utils.ReadResource(filepath.Join(systemManifestPath, file.Name()), false)
		if err != nil {
			return pods, err
		}

		if kind == "Pod" {
			var podResManifest rest.Pod
			err = yaml.Unmarshal(data, &podResManifest)
			if err != nil {
				return pods, fmt.Errorf("error parsing pod from event: %v", err)
			}

			podRes, err := pod.CreatePodContainers(podResManifest, podCIDR, podBridgeName)
			if err != nil {
				return pods, err
			}

			podRes.Spec.NodeName = hostname
			pods = append(pods, podRes)

			log.Printf("Pod %s %s is created and started", podRes.Metadata.Name, podRes.Metadata.UID)
		}
	}

	return pods, err
}

func initCIDRPodNetwork(cidr string, bridgeName string) error {
	log.Printf("setting up pod cidr %s with bridge name %s", cidr, bridgeName)

	if !kubelet_net.IsNetDeviceExists(bridgeName) {
		if err := kubelet_net.CreateBridge(bridgeName, podCIDR); err != nil {
			return err
		}
	}

	return nil
}
