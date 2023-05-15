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
	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)          //5.9s 4.5s 27s 15s
	ListObjectsWithChan(profile, bucketName, prefix string, input Input, objectsChan chan ChanObject) //使用chan的方式降低内存占用并降低大量数据的等待时间 16s 12s 6s
	RmObject(profile, bucketName, prefix string, input Input) error
}

type BucketIo interface {
	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)
	ListObjectsWithChan(profile, bucketName, prefix string, input Input, objectsChan chan ChanObject) //使用chan的方式降低内存占用并降低大量数据的等待时间
	RmObject(profile, bucketName, object string) error
}

type Object struct {
	Key          string
	Size         int64
	ETag         string
	StorageClass string
	LastModified time.Time
}

type ChanObject struct {
	Error *error
	Obj   *Object
	Dir   *string
	Count int64
}
