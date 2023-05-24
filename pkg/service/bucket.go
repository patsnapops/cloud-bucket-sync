package service

import (
	"cbs/pkg/model"
)

type BucketService struct {
	BucketClient model.BucketIo
}

func NewBucketService(cli model.BucketIo) model.BucketContract {
	return &BucketService{
		BucketClient: cli,
	}
}

func (s *BucketService) ListObjects(profile, bucketName, prefix string, input model.Input) ([]string, []model.Object, error) {
	dirs, objects, err := s.BucketClient.ListObjects(profile, bucketName, prefix, input)
	if err != nil {
		return nil, nil, err
	}
	return dirs, objects, nil
}

func (s *BucketService) ListObjectsWithChan(profile, bucketName, prefix string, input model.Input, objectsChan chan model.ChanObject) {
	s.BucketClient.ListObjectsWithChan(profile, bucketName, prefix, input, objectsChan)
}

func (s *BucketService) DownloadObject(profile, bucketName, object string) ([]byte, error) {
	return nil, nil
}

func (s *BucketService) UploadObject(profile, bucketName, object string, data []byte) error {
	return nil
}

func (s *BucketService) CopyObject(sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) error {
	return nil
}

func (s *BucketService) RmObject(profile, bucketName, prefix string) error {
	return s.BucketClient.RmObject(profile, bucketName, prefix)
}
