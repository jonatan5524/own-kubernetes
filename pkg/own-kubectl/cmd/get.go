package cmd

import (
	"encoding/json"
	"fmt"

	ownkubectl "github.com/jonatan5524/own-kubernetes/pkg/own-kubectl"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

		if len(pods) == 0 {
			fmt.Printf("No resource found in %s namespace\n", namespace)

			return nil
		}

		outputFormat, err := cmd.Flags().GetString(outputFlag)
		if err != nil {
			return err
		}

		if outputFormat == ownkubectl.OutputFormatJSON {
			podsJSONBytes, err := json.MarshalIndent(pods, "", " ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(podsJSONBytes))
		} else if outputFormat == ownkubectl.OutputFormatYAML {
			podsYAMLBytes, err := yaml.Marshal(pods)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(podsYAMLBytes))
		} else {
			ownkubectl.PrintPodsInTableFormat(pods, outputFormat)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getPodsCmd)
	getPodsCmd.Flags().StringP(namespaceFlag, "n", defaultNamespace, "pod's namespace")
	getPodsCmd.Flags().StringP(outputFlag, "o", "",
		fmt.Sprintf("output format: %s, %s, %s", ownkubectl.OutputFormatWide, ownkubectl.OutputFormatYAML, ownkubectl.OutputFormatJSON))
}
