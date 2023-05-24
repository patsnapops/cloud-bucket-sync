package io

import (
	"cbs/config"
	"cbs/pkg/model"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/patsnapops/noop/log"
	"gorm.io/gorm"
)

type managerClient struct {
	db *gorm.DB
}

func NewManagerClient(db *gorm.DB) model.ManagerIo {
	return &managerClient{db: db}
}

func (c *managerClient) ListRecords() ([]*model.Record, error) {
	var records []*model.Record
	err := c.db.Find(&records).Error
	return records, err
}

func (c *managerClient) QueryRecord(input model.RecordInput) ([]*model.Record, error) {
	var taskRecords []*model.Record
	log.Debugf(tea.Prettify(input))
	sql := c.db.Model(&taskRecords).Where(&model.Record{
		TaskId: input.TaskID,
		Id:     input.RecordID,
		Status: input.Status,
	})
	resL := sql.Find(&taskRecords)
	return taskRecords, resL.Error
}

func (c *managerClient) UpdateRecord(record *model.Record) error {
	recordId := record.Id
	log.Debugf("update record: %s", tea.Prettify(record))
	return c.db.Model(&model.Record{}).Where("id = ?", recordId).Updates(record).Error
}

func (c *managerClient) UpdateRecordStatus(recordID string, status model.Status) error {
	return c.db.Model(&model.Record{}).Where("id = ?", recordID).Update("status", status).Error
}

func (c *managerClient) ListWorkers() ([]*model.Worker, error) {
	var workers []*model.Worker
	err := c.db.Model(model.Worker{}).Find(&workers).Error
	return workers, err
}

func (c *managerClient) QueryWorker(input model.WorkerInput) ([]*model.Worker, error) {
	var workers []*model.Worker
	log.Debugf(tea.Prettify(input))
	sql := c.db.Model(&workers).Where(&model.Worker{
		ID:     input.WorkerID,
		Cloud:  input.Cloud,
		Region: input.Region,
	})
	resL := sql.Find(&workers)
	return workers, resL.Error
}

func (c *managerClient) CreateWorker(cloud, region string) (string, error) {
	worker := model.Worker{
		ID:     uuid.New().String(),
		Cloud:  cloud,
		Region: region,
	}
	log.Debugf("create worker: %s", tea.Prettify(worker))
	res := c.db.Create(&worker)
	if res.Error != nil {
		return "", res.Error
	}
	return worker.ID, nil
}

// 只更新worker的hc时间
func (c *managerClient) UpdateWorker(workerID string) error {
	var worker model.Worker
	return c.db.Model(&worker).Where("id = ?", workerID).Update("hc", time.Now()).Error
}

func (c *managerClient) DeleteWorker(workerID string) error {
	var worker model.Worker
	return c.db.Model(&worker).Where("id = ?", workerID).Update("is_deleted", true).Error
}

func (c *managerClient) ListTasks() ([]*model.Task, error) {
	log.Debugf("list tasks")
	var tasks []*model.Task
	err := c.db.Find(&tasks).Error
	return tasks, err
}

func (c *managerClient) QueryTask(input model.TaskInput) ([]*model.Task, error) {
	var tasks []*model.Task
	// log.Debugf(tea.Prettify(input))
	sql := c.db.Model(&tasks).Where(&model.Task{
		ID:        input.ID,
		Name:      input.Name,
		WorkerTag: input.WorkerTag,
	})
	resL := sql.Find(&tasks)
	return tasks, resL.Error
}

func (c *managerClient) CreateTask(task *model.Task, managerConfig config.ManagerConfig) (string, error) {
	task.ID = uuid.New().String()
	task.WorkerTag = fmtWorkerTag(task, managerConfig)
	return task.ID, c.db.Create(task).Error
}

// fmt task worker tags，依据
func fmtWorkerTag(task *model.Task, managerConfig config.ManagerConfig) string {
	var sourceProfile config.Profile
	var targetProfile config.Profile
	for _, profile := range managerConfig.Profiles {
		if profile.Name == task.TargetProfile {
			targetProfile = profile
			break
		}
	}
	for _, profile := range managerConfig.Profiles {
		if profile.Name == task.SourceProfile {
			sourceProfile = profile
			break
		}
	}
	if sourceProfile.Name == "" || targetProfile.Name == "" {
		log.Errorf("source profile or target profile not found, source: %s, target: %s", task.SourceProfile, task.TargetProfile)
		return ""
	}
	sourceRegionPrefix := strings.Split(sourceProfile.Region, "-")[0]
	targetRegionPrefix := strings.Split(targetProfile.Region, "-")[0]
	// same cloud and same region
	if sourceProfile.Cloud == targetProfile.Cloud && sourceRegionPrefix == targetRegionPrefix {
		return fmt.Sprintf("%s-%s", sourceProfile.Cloud, sourceRegionPrefix)
	}
	// same cloud and different region
	if sourceProfile.Cloud == targetProfile.Cloud && sourceRegionPrefix != targetRegionPrefix {
		return fmt.Sprintf("%s-%s:%s", sourceProfile.Cloud, sourceRegionPrefix, targetRegionPrefix)
	}
	// different cloud and same region
	if sourceProfile.Cloud != targetProfile.Cloud && sourceRegionPrefix == targetRegionPrefix {
		return fmt.Sprintf("%s:%s-%s", sourceProfile.Cloud, targetProfile.Cloud, sourceRegionPrefix)
	}
	return fmt.Sprintf("%s:%s-%s:%s", sourceProfile.Cloud, targetProfile.Cloud, sourceRegionPrefix, targetRegionPrefix)
}

func (c *managerClient) UpdateTask(task *model.Task) error {
	return c.db.Save(task).Error
}

func (c *managerClient) DeleteTask(taskID string) error {
	var task model.Task
	return c.db.Model(&task).Where("id = ?", taskID).Update("is_deleted", true).Error
}

// ExecuteTask 创建一个新的record
// 检查同一个taskid的只能存在一个running或者一个pending的record
func (c *managerClient) ExecuteTask(taskID, operator string, runningMode model.Mode) (string, error) {
	records := []model.Record{}
	c.db.Model(&model.Record{}).Where("task_id = ? and status in (?)", taskID, []model.Status{model.TaskRunning, model.TaskPending}).Find(&records)
	if len(records) > 0 {
		return "", model.ErrTaskRunning
	}
	recordTask := model.Record{
		Id:          uuid.New().String(),
		TaskId:      taskID,
		RunningMode: runningMode,
		Operator:    operator,
	}
	resL := c.db.Model(&recordTask).Create(&recordTask)
	return recordTask.Id, resL.Error
}
