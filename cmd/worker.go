package cmd

import (
	"cbs/pkg/model"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/spf13/cobra"
)

var ()

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.AddCommand(showWorkerCmd)
}

var workerCmd = &cobra.Command{
	Use:     "worker",
	Aliases: []string{"w"},
	Short:   "this is worker client, you know for worker.",
	Long:    "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var showWorkerCmd = &cobra.Command{
	Use:   "show",
	Short: "show workers",
	Long:  "\nyou know for show workers!",
	Run: func(cmd *cobra.Command, args []string) {
		initApp()
		switch len(args) {
		case 0:
			showWorker(cmd, args)
		default:
			cmd.Help()
		}
	},
}

func showWorker(cmd *cobra.Command, args []string) {
	workers, err := requestC.WorkerQuery(model.WorkerInput{})
	if err != nil {
		panic(err)
	}
	for _, worker := range workers {
		cmd.Println(tea.Prettify(worker))
		// fmt.Println(requestC.WorkerHcUpdate(worker.ID))
	}
}
