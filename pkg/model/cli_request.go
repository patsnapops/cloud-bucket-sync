package model

type RequestContract interface {
	TaskGet(taskID string) (*TaskResponse, error)
	TaskList() ([]TaskResponse, error)
	TaskApply(args Task) (string, error)
	TaskExec(taskID string) (string, error)

	RecordUpdateStatus(string, Status) error
	RecordUpdate(*Record) error
	RecordQuery(RecordInput) ([]Record, error)
}
