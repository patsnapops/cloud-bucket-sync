package cmd

import (
	"cbs/config"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/spf13/cobra"
)

var apiServerCmd = &cobra.Command{
	Use:  "api",
	Long: "api",
	Run: func(cmd *cobra.Command, args []string) {
		apiConfig := config.LoadApiConfig(*configPath)
		fmt.Println(tea.Prettify(apiConfig))
	},
}
