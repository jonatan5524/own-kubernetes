package cmd

import (
	"fmt"

	ownkubectl "github.com/jonatan5524/own-kubernetes/pkg/own-kubectl"
	"github.com/spf13/cobra"
)

const (
	namespaceFlag    = "namespace"
	outputFlag       = "output"
	defaultNamespace = "default"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get resources",
}

var getPodsCmd = &cobra.Command{
	Use:   "pods",
	Short: "pods",
	RunE: func(cmd *cobra.Command, _ []string) error {
		namespace, err := cmd.Flags().GetString(namespaceFlag)
		if err != nil {
			return err
		}

		pods, err := ownkubectl.GetPods(namespace)
		if err != nil {
			return err
		}

		outputFormat, err := cmd.Flags().GetString(outputFlag)
		if err != nil {
			return err
		}
		ownkubectl.PrintPodsInTableFormat(pods, outputFormat)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getPodsCmd)
	getPodsCmd.Flags().StringP(namespaceFlag, "n", defaultNamespace, "pod's namespace")
	getPodsCmd.Flags().StringP(outputFlag, "o", "", fmt.Sprintf("output format: %s", ownkubectl.OutputFormatWide))
}
