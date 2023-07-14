package model

type ManagerContract interface {
	CheckWorker()   // 检查worker是否正常,并修复running的任务重新执行如果worker挂掉
	CheckTaskCorn() // 检查task的cron表达式，符合条件的task会执行生成pending状态record去跑

	// 任务相关 有逻辑处理的上升到 service 处理，否则在 IO 处理
	UpdateRecordStatus(recordID string, status Status) error
	CreateTask(task *Task) (taskID string, err error)
	QueryRecord(input RecordInput) ([]*Record, error)
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
	// 创建任务，同时判别如果是定时任务是否立刻执行当前任务；支持钉钉审批接入特性
	CreateTask(task *Task) (taskID string, err error)
	UpdateTask(task *Task) error
	DeleteTask(taskID string) error
	ExecuteTask(id, operator, new_mode string) (recordID string, err error) //如果输入new_mode要指定
}
