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
