package cmd

import (
	"cbs/config"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	configPath = pflag.StringP("config", "c", "~/.cbs/", "config file dir,default is ~/.cbs/")
	debug = pflag.BoolP("debug", "d", false, "enable debug mode")
	pflag.Parse()
	rootCmd.AddCommand(apiServerCmd)
}

var (
	configPath *string
	debug      *bool
)

var apiServerCmd = &cobra.Command{
	Use:  "manager",
	Long: "manager",
	Run: func(cmd *cobra.Command, args []string) {

		if *debug {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}
		apiConfig := config.LoadApiConfig(*configPath)
		fmt.Println(tea.Prettify(apiConfig))
	},
}
