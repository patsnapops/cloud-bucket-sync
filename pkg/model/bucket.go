package model

import (
	"strings"
	"time"

	"github.com/patsnapops/noop/log"
)

type Input struct {
	Recursive  bool
	Include    []string
	Exclude    []string
	TimeBefore *time.Time // 2023-03-01 21:26:30
	TimeAfter  *time.Time // 1992-03-01 21:26:30
	Limit      int64
}

func NewInput(recursive bool, include, exclude string, timeBefore, timeAfter string, limit int64) Input {

	input := Input{
		Recursive: recursive,
		Limit:     limit,
	}
	if include != "" {
		input.Include = strings.Split(include, ",")
	}
	if exclude != "" {
		input.Exclude = strings.Split(exclude, ",")
	}
	if timeBefore != "" {
		timeB, err := time.Parse("2006-01-02 15:04:05", timeBefore)
		if err != nil {
			log.Errorf("timeBefore format error: %v", err)
		} else {
			input.TimeBefore = &timeB
		}
	}
	if timeAfter != "" {
		timeA, err := time.Parse("2006-01-02 15:04:05", timeAfter)
		if err != nil {
			log.Errorf("timeAfter format error: %v", err)
		} else {
			input.TimeAfter = &timeA
		}
	}
	return input
}

type BucketContract interface {
	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)          //5.9s 4.5s 27s 15s
	ListObjectsWithChan(profile, bucketName, prefix string, input Input, objectsChan chan ChanObject) //使用chan的方式降低内存占用并降低大量数据的等待时间 16s 12s 6s
	RmObject(profile, bucketName, prefix string) error
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
