package service

import (
	"cbs/pkg/model"
	"strings"

	"github.com/patsnapops/noop/log"
)

type WorkerService struct {
	BucketIo model.BucketIo
}

func NewWorkerService(bucketIo model.BucketIo) model.WorkerContract {
	return &WorkerService{
		BucketIo: bucketIo,
	}
}

func (w *WorkerService) SyncOnce(task model.Task, record model.Record, isServerSide bool) {
	objectsChan := make(chan model.ChanObject, 1000)
	sourceBucket, sourcePrefix := model.ParseBucketAndPrefix(task.SourceUrl)
	targetBucket, targetPrefix := model.ParseBucketAndPrefix(task.TargetUrl)
	go w.BucketIo.ListObjectsWithChan(task.SourceProfile, sourceBucket, sourcePrefix, model.Input{
		Recursive:  true,
		Include:    strings.Split(task.Include, ","),
		Exclude:    strings.Split(task.Exclude, ","),
		TimeBefore: model.StringToTime(task.TimeBefore),
		TimeAfter:  model.StringToTime(task.TimeAfter),
		Limit:      0,
	}, objectsChan)
	for object := range objectsChan {
		if *task.IsOverwrite {
			// overwrite
			if isServerSide {
				err := w.BucketIo.CopyObjectV1(task.SourceProfile, sourceBucket, *object.Obj, targetBucket, targetPrefix+object.Obj.Key)
				if err != nil {
					log.Errorf("copy object error: %v", err)
				}
				log.Infof("copy object %s/%s success", targetBucket, object.Obj.Key)
			} else {
				// client side
				data, err := w.BucketIo.GetObject(task.SourceProfile, sourceBucket, object.Obj.Key)
				if err != nil {
					log.Errorf("download object error: %v", err)
				}
				err = w.BucketIo.UploadObject(task.TargetProfile, targetBucket, targetPrefix+object.Obj.Key, data)
				if err != nil {
					log.Errorf("upload object error: %v", err)
				}
				log.Infof("upload object %s/%s success", targetBucket, object.Obj.Key)
			}
		} else {
			continue
		}
	}
}

func (w *WorkerService) KeepSync(task model.Task, record model.Record, isServerSide bool) {
}
