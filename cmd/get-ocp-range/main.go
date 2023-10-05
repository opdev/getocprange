package main

import (
	"fmt"
	"os"

	"github.com/opdev/getocprange"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "get-ocp-version <kubeVersionRange>",
	Short: "get-ocp-verion",
	Long:  `get-ocp-version derives a range of OCP versions from a given range of Kubernetes Version. It uses Mastermind/semver/v3 under the hood.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		resultOCPRange, err := getocprange.GetOCPRange(args[0])
		if err != nil {
			return err
		}
		fmt.Println(resultOCPRange)
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
