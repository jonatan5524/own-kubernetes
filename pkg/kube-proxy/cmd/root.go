package cmd

import (
	"os"

	kubeproxy "github.com/jonatan5524/own-kubernetes/pkg/kube-proxy"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kube-proxy",
	Short: "CLI util for running kube-proxy program",
	RunE: func(_ *cobra.Command, _ []string) error {
		app := kubeproxy.NewKubeProxy()
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
	// rootCmd.Flags().StringVar(&etcdServers, "etcd-servers", "", "etcd servers endpoints")
	// err := rootCmd.MarkFlagRequired("etcd-servers")
	// if err != nil {
	// 	panic(err)
	// }
}
