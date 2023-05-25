package io

import (
	"bytes"
	"cbs/config"
	"cbs/pkg/model"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/patsnapops/noop/log"
)

type bucketClient struct {
	configProfiles []config.Profile
	sessions       map[string]*session.Session
}

func NewBucketClient(config []config.Profile) model.BucketIo {
	sessions, err := newSession(config)
	if err != nil {
		log.Errorf("new session error %s", err)
	}
	return &bucketClient{
		configProfiles: config,
		sessions:       sessions,
	}
}

func newSession(configProfiles []config.Profile) (map[string]*session.Session, error) {
	sessions := make(map[string]*session.Session)
	for _, param := range configProfiles {
		s3_conf := &aws.Config{
			Credentials: credentials.NewStaticCredentials(param.AK, param.SK, ""),
			Region:      aws.String(param.Region),
			// DisableSSL:       aws.Bool(true),  //trail 需要关闭
			S3ForcePathStyle: aws.Bool(false), //virtual-host style方式，不要修改
			// MaxRetries:       aws.Int(0),
			Endpoint: aws.String(param.Endpoint),
		}
		// log.Debugf(s3_conf.Endpoint)
		sess, err := session.NewSession(s3_conf)
		if err != nil {
			return sessions, err
		}
		sessions[param.Name] = sess
	}
	return sessions, nil

}

func (c *bucketClient) ListObjects(profile, bucketName, prefix string, input model.Input) ([]string, []model.Object, error) {
	log.Debugf("list objects %s/%s", bucketName, prefix)
	objects := make([]model.Object, 0)
	dirs := make([]string, 0)
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		s3Input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		}
		log.Debugf(tea.Prettify(input))
		if input.Recursive {
			s3Input.Delimiter = aws.String("")
		} else {
			s3Input.Delimiter = aws.String("/")
		}
		for {
			resp, err := svc.ListObjectsV2(s3Input)
			if err != nil {
				return dirs, objects, err
			}
			for _, content := range resp.Contents {
				obj := model.Object{
					Key:          *content.Key,
					Size:         *content.Size,
					ETag:         *content.ETag,
					StorageClass: *content.StorageClass,
					LastModified: *content.LastModified,
				}
				if model.ListObjectsWithFilter(obj, input) {
					objects = append(objects, obj)
				}
			}
			for _, commonPrefix := range resp.CommonPrefixes {
				dirs = append(dirs, *commonPrefix.Prefix)
			}
			if len(objects) > int(input.Limit) && input.Limit != 0 {
				return dirs, objects[:input.Limit], nil
			}
			if !*resp.IsTruncated {
				return dirs, objects, nil
			}
			s3Input.ContinuationToken = resp.NextContinuationToken
		}
	}
	return dirs, objects, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)

}

func (c *bucketClient) ListObjectsWithChan(profile, bucketName, prefix string, input model.Input, objectsChan chan model.ChanObject) {
	log.Debugf("list objects %s/%s", bucketName, prefix)
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		s3Input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		}
		log.Debugf(tea.Prettify(input))
		if input.Recursive {
			s3Input.Delimiter = aws.String("")
		} else {
			s3Input.Delimiter = aws.String("/")
		}
		var index int64
		for {
			resp, err := svc.ListObjectsV2(s3Input)
			if err != nil {
				objectsChan <- model.ChanObject{
					Error: &err,
				}
				close(objectsChan)
			}
			for _, content := range resp.Contents {
				index++
				obj := model.Object{
					Key:          *content.Key,
					Size:         *content.Size,
					ETag:         *content.ETag,
					StorageClass: *content.StorageClass,
					LastModified: *content.LastModified,
				}
				if model.ListObjectsWithFilter(obj, input) {
					objectsChan <- model.ChanObject{
						Obj: &obj,
					}
				}

			}
			log.Debugf("index %d", index)

			for _, commonPrefix := range resp.CommonPrefixes {
				index++
				objectsChan <- model.ChanObject{
					Dir: commonPrefix.Prefix,
				}
			}
			if int(index) > int(input.Limit) && input.Limit != 0 {
				close(objectsChan)
				return
			}
			if !*resp.IsTruncated {
				close(objectsChan)
				return
			}
			s3Input.ContinuationToken = resp.NextContinuationToken
		}
	}
	close(objectsChan)
}

func (c *bucketClient) RmObject(profile, bucketName, prefix string) error {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		s3Input := &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(prefix),
		}
		resp, err := svc.DeleteObject(s3Input)
		// log.Infof(tea.Prettify(resp))
		// 对象不存在的时候也不会报错
		if err != nil {
			log.Errorf(resp.String())
		}
		return err
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) CreateMutiUpload(profile, bucketName, object string) (string, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		// TODO: ListMultipartUploads AbortMultipartUpload来优化重新上传的情况
		input := &s3.CreateMultipartUploadInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
		}
		result, err := svc.CreateMultipartUpload(input)
		if err != nil {
			return "", err
		}
		upload_id := *result.UploadId
		return upload_id, nil
	}
	return "", fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) ComplateMutiPartUpload(profile, bucketName, object, uploadId string, completed_parts []*s3.CompletedPart) error {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.CompleteMultipartUploadInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
			MultipartUpload: &s3.CompletedMultipartUpload{
				Parts: completed_parts,
			},
			UploadId: aws.String(uploadId),
		}

		_, err := svc.CompleteMultipartUpload(input)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// fmt.Sprintf("bytes=%d-%d", src.Start, src.End)
func (c *bucketClient) UploadPart(profile, bucketName, object, copySource, copySourceRange, uploadId string, partNumber int64) (*s3.CompletedPart, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.UploadPartCopyInput{
			Bucket:          aws.String(bucketName),
			CopySource:      aws.String(copySource),
			CopySourceRange: aws.String(copySourceRange),
			Key:             aws.String(object),
			PartNumber:      aws.Int64(partNumber),
			UploadId:        aws.String(uploadId),
		}
		resp, err := svc.UploadPartCopy(input)
		if err != nil {
			return nil, err
		}
		return &s3.CompletedPart{
			ETag:       resp.CopyPartResult.ETag,
			PartNumber: aws.Int64(partNumber),
		}, nil
	}
	return nil, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) copyObject(profile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) error {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.CopyObjectInput{
			Bucket:     aws.String(targetBucket),
			CopySource: aws.String(fmt.Sprintf("/%s/%s", sourceBucket, sourceObj.Key)),
			Key:        aws.String(targetKey),
		}
		_, err := svc.CopyObject(input)
		if err != nil {
			return err
		}
		// 解决destkey /开头问题
		log.Infof(`copy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, strings.TrimPrefix(targetKey, "/"), model.FormatSize(sourceObj.Size))
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 兼容了大文件自动使用muti
func (c *bucketClient) CopyObjectV1(profile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) error {
	var partIndex int64 = 0
	// 低于5G的数据直接使用copy
	if sourceObj.Size <= model.MaxPartSize {
		// TODO:需要支持文件到目录的格式
		err := c.copyObject(profile, sourceBucket, sourceObj, targetBucket, targetKey)
		return err
	} else {
		// 文件过大5TB？
		totalParts := model.PartsRequired(sourceObj.Size)
		// Do we need more parts than we are allowed?
		if totalParts > model.MaxPartsCount {
			return fmt.Errorf("Your proposed compose object requires more than %d parts", model.MaxPartsCount)
		}

		upload_id, err := c.CreateMutiUpload(profile, targetBucket, targetKey)
		if err != nil {
			return err
		}

		startIdx, endIdx := model.CalculateEvenSplits(sourceObj.Size)
		// log.Debugf("calculateEvenSplits", startIdx, endIdx)

		var completed_parts []*s3.CompletedPart
		for j, start := range startIdx {
			partIndex++
			Start := start
			End := endIdx[j]
			copySource := fmt.Sprintf("/%s/%s", sourceBucket, sourceObj.Key)
			copySourceRange := fmt.Sprintf("bytes=%d-%d", Start, End)
			part, err := c.UploadPart(profile, targetBucket, targetKey, copySource, copySourceRange, upload_id, partIndex)
			if err != nil {
				log.Fatal(err.Error())
				break
			}
			completed_parts = append(completed_parts, part)
			log.Infof(`UploadPartCopy s3://%s/%s %d/%d`, targetBucket, targetKey, partIndex, totalParts)
		}
		err = c.ComplateMutiPartUpload(profile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
			return err
		} else {
			log.Infof(`muticopy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, targetKey, model.FormatSize(sourceObj.Size))
			return nil
		}
	}
}

func (c *bucketClient) UploadObject(profile, bucketName, object string, data []byte) error {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.PutObjectInput{
			Body:   bytes.NewReader(data),
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
		}
		_, err := svc.PutObject(input)
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) GetObject(profile, bucketName, object string) ([]byte, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
		}
		resp, err := svc.GetObject(input)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) HeadObject(profile, bucketName, object string) (model.Object, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
		}
		resp, err := svc.HeadObject(input)
		if err != nil {
			return model.Object{}, err
		}
		return model.Object{
			Key:          object,
			Size:         *resp.ContentLength,
			LastModified: *resp.LastModified,
			ETag:         *resp.ETag,
		}, nil
	}
	return model.Object{}, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) MutiDownloadObject(objectSize int64, sourceKey, sourceEtag string, ch chan<- *model.ChData) {
}
