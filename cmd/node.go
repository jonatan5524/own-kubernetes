package cmd

import (
	"fmt"
	"log"

	"github.com/jonatan5524/own-kubernetes/pkg/node"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "The command line tool to run commands on nodes",
}

var (
	nodeName string
)
var createNodeCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new node and run",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := node.NewNodeAndRun(); err != nil {
			return err
		}

		return nil
	},
}

var listNodeCmd = &cobra.Command{
	Use:   "list",
	Short: "lists existing nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		runningNodes, err := node.ListRunningNodes()
		if err != nil {
			return err
		}

		if len(runningNodes) == 0 {
			fmt.Println("There are no running nodes")
		}

		for _, pod := range runningNodes {
			fmt.Println(pod)
		}

		return nil
	},
}

var killNodeCmd = &cobra.Command{
	Use:   "kill",
	Short: "kill existing node",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := node.KillNode(nodeName)
		if err != nil {
			return err
		}

		log.Println(id + " deleted")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)
	nodeCmd.AddCommand(listNodeCmd)
	nodeCmd.AddCommand(createNodeCmd)

	nodeCmd.AddCommand(killNodeCmd)
	killNodeCmd.Flags().StringVar(&nodeName, "id", "", "the pod id (required)")
	killNodeCmd.MarkFlagRequired("id")
}
