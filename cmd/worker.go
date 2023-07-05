package cmd

import (
	"cbs/pkg/model"
	"cbs/pkg/service"
	"os"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/olekukonko/tablewriter"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
)

var (
	region  string
	cloud   string
	workerC model.WorkerContract

	workerThreadNum int64 // 任务并发数,同时进行同步的对象数量
)

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.AddCommand(showWorkerCmd)
	workerCmd.AddCommand(runWorker)
	runWorker.Flags().StringVarP(&region, "region", "", "cn", "eg: cn, us, eu, ap")
	runWorker.Flags().StringVarP(&cloud, "cloud", "", "aws", "eg: aws, azure, aliyun, huawei, tencent, google")
	runWorker.Flags().Int64VarP(&workerThreadNum, "thread", "", 4, "worker num")
	showWorkerCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format, support table/json")

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

var runWorker = &cobra.Command{
	Use:     "start",
	Short:   "start a worker",
	Aliases: []string{"run"},
	Long:    "\nyou know for start a worker!",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		requestC = service.NewRequestService(cliConfig.Manager)
		workerC = service.NewWorkerService(bucketIo, requestC, workerThreadNum)
		switch len(args) {
		case 0:
			runWorkerCmd(cmd, args)
		default:
			cmd.Help()
		}
	},
}

func runWorkerCmd(cmd *cobra.Command, args []string) {
	// 注册
	workerID, err := requestC.WorkerRegister(cloud, region)
	if err != nil {
		panic(err)
	}
	worker := model.Worker{
		ID:     workerID,
		Cloud:  cloud,
		Region: region,
	}
	log.Infof("worker start with id: %s", workerID)
	log.Infof("cloud,region: %s,%s", cloud, region)
	for {
		run(worker)
		time.Sleep(60 * time.Second)
	}
}

func run(worker model.Worker) {
	// 查询pending的任务record
	records, err := requestC.RecordQuery(model.RecordInput{Status: "pending"})
	if err != nil {
		log.Errorf("query pending record error: %s", err)
	}
	for _, record := range records {
		// 任务准备
		if record.TaskId == "" {
			continue
		}
		task, err := requestC.TaskGetByID(record.TaskId)
		if err != nil {
			log.Errorf("query task %s error: %s", record.TaskId, err)
			err := requestC.RecordUpdateStatus(record.Id, model.TaskFailed)
			if err != nil {
				log.Errorf("update record status error: %s", err)
			}
			continue
		}
		log.Debugf(tea.Prettify(task))
		// 判断任务和节点的亲和性
		if !checkTaskAndWorkerAffinity(task, worker) {
			continue
		}
		log.Debugf(tea.Prettify(record))
		// 实时同步任务不在同一个worker上重复执行。
		// 更新任务的workerID
		if checkIsRunBySameWorker(&record, worker) {
			continue
		}

		// 更新record的workerID
		record.WorkerId = worker.ID
		err = requestC.RecordUpdate(record)
		if err != nil {
			log.Errorf("update record workerID error: %s", err)
			continue
		}
		// 更新record
		err = requestC.RecordUpdateStatus(record.Id, model.TaskRunning)
		record.Status = model.TaskRunning
		if err != nil {
			log.Errorf("update record status error: %s", err)
			continue
		}
		log.Infof("start record %s", record.Id)
		// 执行任务
		switch record.RunningMode {
		case "syncOnce":
			log.Debugf("syncOnce")
			go workerC.SyncOnce(*task, record)
		case "keepSync":
			log.Debugf("keepSync")
			go workerC.KeepSync(task.Id, record.Id)
		default:
			log.Errorf("unknown running mode: %s", record.RunningMode)
		}
	}
	// update hc
	requestC.WorkerHcUpdate(worker.ID)
}

// 判断任务和节点的亲和性
func checkTaskAndWorkerAffinity(task *model.Task, worker model.Worker) bool {
	if task.WorkerTag == "" || !strings.Contains(task.WorkerTag, "-") {
		log.Errorf("task taskId:%s workerTag err. %s", task.Id, task.WorkerTag)
		return false
	}
	taskCloud := task.WorkerTag[0:strings.Index(task.WorkerTag, "-")]
	taskRegion := task.WorkerTag[strings.Index(task.WorkerTag, "-")+1:]
	if taskCloud != worker.Cloud || !strings.Contains(worker.Region, taskRegion) {
		log.Infof("ignore taskId:%s tag:%s", task.Id, task.WorkerTag)
		return false
	}
	return true
}

func checkIsRunBySameWorker(record *model.Record, worker model.Worker) bool {
	if record.WorkerId == worker.ID {
		log.Errorf("跳过一个worker同时执行任务的情况 %s", record.Id)
		return true
	}
	return false
}

var showWorkerCmd = &cobra.Command{
	Use:   "show",
	Short: "show workers",
	Long:  "\nyou know for show workers!",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		switch len(args) {
		case 0:
			workers, err := requestC.WorkerQuery(model.WorkerInput{})
			if err != nil {
				panic(err)
			}
			if outputFormat == "json" {
				for _, worker := range workers {
					cmd.Println(tea.Prettify(worker))
				}
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"ID", "Cloud", "Region", "Hc", "CreatedAt"})
				for _, worker := range workers {
					table.Append([]string{worker.ID, worker.Cloud, worker.Region, worker.Hc.Format("2006-01-02 15:04:05"), worker.CreatedAt.Format("2006-01-02 15:04:05")})
				}
				table.Render()
			}

		default:
			cmd.Help()
		}
	},
}
