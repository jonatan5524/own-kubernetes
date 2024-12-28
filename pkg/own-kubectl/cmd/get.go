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

var getNamespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "namespaces",
	RunE: func(cmd *cobra.Command, _ []string) error {
		namespaces, err := ownkubectl.GetNamespaces()
		if err != nil {
			return err
		}

		if len(namespaces) == 0 {
			fmt.Printf("No resource found\n")

			return nil
		}

		outputFormat, err := cmd.Flags().GetString(outputFlag)
		if err != nil {
			return err
		}

		if outputFormat == ownkubectl.OutputFormatJSON {
			namespacesJSONBytes, err := json.MarshalIndent(namespaces, "", " ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(namespacesJSONBytes))
		} else if outputFormat == ownkubectl.OutputFormatYAML {
			namespacesYAMLBytes, err := yaml.Marshal(namespaces)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(namespacesYAMLBytes))
		} else {
			ownkubectl.PrintNamespacesInTableFormat(namespaces)
		}

		return nil
	},
}

var getServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "services",
	RunE: func(cmd *cobra.Command, _ []string) error {
		namespace, err := cmd.Flags().GetString(namespaceFlag)
		if err != nil {
			return err
		}

		services, err := ownkubectl.GetServices(namespace)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			fmt.Printf("No resource found\n")

			return nil
		}

		outputFormat, err := cmd.Flags().GetString(outputFlag)
		if err != nil {
			return err
		}

		if outputFormat == ownkubectl.OutputFormatJSON {
			servicesJSONBytes, err := json.MarshalIndent(services, "", " ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(servicesJSONBytes))
		} else if outputFormat == ownkubectl.OutputFormatYAML {
			servicesYAMLBytes, err := yaml.Marshal(services)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(servicesYAMLBytes))
		} else {
			ownkubectl.PrintServicesInTableFormat(services, outputFormat)
		}

		return nil
	},
}

var getEndpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "endpoints",
	RunE: func(cmd *cobra.Command, _ []string) error {
		namespace, err := cmd.Flags().GetString(namespaceFlag)
		if err != nil {
			return err
		}

		endpoints, err := ownkubectl.GetEndpoints(namespace)
		if err != nil {
			return err
		}

		if len(endpoints) == 0 {
			fmt.Printf("No resource found\n")

			return nil
		}

		outputFormat, err := cmd.Flags().GetString(outputFlag)
		if err != nil {
			return err
		}

		if outputFormat == ownkubectl.OutputFormatJSON {
			endpointsJSONBytes, err := json.MarshalIndent(endpoints, "", " ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(endpointsJSONBytes))
		} else if outputFormat == ownkubectl.OutputFormatYAML {
			endpointsYAMLBytes, err := yaml.Marshal(endpoints)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", string(endpointsYAMLBytes))
		} else {
			ownkubectl.PrintEndpointsInTableFormat(endpoints)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.AddCommand(getPodsCmd)
	getPodsCmd.Flags().StringP(outputFlag, "o", "",
		fmt.Sprintf("output format: %s, %s, %s", ownkubectl.OutputFormatWide, ownkubectl.OutputFormatYAML, ownkubectl.OutputFormatJSON))
	getPodsCmd.Flags().StringP(namespaceFlag, "n", defaultNamespace, "pod's namespace")

	getCmd.AddCommand(getNamespacesCmd)
	getNamespacesCmd.Flags().StringP(outputFlag, "o", "",
		fmt.Sprintf("output format: %s, %s, %s", ownkubectl.OutputFormatWide, ownkubectl.OutputFormatYAML, ownkubectl.OutputFormatJSON))

	getCmd.AddCommand(getServicesCmd)
	getServicesCmd.Flags().StringP(outputFlag, "o", "",
		fmt.Sprintf("output format: %s, %s, %s", ownkubectl.OutputFormatWide, ownkubectl.OutputFormatYAML, ownkubectl.OutputFormatJSON))
	getServicesCmd.Flags().StringP(namespaceFlag, "n", defaultNamespace, "service namespace")

	getCmd.AddCommand(getEndpointsCmd)
	getEndpointsCmd.Flags().StringP(outputFlag, "o", "",
		fmt.Sprintf("output format: %s, %s, %s", ownkubectl.OutputFormatWide, ownkubectl.OutputFormatYAML, ownkubectl.OutputFormatJSON))
	getEndpointsCmd.Flags().StringP(namespaceFlag, "n", defaultNamespace, "endpoint namespace")
}
