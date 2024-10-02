package kubelet

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

func ReadAndStartSystemManifests(systemManifestPath string) error {
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
			pod, err := CreatePod(data)
			if err != nil {
				return err
			}

			log.Printf("Pod %s is created and started", pod.Metadata.UID)
		}
	}

	return nil
}
