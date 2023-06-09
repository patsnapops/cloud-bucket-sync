package cmd

import (
	"cbs/pkg/model"
	"cbs/pkg/service"
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

	// show task flag
	// 支持 table/json的查看方式
	outputFormat string
	// 默认只展示正在运行的任务
	// showAll bool
)

func init() {
	rootCmd.AddCommand(taskCmd)

	taskCmd.AddCommand(applyCmd)
	taskCmd.AddCommand(showCmd)
	taskCmd.AddCommand(execCmd)
	taskCmd.AddCommand(cancelCmd)
	taskCmd.AddCommand(deleteCmd)
	taskCmd.PersistentFlags().StringVarP(&taskFile, "file", "f", "", "task file path, default is ./task.json")

	showCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format, support table/json")
	// showCmd.Flags().BoolVarP(&showAll, "all", "a", false, "show all tasks")

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

// delete task
var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "delete task",
	Aliases: []string{"del"},
	Long:    "\nyou know for delete a task!",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)
		switch len(args) {
		case 1:
			taskID := args[0]
			err := requestC.TaskDelete(taskID)
			if err != nil {
				panic(err)
			}
			fmt.Printf("task %s delete success\n", taskID)
		default:
			cmd.Help()
		}
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "to create task",
	Long:  "\nyou know for submit a task!",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)
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
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)

		switch len(args) {
		case 0:
			input := model.TaskInput{}
			tasks, err := requestC.TaskQuery(input)
			if err != nil {
				panic(err)
			}
			if outputFormat == "json" {

				for _, task := range tasks {
					fmt.Println(tea.Prettify(task))
				}
			} else {
				showTask(tasks)
			}
		case 1:
			// show taskID
			taskID := args[0]
			task, err := requestC.TaskGetByID(taskID)
			if err != nil {
				panic(err)
			}
			// only show last 5 records
			records, err := requestC.RecordQuery(model.RecordInput{
				TaskID: task.Id,
			})
			if err != nil {
				records = []model.Record{}
			}
			if len(records) > 5 {
				fmt.Println("only show last 5 record by create_at...")
				records = records[len(records)-5:]
			}
			fmt.Println(tea.Prettify(model.TaskWithRecords{
				Task:    *task,
				Records: records,
			}))
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
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)
		switch len(args) {
		case 1:
			execTask(cmd, args)
		default:
			cmd.Help()
		}
	},
}

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "to cancel task",
	Long:  "\nyou know for cancel task!",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)
		switch len(args) {
		case 1:
			cancelTask(cmd, args)
		default:
			cmd.Help()
		}
	},
}

func showTask(tasks []*model.Task) {
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

// cancel task
func cancelTask(cmd *cobra.Command, args []string) {
	taskID := args[0]
	// query record by taskid
	records, err := requestC.RecordQuery(model.RecordInput{
		TaskID: taskID,
	})
	if err != nil {
		panic(err)
	}
	if len(records) == 0 {
		log.Infof("task %s not found", taskID)
		return
	}
	for _, record := range records {
		if record.Status == model.TaskRunning ||
			record.Status == model.TaskPending {
			err := requestC.TaskCancel(record.Id, "cli")
			if err != nil {
				panic(err)
			}
			log.Infof("canceled record %s success", record.Id)
		}
	}
	log.Infof("cancel task end...")
}
