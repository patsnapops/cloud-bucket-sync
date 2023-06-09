package model

type RequestContract interface {
	TaskQuery(TaskInput) ([]*Task, error)
	TaskGetByID(taskID string) (*Task, error)
	TaskApply(args Task) (string, error)
	TaskExec(taskID, operator, syncMode string) (string, error)
	TaskCancel(recordID, operator string) error
	TaskDelete(taskID string) error

	RecordUpdateStatus(recordID string, status Status) error
	RecordUpdate(Record) error
	RecordQuery(RecordInput) ([]Record, error)
	RecordGetByID(recordID string) (*Record, error)

	WorkerRegister(cloud, region string) (string, error)
	WorkerQuery(WorkerInput) ([]Worker, error)
	WorkerHcUpdate(workerID string) error
}

type TaskWithRecords struct {
	Task
	Records []Record `json:"records"`
}

type TaskExecInput struct {
	TaskID   string `json:"task_id"`
	Operator string `json:"operator"`
	SyncMode string `json:"sync_mode"` // 执行模式，支持修改同步模式。keepSync（实时同步） syncOnce（一次同步）
}
