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

	var resp *http.Response
	if kind == "Namespace" {
		resp, err = http.Post(
			fmt.Sprintf("%s/namespaces", os.Getenv("KUBE_API_ENDPOINT")),
			"application/json",
			bytes.NewReader(data),
		)
	} else {
		if namespace == "" {
			namespace = "default"
		}

		resp, err = http.Post(
			fmt.Sprintf("%s/namespaces/%s/%ss", os.Getenv("KUBE_API_ENDPOINT"), namespace, strings.ToLower(kind)),
			"application/json",
			bytes.NewReader(data),
		)
	}

	if err != nil {
		return err
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
