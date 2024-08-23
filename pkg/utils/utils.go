package utils

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func ConvertYAMLtoJSON(data []byte) (string, error) {
	var yamlData interface{}

	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return "", err
	}

	jsonData, err := json.MarshalIndent(yamlData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
