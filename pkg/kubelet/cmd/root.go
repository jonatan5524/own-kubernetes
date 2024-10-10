package cmd

import (
	"os"

	"github.com/jonatan5524/own-kubernetes/pkg/kubelet"
	"github.com/spf13/cobra"
)

var kubeAPIEndpoint string

var rootCmd = &cobra.Command{
	Use:   "kubelet",
	Short: "CLI util for running kubelet program",
	RunE: func(_ *cobra.Command, _ []string) error {
		app := kubelet.NewKubelet(kubeAPIEndpoint)
		defer app.Stop()

		if err := app.Setup(); err != nil {
			return err
		}

		if err := app.Run(); err != nil {
			return err
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&kubeAPIEndpoint, "kubernetes-api-endpoint", "", "kubernetes api endpoint")
	err := rootCmd.MarkFlagRequired("kubernetes-api-endpoint")
	if err != nil {
		panic(err)
	}
}
