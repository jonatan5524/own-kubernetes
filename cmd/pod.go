package cmd

import (
	"fmt"
	"log"

	"github.com/jonatan5524/own-kubernetes/pkg/pod"
	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "The command line tool to run commands on pods",
}

var (
	podImageRegistry string
	podName          string
)
var createPodCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new pod and run",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := pod.NewPodAndRun(podImageRegistry, podName); err != nil {
			return err
		}

		return nil
	},
}

var listPodCmd = &cobra.Command{
	Use:   "list",
	Short: "lists existing pods",
	RunE: func(cmd *cobra.Command, args []string) error {
		runningPods, err := pod.ListRunningPods()
		if err != nil {
			return err
		}

		if len(runningPods) == 0 {
			fmt.Println("There are no running pods")
		}

		for _, pod := range runningPods {
			fmt.Println(pod)
		}

		return nil
	},
}

var killPodCmd = &cobra.Command{
	Use:   "kill",
	Short: "kill existing pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := pod.KillPod(podName)
		if err != nil {
			return err
		}

		log.Println(id + " deleted")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(podCmd)
	podCmd.AddCommand(listPodCmd)

	podCmd.AddCommand(createPodCmd)
	createPodCmd.Flags().StringVar(&podImageRegistry, "registry", "", "image registry to pull (required)")
	createPodCmd.MarkFlagRequired("registry")
	createPodCmd.Flags().StringVar(&podName, "name", "nameless", "the pod name")

	podCmd.AddCommand(killPodCmd)
	killPodCmd.Flags().StringVar(&podName, "id", "", "the pod id (required)")
	killPodCmd.MarkFlagRequired("id")
}
