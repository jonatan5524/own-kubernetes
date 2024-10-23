package ownkubectl

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

func CreateResource(file string) error {
	data, kind, namespace, err := utils.ReadResource(file, true)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/namespaces/%s/%ss", os.Getenv("KUBE_API_ENDPOINT"), namespace, strings.ToLower(kind)),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("error from api: %s %s", resp.Status, string(body))
	}

	return nil
}
