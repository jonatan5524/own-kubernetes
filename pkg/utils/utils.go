package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func ReadResource(file string, convertToJSON bool) ([]byte, string, error) {
	var resourceData []byte

	resourceData, err := os.ReadFile(file)
	if err != nil {
		return resourceData, "", err
	}

	var resource struct {
		Kind string `json:"kind" yaml:"kind"`
	}
	err = yaml.Unmarshal(resourceData, &resource)
	if err != nil {
		return resourceData, "", fmt.Errorf("kind not found in yaml, %v", err)
	}

	if convertToJSON {
		resourceData, err = convertYAMLtoJSON(resourceData)
		if err != nil {
			return resourceData, "", fmt.Errorf("invalid YAML, %v", err)
		}
	}

	return resourceData, resource.Kind, nil
}

func convertYAMLtoJSON(data []byte) ([]byte, error) {
	var yamlData interface{}
	var jsonData []byte

	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return jsonData, err
	}

	return json.MarshalIndent(yamlData, "", "  ")
}

func GenerateNewID(name string) string {
	id := uuid.New()

	return fmt.Sprintf("%s-%s", name, id)
}

func CreateDirectory(path string, mode fs.FileMode) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(path, mode)
	}

	return nil
}

func CreateAndWriteToFile(path string, data string, mode fs.FileMode) error {
	if err := CreateDirectory(filepath.Dir(path), 0o755); err != nil {
		log.Fatalf("error creating %s: %v", path, err)
	}

	err := os.WriteFile(path, []byte(data), mode)
	if err != nil {
		return fmt.Errorf("error creating %s: %v", path, err)
	}

	return nil
}
