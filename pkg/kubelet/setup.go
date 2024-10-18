package kubelet

import (
	"log"
	"os"
	"path/filepath"

	kubelet_net "github.com/jonatan5524/own-kubernetes/pkg/kubelet/net"
	"github.com/jonatan5524/own-kubernetes/pkg/kubelet/pod"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

func readAndStartSystemManifests(systemManifestPath string) error {
	log.Printf("Reading manifest in system %s", systemManifestPath)

	files, err := os.ReadDir(systemManifestPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		log.Printf("Reading file %s", file.Name())

		data, kind, err := utils.ReadResource(filepath.Join(systemManifestPath, file.Name()), false)
		if err != nil {
			return err
		}

		if kind == "Pod" {
			pod, err := pod.CreatePod(data, podCIDR, podBridgeName)
			if err != nil {
				return err
			}

			log.Printf("Pod %s is created and started", pod.Metadata.UID)
		}
	}

	return nil
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
