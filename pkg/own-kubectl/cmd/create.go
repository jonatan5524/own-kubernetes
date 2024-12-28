package cmd

import (
	"fmt"

	ownkubectl "github.com/jonatan5524/own-kubernetes/pkg/own-kubectl"
	"github.com/spf13/cobra"
)

const fileFlag = "file"

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new resource",
	RunE: func(cmd *cobra.Command, _ []string) error {
		filename, err := cmd.Flags().GetString(fileFlag)
		if err != nil {
			return err
		}

		if err := ownkubectl.CreateResource(filename); err != nil {
			return err
		}

		fmt.Println("success")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringP(fileFlag, "f", "", "manifest file of the resource")
	err := createCmd.MarkFlagRequired(fileFlag)
	if err != nil {
		panic(err)
	}
}
