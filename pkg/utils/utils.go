package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadResource(file string, convertToJSON bool) ([]byte, string, string, error) {
	var resourceData []byte

	resourceData, err := os.ReadFile(file)
	if err != nil {
		return resourceData, "", "", err
	}

	var resource struct {
		Kind string `json:"kind" yaml:"kind"`

		Metadata struct {
			Namespace string `json:"namespace" yaml:"namespace"`
		} `json:"metadata" yaml:"metadata"`
	}
	err = yaml.Unmarshal(resourceData, &resource)
	if err != nil {
		return resourceData, "", "", fmt.Errorf("kind not found in yaml, %v", err)
	}

	if convertToJSON {
		resourceData, err = convertYAMLtoJSON(resourceData)
		if err != nil {
			return resourceData, "", "", fmt.Errorf("invalid YAML, %v", err)
		}
	}

	return resourceData, resource.Kind, resource.Metadata.Namespace, nil
}

func convertYAMLtoJSON(data []byte) ([]byte, error) {
	var yamlData interface{}
	var jsonData []byte

	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return jsonData, err
	}

	return json.MarshalIndent(yamlData, "", "  ")
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

func ReplaceAtIndex(str string, replacement rune, index int) string {
	return str[:index] + string(replacement) + str[index+1:]
}

func ExecuteCommand(command string, waitToComplete bool) error {
	log.Printf("Executing: %s", command)

	splitedCommand := strings.Split(command, " ")
	cmd := exec.Command(splitedCommand[0], splitedCommand[1:]...)

	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error running command: %v", err)
	}

	if len(stdout) > 0 {
		log.Printf("command output: %s", string(stdout))
	}

	return nil
}
