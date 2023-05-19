package model

import "time"

type CommonObj struct {
	Key          string
	Etag         string
	Size         int64
	LastModified time.Time
	StorageClass string
}

// 对象过滤
type ObjectFilter struct {
	TimeBefore   string
	TimeAfter    string
	Include      string
	Exclude      string
	StorageClass string
}

type CommonDir struct {
	Prefix string
}

type ObjectConstract interface {
	GetTaskObjectList(s3url string, obf *ObjectFilter) ([]*CommonObj, error)
	GetAllObjects(s3url string, obf *ObjectFilter) ([]*CommonObj, error)
	GetDirObjects(s3url string) ([]*CommonDir, error)
}

type ChanObjects struct {
	Key          *string
	Size         *int64
	ETag         *string
	LastModified *time.Time
	StorageClass *string
	// Owner        *Owner
	// ChecksumAlgorithm []*string
}

// type Owner struct {
// 	DisplayName *string
// 	ID          *string
// }
