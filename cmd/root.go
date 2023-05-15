package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(bucketCmd)
}

var (
	rootCmd = &cobra.Command{
		Use:   "cbs",
		Short: "Welcome to use cbs",
		Long:  "CloudBucketSync is a powerful tool for syncing data between cloud storage buckets. Built in Go.\nyou should use `cbs [command] --help` to see the usage of each command.\n",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		Version: "v0.0.1-beta",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
