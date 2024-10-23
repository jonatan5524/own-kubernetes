package ownkubectl

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
)

const OutputFormatWide = "wide"

func PrintPodsInTableFormat(pods []rest.Pod, outputFormat string) {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 5, ' ', 0)
	if outputFormat == "" {
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")
	} else if outputFormat == OutputFormatWide {
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	}

	for _, pod := range pods {
		if outputFormat == "" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				pod.Metadata.Name,
				"Not yet supported",
				pod.Status.Phase,
				"Not yet supported",
				getAge(pod.Metadata.CreationTimestamp),
			)
		} else if outputFormat == OutputFormatWide {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				pod.Metadata.Name,
				"Not yet supported",
				pod.Status.Phase,
				"Not yet supported",
				getAge(pod.Metadata.CreationTimestamp),
				pod.Status.PodIP,
				pod.Spec.NodeName,
			)
		}
	}

	w.Flush()
}

func getAge(timeStampStr string) string {
	now := time.Now()
	timeStamp, err := time.Parse(time.RFC3339, timeStampStr)
	if err != nil {
		return ""
	}

	duration := now.Sub(timeStamp)

	if duration.Hours() > 24 {
		days := roundTime(duration.Seconds() / 86400)
		hours := roundTime(duration.Hours()) - 24*days

		if hours >= 1 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}

		return fmt.Sprintf("%dd", days)
	}

	return fmt.Sprintf("%dh", roundTime(duration.Hours()))
}

func roundTime(input float64) int {
	var result float64
	if input < 0 {
		result = math.Ceil(input - 0.5)
	} else {
		result = math.Floor(input + 0.5)
	}
	// only interested in integer, ignore fractional
	i, _ := math.Modf(result)
	return int(i)
}

func GetAllPods() ([]rest.Pod, error) {
	var pods []rest.Pod

	resp, err := http.Get(
		fmt.Sprintf("%s/pods", os.Getenv("KUBE_API_ENDPOINT")),
	)
	if err != nil {
		return pods, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return pods, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pods, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &pods)
	if err != nil {
		return pods, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return pods, nil
}

func GetPods(namespace string) ([]rest.Pod, error) {
	var pods []rest.Pod

	resp, err := http.Get(
		fmt.Sprintf("%s/pods", os.Getenv("KUBE_API_ENDPOINT")),
	)
	if err != nil {
		return pods, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return pods, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pods, fmt.Errorf("error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &pods)
	if err != nil {
		return pods, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return pods, nil
}
