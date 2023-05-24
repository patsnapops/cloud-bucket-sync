package service

import (
	"cbs/pkg/model"
	"strings"

	"github.com/patsnapops/noop/log"
)

type WorkerService struct {
	BucketService model.BucketContract
}

func NewWorkerService(bucketSvc model.BucketContract) model.WorkerContract {
	return &WorkerService{
		BucketService: bucketSvc,
	}
}

func (w *WorkerService) SyncOnce(task model.Task, record model.Record, isServerSide bool) {
	objectsChan := make(chan model.ChanObject, 1000)
	sourceBucket, sourcePrefix := s3UrlToBucketAndPrefix(task.SourceUrl)
	targetBucket, targetPrefix := s3UrlToBucketAndPrefix(task.TargetUrl)
	go w.BucketService.ListObjectsWithChan(task.SourceProfile, sourceBucket, sourcePrefix, model.Input{
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
				err := w.BucketService.CopyObject(sourceBucket, *object.Obj, targetBucket, targetPrefix+object.Obj.Key)
				if err != nil {
					log.Errorf("copy object error: %v", err)
				}
				log.Infof("copy object %s/%s success", targetBucket, object.Obj.Key)
			} else {
				// client side
				data, err := w.BucketService.DownloadObject(task.SourceProfile, sourceBucket, object.Obj.Key)
				if err != nil {
					log.Errorf("download object error: %v", err)
				}
				err = w.BucketService.UploadObject(task.TargetProfile, targetBucket, targetPrefix+object.Obj.Key, data)
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

func s3UrlToBucketAndPrefix(url string) (string, string) {
	return "", ""
}
