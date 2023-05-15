package model

import "time"

type Input struct {
	Recursive  bool
	Include    []string
	Exclude    []string
	TimeBefore string // 2023-03-01 21:26:30
	TimeAfter  string // 1992-03-01 21:26:30
	Limit      int64
}

func NewInput(recursive bool, include, exclude []string, timeBefore, timeAfter string, limit int64) Input {
	return Input{
		Recursive:  recursive,
		Include:    include,
		Exclude:    exclude,
		TimeBefore: timeBefore,
		TimeAfter:  timeAfter,
		Limit:      limit,
	}
}

type BucketContract interface {
	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)
	RmObject(profile, bucketName, prefix string, input Input) error
}

type BucketIo interface {
	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)
	RmObject(profile, bucketName, object string) error
}

type Object struct {
	Key          string
	Size         int64
	ETag         string
	StorageClass string
	LastModified time.Time
}
