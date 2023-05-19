package cmd

import (
	"cbs/pkg/model"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(taskCmd)

	taskCmd.AddCommand(applyCmd)
	taskCmd.AddCommand(showCmd)
	taskCmd.AddCommand(execCmd)

}

var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"t"},
	Long:    "this is task client, you know for task.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "to create task",
	Long:  "\nyou know for submit a task!",
	Run: func(cmd *cobra.Command, args []string) {
		initApp()
		switch len(args) {
		case 0:
			applyTask(cmd, args)
		default:
			cmd.Help()
		}
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show tasks",
	Long:  "\nyou know for show tasks!",
	Run: func(cmd *cobra.Command, args []string) {
		initApp()
		switch len(args) {
		case 0:
			showTask(cmd, args)
		case 1:
			// show taskID
			taskID := args[0]
			task, err := requestC.TaskGet(taskID)
			if err != nil {
				panic(err)
			}
			// only show last 10 records
			records := task.Records
			if len(records) > 5 {
				task.Records = task.Records[len(task.Records)-5:]
			}
			fmt.Println(tea.Prettify(task))
			if len(records) > 5 {
				fmt.Println("only show last 5 record by create_at...")
			}
		default:
			cmd.Help()
		}
	},
}

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "to exec task",
	Long:  "\nyou know for exec task!",
	Run: func(cmd *cobra.Command, args []string) {
		initApp()
		switch len(args) {
		case 1:
			execTask(cmd, args)
		default:
			cmd.Help()
		}
	},
}

func showTask(cmd *cobra.Command, args []string) {
	tasks, err := requestC.TaskList()
	if err != nil {
		panic(err)
	}
	// table show
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetHeader([]string{"ID", "Name", "WorkerTag", "SyncMode", "Submitter", "Records"})
	for _, t := range tasks {
		taskName := t.Name
		// 处理换行
		// taskName = strings.Replace(taskName, " ", "-", -1)
		fmt.Println(tea.Prettify(t))
		recordStatus := ""
		running := 0
		success := 0
		failed := 0
		cancel := 0
		pending := 0
		for _, record := range t.Records {
			switch record.Status {
			case "running":
				running++
			case "success":
				success++
			case "failed":
				failed++
			case "cancel":
				cancel++
			case "pending":
				pending++
			}
		}
		recordStatus = fmt.Sprintf("pending:%d,running:%d,success:%d,failed:%d,cancel:%d", pending, running, success, failed, cancel)
		table.Append([]string{t.ID, taskName, string(t.Worker), string(t.SyncMode), t.Submitter, recordStatus})
	}
	table.SetFooter([]string{"", "", "", "", "count", tea.ToString(len(tasks))})
	table.Render()
}

func applyTask(cmd *cobra.Command, args []string) {
	file, _ := cmd.Flags().GetString("file")
	if file == "" {
		panic("file is empty")
	}
	// load json file to struct
	// var file = "task.json"
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var tasksJson []model.Task
	err = json.Unmarshal(data, &tasksJson)
	if err != nil {
		panic(err)
	}
	// apply task
	for _, taskJson := range tasksJson {
		// fmt.Println(tea.Prettify(taskJson))
		// continue
		taskID, err := requestC.TaskApply(taskJson)
		if err != nil {
			panic(err)
		}
		fmt.Println("taskID:", taskID)
	}
}

// exec task
func execTask(cmd *cobra.Command, args []string) {
	taskID := args[0]
	record, err := requestC.TaskExec(taskID)
	if err != nil {
		panic(err)
	}
	fmt.Println(tea.Prettify(record))
}
