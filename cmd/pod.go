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
	imageRegistry string
	name          string
)
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new pod and run",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := pod.NewPodAndRun(imageRegistry, name); err != nil {
			return err
		}

		return nil
	},
}

var listCmd = &cobra.Command{
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

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "kill existing pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := pod.KillPod(name)
		if err != nil {
			return err
		}

		log.Println(id + " deleted")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(podCmd)
	podCmd.AddCommand(listCmd)

	podCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&imageRegistry, "registry", "", "image registry to pull (required)")
	createCmd.MarkFlagRequired("registry")
	createCmd.Flags().StringVar(&name, "name", "nameless", "the pod name")

	podCmd.AddCommand(killCmd)
	killCmd.Flags().StringVar(&name, "id", "", "the pod id (required)")
	killCmd.MarkFlagRequired("id")
}
