package model

import "cbs/config"

type ManagerContract interface {
	CheckWorker()   // 检查worker是否正常,并修复running的任务重新执行如果worker挂掉
	CheckTaskCorn() // 检查task的cron表达式，符合条件的task会执行生成pending状态record去跑

	// 任务相关
	UpdateRecordStatus(recordID string, status Status) error
}

type ManagerIo interface {
	ListRecords() ([]*Record, error)
	// 获取任务的执行记录列表
	QueryRecord(input RecordInput) ([]*Record, error)
	GetRecord(recordID string) (*Record, error)
	UpdateRecord(record *Record) error
	UpdateRecordStatus(recordID string, status Status) error
	// DeleteRecord(recordID string) error // 不需要删除

	QueryWorker(input WorkerInput) ([]*Worker, error)
	CreateWorker(cloud, region string) (string, error)
	UpdateWorker(string) error
	DeleteWorker(workerID string) error

	QueryTask(input TaskInput) ([]*Task, error)
	GetTaskById(taskID string) (*Task, error)
	CreateTask(task *Task, profiles config.ManagerConfig) (taskID string, err error)
	UpdateTask(task *Task) error
	DeleteTask(taskID string) error
	ExecuteTask(id, operator, new_mode string) (recordID string, err error) //如果输入new_mode要指定
}
