package cmd

import (
	"fmt"

	ownkubectl "github.com/jonatan5524/own-kubernetes/pkg/own-kubectl"
	"github.com/spf13/cobra"
)

const (
	namespaceDeleteFlag    = "namespace"
	defaultNamespaceDelete = "default"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete resources",
}

var deletePodsCmd = &cobra.Command{
	Use:   "pods",
	Short: "pods",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, err := cmd.Flags().GetString(namespaceDeleteFlag)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return fmt.Errorf("pod name must be specify")
		}

		err = ownkubectl.DeleteResource(namespace, "pods", args[0])
		if err != nil {
			return err
		}

		fmt.Println("success")

		return nil
	},
}

var deleteServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "services",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, err := cmd.Flags().GetString(namespaceDeleteFlag)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return fmt.Errorf("pod name must be specify")
		}

		err = ownkubectl.DeleteResource(namespace, "services", args[0])
		if err != nil {
			return err
		}

		fmt.Println("success")

		return nil
	},
}

var deleteEndpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, err := cmd.Flags().GetString(namespaceDeleteFlag)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return fmt.Errorf("pod name must be specify")
		}

		err = ownkubectl.DeleteResource(namespace, "endpoints", args[0])
		if err != nil {
			return err
		}

		fmt.Println("success")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.AddCommand(deletePodsCmd)
	deletePodsCmd.Flags().StringP(namespaceDeleteFlag, "n", defaultNamespaceDelete, "pod's namespace")

	deleteCmd.AddCommand(deleteServicesCmd)
	deleteServicesCmd.Flags().StringP(namespaceDeleteFlag, "n", defaultNamespaceDelete, "service namespace")

	deleteCmd.AddCommand(deleteEndpointsCmd)
	deleteEndpointsCmd.Flags().StringP(namespaceDeleteFlag, "n", defaultNamespaceDelete, "endpoint namespace")
}
