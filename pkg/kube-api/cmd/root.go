package cmd

import (
	"os"

	kubeapi "github.com/jonatan5524/own-kubernetes/pkg/kube-api"
	"github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"github.com/spf13/cobra"
)

var etcdServers string

var rootCmd = &cobra.Command{
	Use:   "kube-api",
	Short: "CLI util for running kubernetes api program",
	RunE: func(_ *cobra.Command, _ []string) error {
		app := kubeapi.NewKubeAPI(
			etcdServers,
			[]kubeapi.Rest{
				&rest.Pod{},
				&rest.Namespace{},
				&rest.Service{},
				&rest.Endpoint{},
			})
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
	rootCmd.Flags().StringVar(&etcdServers, "etcd-servers", "", "etcd servers endpoints")
	err := rootCmd.MarkFlagRequired("etcd-servers")
	if err != nil {
		panic(err)
	}
}
