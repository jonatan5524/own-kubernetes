package ownkubectl

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
	"gopkg.in/yaml.v2"
)

type resourceYaml struct {
	Kind string
}

func CreateResource(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var resource resourceYaml
	err = yaml.Unmarshal(data, &resource)
	if err != nil {
		return fmt.Errorf("kind not found in yaml, %v", err)
	}

	jsonData, err := utils.ConvertYAMLtoJSON(data)
	if err != nil {
		return fmt.Errorf("invalid YAML, %v", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/%s", os.Getenv("KUBE_API_ENDPOINT"), strings.ToLower(resource.Kind)),
		"application/json",
		bytes.NewReader([]byte(jsonData)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not success from kube api server")
	}

	return nil
}
