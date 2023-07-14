package service

import (
	"cbs/pkg/model"
	"fmt"
	"time"

	"github.com/patsnapops/noop/log"
	"github.com/robfig/cron"
)

type ManagerService struct {
	Client              model.ManagerIo
	Dt                  model.DingtalkIo
	withDingtalkApprove bool
}

func NewManagerService(client model.ManagerIo, dtc model.DingtalkIo, withDingtalkApprove bool) model.ManagerContract {
	return &ManagerService{
		Client:              client,
		Dt:                  dtc,
		withDingtalkApprove: withDingtalkApprove,
	}
}

func (s *ManagerService) CheckWorker() {
	checkMap := make(map[string]string) // 用于记录已经修复过的任务,保证不会重复修复
	workers, err := s.Client.QueryWorker(model.WorkerInput{})
	if err != nil {
		log.Panicf("list workers failed, err: %v", err)
	}
	records, err := s.Client.ListRecords()
	if err != nil {
		log.Panicf("list records failed, err: %v", err)
	}
	for _, worker := range workers {
		if worker.IsDeleted {
			continue
		}
		// 检查worker的LastHc是否超过5分钟没更新，如果超过则认为worker挂掉，需要修复running的任务
		if worker.UpdatedAt.Add(5 * time.Minute).Before(time.Now()) {
			err = s.Client.DeleteWorker(worker.ID)
			if err != nil {
				log.Errorf("delete worker %s failed, err: %v", worker.ID, err)
				continue
			}
			log.Infof("delete worker %s success", worker.ID)
			s.restartRecord(checkMap, records, worker.ID)
		}
	}
}

func (s *ManagerService) restartRecord(checkMap map[string]string, records []*model.Record, workerID string) {
	for _, record := range records {
		if record.Status == model.TaskRunning && record.WorkerId == workerID {
			// 支持task计数，最多保持1个任务的修正，避免重复修复
			if _, ok := checkMap[record.TaskId]; ok {
				if checkMap[record.TaskId] == workerID {
					continue
				}
			}
			// 修改record的状态为failed
			record.Status = model.TaskFailed
			err := s.Client.UpdateRecord(record)
			if err != nil {
				log.Errorf("update record %s failed, err: %v", record.Id, err)
				continue
			}
			log.Infof("update record %s to failed.", record.Id)
			// 修复running的任务
			recordID, err := s.Client.ExecuteTask(record.TaskId, "system", record.RunningMode)
			if err != nil {
				log.Errorf("restart record %s failed, err: %v", record.Id, err)
				continue
			}
			log.Infof("restart record %s success, new record id: %s", record.Id, recordID)
			// 记录已经修复过的任务
			checkMap[record.TaskId] = workerID
		}
	}
}

func (s *ManagerService) CheckTaskCorn() {
	// 扫描task任务设置corn的
	tasks, err := s.Client.QueryTask(model.TaskInput{})
	if err != nil {
		log.Errorf("get task list error: %v", err)
		return
	}
	for _, task := range tasks {
		if task.SyncMode != string(model.ModeSyncOnce) {
			continue
		}
		if task.Corn == "" {
			continue
		}
		// 判断是否满足corn条件
		if !s.cornMatch(*task) {
			continue
		}
		// 判断是否需要执行
		if !s.needExecute(*task) {
			continue
		}
		// 执行
		// log.Infof("start execute task %s", task.ID)
		// 定时任务只会执行一次同步
		_, err := s.Client.ExecuteTask(task.Id, task.Submitter, string(model.ModeSyncOnce))
		if err != nil {
			log.Errorf("execute task %s error: %v", task.Id, err)
		}
	}
}

// UpdateTaskStatus
func (s *ManagerService) UpdateRecordStatus(recordID string, status model.Status) error {
	err := s.Client.UpdateRecordStatus(recordID, status)
	if err == nil {
		// 发起通知
		log.Debugf("update record %s status to %s failed, err: %v", recordID, status, err)
		if status == model.TaskSuccess || status == model.TaskFailed || status == model.TaskNotAllSuccess {
			record, _ := s.Client.GetRecord(recordID)
			taskInfo, _ := s.Client.GetTaskById(record.TaskId)
			// 发送钉钉消息
			msg := fmt.Sprintf("任务执行结束：%s\n\n任务ID：%s\n名称：%s\n提交者：%s\n耗时：%d 秒\n源：%s\n目标：%s\n消耗流量：%s\n文件数量：%d\n",
				record.Status, taskInfo.Id, taskInfo.Name, taskInfo.Submitter, record.CostTime, taskInfo.SourceUrl, taskInfo.TargetUrl,
				model.FormatSize(record.TotalSize), record.TotalFiles)
			log.Debugf("send dingtalk message: %s", msg)
			err := s.Dt.RobotSendText(msg)
			if err != nil {
				log.Errorf("send dingtalk message error: %v", err)
			}
		}
	}
	return err
}

// needExecute
func (s *ManagerService) needExecute(task model.Task) bool {
	// 判断是否需要执行,只有当task没有pending的record时才执行
	records, err := s.Client.QueryRecord(model.RecordInput{})
	if err != nil {
		log.Errorf("get task record list error: %v", err)
		return false
	}
	for _, record := range records {
		if record.TaskId == task.Id {
			if record.Status == model.TaskPending {
				log.Errorf("task %s is pending,only one task can be run.", task.Id)
				return false
			}
		}
	}
	return true
}

// cornMatch 判断到分钟
func (s *ManagerService) cornMatch(task model.Task) bool {
	// 判断是否满足corn条件
	sched, err := cron.ParseStandard(task.Corn)
	if err != nil {
		log.Errorf("parse corn for task %s:%s error: %v", task.Id, task.Corn, err)
	}

	// Get the current time
	now := time.Now()

	// Determine the next scheduled time
	nextTime := sched.Next(now.Add(-60 * time.Second))
	log.Debugf("now: %s, nextTime: %s", now, nextTime)
	// Compare the next scheduled time with the current time

	return (nextTime.Minute() == now.Minute()) && (nextTime.Hour() == now.Hour()) && (nextTime.Day() == now.Day()) && (nextTime.Month() == now.Month()) && (nextTime.Year() == now.Year())
}

func (s *ManagerService) CreateTask(task *model.Task) (taskID string, err error) {
	taskID, err = s.Client.CreateTask(task)
	if err != nil {
		return taskID, err
	}
	if task.Corn == "" {
		// 不是定时任务的，立刻创建一个record
		recordID, err := s.Client.ExecuteTask(task.Id, task.Submitter, task.SyncMode)
		if err != nil {
			return task.Id, fmt.Errorf("task create success, but execute task error: %v", err)
		}
		log.Infof("execute task success, taskID: %s, recordID: %s", taskID, recordID)
	}
	if s.withDingtalkApprove {
		// 创建钉钉审批流程
		processID, err := s.Dt.CreateDingTalkProcess(task)
		if err != nil {
			return task.Id, fmt.Errorf("task create success, but create dingtalk process error: %v", err)
		}
		// 更新task的processID
		err = s.Client.UpdateTask(&model.Task{
			Id:                 taskID,
			DingtalkInstanceId: processID,
		})
		if err != nil {
			return task.Id, fmt.Errorf("task create success, but update task processID %s error: %v", processID, err)
		}
		log.Infof("create dingtalk process success, taskID: %s, processID: %s", taskID, processID)
	}
	return taskID, nil
}

func (s *ManagerService) QueryRecord(input model.RecordInput) ([]*model.Record, error) {
	records, err := s.Client.QueryRecord(input)
	if err != nil {
		return records, err
	}
	// 支持审批特性，只返回审批通过的记录
	if s.withDingtalkApprove {
		recordsTmp := make([]*model.Record, 0)
		for _, record := range records {
			task, err := s.Client.GetTaskById(record.TaskId)
			if err != nil {
				return records, err
			}
			if task.ApproveResult == "agree" {
				recordsTmp = append(recordsTmp, record)
			}
		}
		return recordsTmp, nil
	}
	return records, nil
}
