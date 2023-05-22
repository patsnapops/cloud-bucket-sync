package model

type ManagerContract interface {
	CheckWorker() // 检查worker是否正常,并修复running的任务重新执行如果worker挂掉

}

type ManagerIo interface {
	ListRecords() ([]*Record, error)
	// 获取任务的执行记录列表
	QueryRecord(input RecordInput) ([]*Record, error)
	UpdateRecord(record *Record) error
	// DeleteRecord(recordID string) error // 不需要删除

	QueryWorker(input WorkerInput) ([]*Worker, error)
	CreateWorker(cloud, region string) (string, error)
	UpdateWorker(string) error
	DeleteWorker(workerID string) error

	ListTasks() ([]*Task, error)
	QueryTask(input TaskInput) ([]*Task, error)
	CreateTask(task *Task) (taskID string, err error)
	UpdateTask(task *Task) error
	DeleteTask(taskID string) error
	ExecuteTask(id, operator string, new_mode Mode) (recordID string, err error)
}
