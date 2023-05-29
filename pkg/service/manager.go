package service

import (
	"cbs/pkg/model"
	"time"

	"github.com/patsnapops/noop/log"
	"github.com/robfig/cron"
)

type ManagerService struct {
	Client model.ManagerIo
}

func NewManagerService(client model.ManagerIo) model.ManagerContract {
	return &ManagerService{Client: client}
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
		// 检查worker的LastHc是否超过10分钟没更新，如果超过则认为worker挂掉，需要修复running的任务
		if worker.UpdatedAt.Add(10 * time.Minute).Before(time.Now()) {
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
			log.Infof("restart record %s, task id: %s", record.Id, record.TaskId)
			// 修改record的状态为failed
			record.Status = model.TaskFailed
			err := s.Client.UpdateRecord(record)
			if err != nil {
				log.Errorf("update record %s failed, err: %v", record.Id, err)
				continue
			}
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
