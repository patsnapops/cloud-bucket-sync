package cmd

import (
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	configPath = pflag.StringP("config", "c", "~/.cbs/", "config file dir,default is ~/.cbs/")
	debug := pflag.BoolP("debug", "d", false, "enable debug mode")
	pflag.Parse()
	if *debug {
		log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
	} else {
		log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
	}
	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(bucketCmd)
}

var (
	configPath *string
)

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
