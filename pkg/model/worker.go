package model

type FailedKey struct {
	SourceBucket string
	SourceKey    string
	Error        string
}

type WorkerContract interface {
	SyncOnce(task Task, record Record, isServerSide bool)
	KeepSync(task Task, record Record, isServerSide bool)
}
