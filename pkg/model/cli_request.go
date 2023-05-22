package model

type RequestContract interface {
	TaskGet(taskID string) ([]*Task, error)
	TaskList() ([]Task, error)
	TaskApply(args Task) (string, error)
	TaskExec(taskID string) (string, error)

	RecordUpdateStatus(string, Status) error
	RecordUpdate(*Record) error
	RecordQuery(RecordInput) ([]Record, error)

	WorkerQuery(WorkerInput) ([]Worker, error)
	WorkerHcUpdate(workerID string) error
}
