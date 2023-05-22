package io

import (
	"cbs/pkg/model"
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
		TaskID: input.TaskID,
		ID:     input.RecordID,
		Status: input.Status,
	})
	resL := sql.Find(&taskRecords)
	return taskRecords, resL.Error
}

func (c *managerClient) UpdateRecord(record *model.Record) error {
	return c.db.Save(record).Error
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
	return worker.ID, c.db.Model(worker).Create(worker).Error
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
	log.Debugf(tea.Prettify(input))
	sql := c.db.Model(&tasks).Where(&model.Task{
		ID:     input.ID,
		Name:   input.Name,
		Worker: input.Worker,
	})
	resL := sql.Find(&tasks)
	return tasks, resL.Error
}

func (c *managerClient) CreateTask(task *model.Task) (string, error) {
	task.ID = uuid.New().String()
	return task.ID, c.db.Create(task).Error
}

func (c *managerClient) UpdateTask(task *model.Task) error {
	return c.db.Save(task).Error
}

func (c *managerClient) DeleteTask(taskID string) error {
	var task model.Task
	return c.db.Model(&task).Where("id = ?", taskID).Update("is_deleted", true).Error
}

// ExecuteTask 创建一个新的record
func (c *managerClient) ExecuteTask(taskID, operator string, mode model.Mode) (string, error) {
	recordTask := model.Record{
		ID:          uuid.New().String(),
		TaskID:      taskID,
		RunningMode: mode,
		Operator:    operator,
	}
	resL := c.db.Model(&recordTask).Create(&recordTask)
	return recordTask.ID, resL.Error
}
