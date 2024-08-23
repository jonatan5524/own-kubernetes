package cmd

import (
	"fmt"

	ownkubectl "github.com/jonatan5524/own-kubernetes/pkg/own-kubectl"
	"github.com/spf13/cobra"
)

var file string

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new resource",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := ownkubectl.CreateResource(file); err != nil {
			return err
		}

		fmt.Println("success")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&file, "file", "", "manifest file of the resource")
	err := createCmd.MarkFlagRequired("file")
	if err != nil {
		panic(err)
	}
}
