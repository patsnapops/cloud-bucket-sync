package service

import (
	"cbs/pkg/model"

	"github.com/patsnapops/noop/log"
)

type BucketService struct {
	BucketClient model.BucketIo
}

func NewBucketService(cli model.BucketIo) model.BucketContract {
	return BucketService{
		BucketClient: cli,
	}
}

func (s BucketService) ListObjects(profile, bucketName, prefix string, input model.Input) ([]string, []model.Object, error) {
	dirs, objects, err := s.BucketClient.ListObjects(profile, bucketName, prefix, input)
	if err != nil {
		return nil, nil, err
	}
	return dirs, objects, nil
}

func (s BucketService) ListObjectsWithChan(profile, bucketName, prefix string, input model.Input, objectsChan chan model.ChanObject) {
	s.BucketClient.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
}

func (s BucketService) RmObject(profile, bucketName, prefix string, input model.Input) error {
	log.Debugf("rm object %s/%s", bucketName, prefix)
	_, objects, err := s.ListObjects(profile, bucketName, prefix, input)
	if err != nil {
		return err
	}
	log.Debugf("objects %v", objects)
	for _, object := range objects {
		log.Debugf("rm object %s/%s", bucketName, object.Key)
	}
	return nil
}
