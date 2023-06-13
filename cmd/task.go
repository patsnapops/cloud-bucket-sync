package cmd

import (
	"cbs/pkg/model"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
)

var (
	taskFile string
	operator string
	syncMode string

	outputFormat string
)

func init() {
	rootCmd.AddCommand(taskCmd)

	taskCmd.AddCommand(applyCmd)
	taskCmd.AddCommand(showCmd)
	taskCmd.AddCommand(execCmd)
	taskCmd.PersistentFlags().StringVarP(&taskFile, "file", "f", "", "task file path, default is ./task.json")

	showCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format, support table/json")

	execCmd.Flags().StringVarP(&operator, "operator", "o", "cli", "task operator")
	execCmd.Flags().StringVarP(&syncMode, "sync-mode", "s", "", "task sync mode, support keepSync（real-time sync） syncOnce（one-time sync）")
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
			if outputFormat == "json" {
				tasks, err := requestC.TaskQuery(model.TaskInput{})
				if err != nil {
					panic(err)
				}
				for _, task := range tasks {
					fmt.Println(tea.Prettify(task))
				}
			} else {
				showTask(cmd, args)
			}
		case 1:
			// show taskID
			taskName := args[0]
			tasks, err := requestC.TaskQuery(model.TaskInput{
				Name: taskName,
			})
			if err != nil {
				panic(err)
			}
			for _, task := range tasks {
				fmt.Println(tea.Prettify(task))
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
	tasks, err := requestC.TaskQuery(model.TaskInput{})
	if err != nil {
		panic(err)
	}
	// table show
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetHeader([]string{"ID", "Name", "Size", "WorkerTag", "SyncMode", "Submitter", "Records"})
	for _, t := range tasks {
		var size int64
		taskName := t.Name
		// 处理换行
		// taskName = strings.Replace(taskName, " ", "-", -1)
		// fmt.Println(tea.Prettify(t))
		recordStatus := ""
		running := 0
		success := 0
		failed := 0
		cancel := 0
		pending := 0
		records, err := requestC.RecordQuery(model.RecordInput{
			TaskID: t.Id,
		})
		if err != nil {
			records = []model.Record{}
		}
		for _, record := range records {
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
			size += record.TotalSize
		}
		recordStatus = fmt.Sprintf("pending:%d,running:%d,success:%d,failed:%d,cancel:%d", pending, running, success, failed, cancel)
		table.Append([]string{t.Id, strings.ReplaceAll(taskName, " ", "/"), model.FormatSize(size), string(t.WorkerTag), string(t.SyncMode), t.Submitter, recordStatus})
	}
	table.SetFooter([]string{"", "", "", "", "", "count", tea.ToString(len(tasks))})
	table.Render()
}

func applyTask(cmd *cobra.Command, args []string) {
	if taskFile == "" {
		panic("file is empty")
	}
	// load json file to struct
	// var file = "task.json"
	data, err := os.ReadFile(taskFile)
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
		if dryRun {
			fmt.Println(tea.Prettify(taskJson) + "\n#######dry run, not apply task.#######")
			continue
		}
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
	log.Debugf(taskID, operator, syncMode)
	record, err := requestC.TaskExec(taskID, operator, syncMode)
	if err != nil {
		panic(err)
	}
	fmt.Println(tea.Prettify(record))
}
