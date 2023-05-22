package service

import (
	"cbs/pkg/model"
	"time"

	"github.com/patsnapops/noop/log"
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
		log.Debugf("worker %s last hc: %s", worker.ID, worker.Hc.Format("2006-01-02 15:04:05"))
		if worker.Hc.Add(10*60).Unix() < time.Now().Unix() {
			log.Infof("worker %s is dead 10m, last hc: %s", worker.ID, worker.Hc.Format("2006-01-02 15:04:05"))
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
		if record.Status == model.TaskRunning && record.WorkerID == workerID {
			// 支持task计数，最多保持1个任务的修正
			if _, ok := checkMap[record.TaskID]; ok {
				if checkMap[record.TaskID] == workerID {
					continue
				}
			}
			log.Infof("restart record %s, task id: %s", record.ID, record.TaskID)
			// 修复running的任务
			recordID, err := s.Client.ExecuteTask(record.TaskID, "system", record.RunningMode)
			if err != nil {
				log.Errorf("restart record %s failed, err: %v", record.ID, err)
				continue
			}
			log.Infof("restart record %s success, new record id: %s", record.ID, recordID)
			// 记录已经修复过的任务
			checkMap[record.TaskID] = workerID
		}
	}
}
