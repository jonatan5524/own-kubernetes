package ownkubectl

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func DeleteResource(namespace string, kind string, name string) error {
	if namespace == "" {
		namespace = "default"
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/namespaces/%s/%s/%s", os.Getenv("KUBE_API_ENDPOINT"), namespace, kind, name),
		bytes.NewBuffer([]byte{}),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		return fmt.Errorf("error from api: %s %s", resp.Status, string(body))
	}

	return nil
}
