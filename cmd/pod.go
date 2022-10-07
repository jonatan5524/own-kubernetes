package cmd

import (
	"fmt"
	"time"

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
	Short: "Create new pod",
	RunE: func(cmd *cobra.Command, args []string) error {
		pod, err := pod.NewPod(imageRegistry, name)
		if err != nil {
			return err
		}

		fmt.Printf("pod created: %s\n", pod.Id)
		fmt.Printf("starting pod\n")

		runningPod, err := pod.Run()
		if err != nil {
			return err
		}

		fmt.Printf("pod started: %s\n", pod.Id)

		time.Sleep(3 * time.Second)

		fmt.Printf("killing pod\n")

		code, err := runningPod.Kill()
		if err != nil {
			return err
		}
		fmt.Printf("pod killed: %s\n", pod.Id)

		fmt.Printf("%s exited with status: %d\n", runningPod.Pod.Id, code)

		pod.Delete()

		fmt.Printf("container deleted: %s\n", pod.Id)

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

		for _, pod := range runningPods {
			fmt.Println(pod)
		}

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
}
