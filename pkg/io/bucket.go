package io

import (
	"bytes"
	"cbs/config"
	"cbs/pkg/model"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

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
		// log.Debugf(tea.Prettify(param))
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
		// log.Debugf("new session %s ", sessions)
	}
	return sessions, nil

}

func (c *bucketClient) Presign(profile, bucketName, objectKey string, expires int64) (string, error) {
	log.Debugf("get object %s/%s", bucketName, objectKey)
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		url, err := req.Presign(time.Duration(expires) * time.Second)
		if err != nil {
			return "", err
		}
		return url, nil

	}
	return "", errors.New("profile not found")
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

func (c *bucketClient) ListObjectsWithChan(profile, bucketName, prefix string, input model.Input, objectsChan chan *model.ChanObject) {
	log.Debugf("list objects s3://%s/%s", bucketName, prefix)
	log.Debugf(tea.Prettify(input))
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		s3Input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			Prefix: aws.String(prefix),
		}
		if input.Recursive {
			s3Input.Delimiter = aws.String("")
		} else {
			s3Input.Delimiter = aws.String("/")
		}
		var index int64
		var match int64
		for {
			resp, err := svc.ListObjectsV2(s3Input)
			if err != nil {
				log.Errorf("list objects error %s", err)
				close(objectsChan)
				return
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
					match++
					log.Debugf("list objects %s", obj.Key)
					objectsChan <- &model.ChanObject{
						Obj: &obj,
					}
				}

			}

			for _, commonPrefix := range resp.CommonPrefixes {
				index++
				objectsChan <- &model.ChanObject{
					Dir: commonPrefix.Prefix,
				}
			}
			log.Debugf("total obj: %d/%d", match, index)

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

// abortmultipartupload
func (c *bucketClient) AbortMutiPartUpload(profile, bucketName, object, uploadId string) error {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucketName),
			Key:      aws.String(object),
			UploadId: aws.String(uploadId),
		}
		resp, err := svc.AbortMultipartUpload(input)
		if err != nil {
			return err
		}
		log.Debugf(tea.Prettify(resp))
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// server side &client all use this.
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
		// log.Debugf(tea.Prettify(input))
		resp, err := svc.CompleteMultipartUpload(input)
		if err != nil {
			// abort mutipart upload
			c.AbortMutiPartUpload(profile, bucketName, object, uploadId)
			return err
		}
		log.Debugf(tea.Prettify(resp))
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 同一个profile下的分片上传，走的后台
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
		log.Debugf("copySourceRange %s etag:%s", copySourceRange, *resp.CopyPartResult.ETag)
		return &s3.CompletedPart{
			ETag:       resp.CopyPartResult.ETag,
			PartNumber: aws.Int64(partNumber),
		}, nil
	}
	return nil, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

func (c *bucketClient) UploadPartWithData(profile, bucketName, object, uploadId string, partNumber int64, data []byte) (*s3.CompletedPart, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.UploadPartInput{
			Bucket:     aws.String(bucketName),
			Key:        aws.String(object),
			PartNumber: aws.Int64(partNumber),
			UploadId:   aws.String(uploadId),
			Body:       bytes.NewReader(data),
		}
		resp, err := svc.UploadPart(input)
		if err != nil {
			return nil, err
		}
		return &s3.CompletedPart{
			ETag:       resp.ETag,
			PartNumber: aws.Int64(partNumber),
		}, nil
	}
	return nil, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 处理同地域下的桶拷贝，支持最大单文件5G，超过要使用mutiPartUpload
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
		log.Infof(`copy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, strings.TrimPrefix(targetKey, "/"), model.FormatSize(sourceObj.Size))
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
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
		// log.Infof(`upload s3://%s/%s %s`, bucketName, strings.TrimPrefix(object, "/"), model.FormatSize(int64(len(data))))
		return nil
	}
	return fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 下载完整的文件
func (c *bucketClient) GetObject(profile, bucketName, object string) ([]byte, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
		}
		log.Debugf(tea.Prettify(input))
		resp, err := svc.GetObject(input)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 分片下载对象数据
func (c *bucketClient) GetObjectPart(profile, bucketName, object string, start, end int64) ([]byte, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(object),
			Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
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
		log.Debugf(tea.Prettify(resp))
		return model.Object{
			Key:          object,
			Size:         *resp.ContentLength,
			LastModified: *resp.LastModified,
			ETag:         *resp.ETag,
		}, nil
	}
	return model.Object{}, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 下载分片的时候按照分片大小进行分片下载
func (c *bucketClient) MutiDownloadObject(profileFrom, sourceBucket string, sourceObj model.Object, sourcePart int64, ch chan<- *model.ChData) {
	defer close(ch)
	var partIndex int64 = 0
	if c.isTencent(sourceBucket) {
		startIdx, endIdx := model.CalculateEvenSplits(sourceObj.Size)
		// Do we need more parts
		for j, start := range startIdx {
			partIndex++
			Start := start
			End := endIdx[j]
			data, err := c.GetObjectPart(profileFrom, sourceBucket, sourceObj.Key, Start, End)
			ch <- &model.ChData{
				PartIndex: partIndex,
				Data:      data,
				Err:       err,
			}
		}
	} else {
		// 依据源分片大小进行分片下载
		startIndex, endIndex, err := c.GetSourceSplit(profileFrom, sourceBucket, sourceObj.Key, sourcePart)
		if err != nil {
			ch <- &model.ChData{
				Err: err,
			}
			return
		}
		for j, start := range startIndex {
			partIndex++
			Start := start
			End := endIndex[j]
			log.Infof("partIndex:%d,start:%d,end:%d", partIndex, Start, End)
			data, err := c.GetObjectPart(profileFrom, sourceBucket, sourceObj.Key, Start, End)
			if err != nil {
				ch <- &model.ChData{
					Err: err,
				}
				break
			}
			ch <- &model.ChData{
				PartIndex: partIndex,
				Data:      data,
				Err:       err,
			}
		}
	}
}

func (c *bucketClient) MutiDownloadObjectThread(profileFrom, sourceBucket string, sourceObj model.Object, sourcePart int64, ch chan<- *model.ChData) {
	defer func() {
		log.Debugf("close ch")
		close(ch)
	}()
	threadChan := make(chan int, 8)
	var partIndex int64 = 0
	if c.isTencent(sourceBucket) {
		startIdx, endIdx := model.CalculateEvenSplits(sourceObj.Size)
		// Do we need more parts
		for j, start := range startIdx {
			partIndex++
			Start := start
			End := endIdx[j]
			threadChan <- 1
			go func(partIndex int64, Start, End int64) {
				data, err := c.GetObjectPart(profileFrom, sourceBucket, sourceObj.Key, Start, End)
				ch <- &model.ChData{
					PartIndex: partIndex,
					Data:      data,
					Err:       err,
				}
				<-threadChan
			}(partIndex, Start, End)
		}
	} else {
		// 依据源分片大小进行分片下载
		startIndex, endIndex, err := c.GetSourceSplit(profileFrom, sourceBucket, sourceObj.Key, sourcePart)
		if err != nil {
			ch <- &model.ChData{
				Err: err,
			}
			return
		}
		for j, start := range startIndex {
			partIndex++
			Start := start
			End := endIndex[j]
			threadChan <- 1
			go func(partIndex, Start, End int64) {
				log.Debugf("partIndex:%d,start:%d,end:%d", partIndex, Start, End)
				data, err := c.GetObjectPart(profileFrom, sourceBucket, sourceObj.Key, Start, End)
				if err != nil {
					ch <- &model.ChData{
						Err: err,
					}
				}
				ch <- &model.ChData{
					PartIndex: partIndex,
					Data:      data,
					Err:       err,
				}
				<-threadChan
			}(partIndex, Start, End)
		}
	}
	for {
		// log.Debugf("threadChan:%d", len(threadChan))
		if len(threadChan) == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (c *bucketClient) MutiReadFile(sourceObj model.LocalFile, sourcePart int64, ch chan *model.ChData) {
	var partIndex int64 = 0
	startIdx, endIdx := model.CalculateEvenSplits(sourceObj.Size)
	// Do we need more parts
	for j, start := range startIdx {
		partIndex++
		End := endIdx[j]
		// log.Debugf("partIndex:%d,start:%d,end:%d", partIndex, start, End)
		data := sourceObj.Data[start : End+1]
		// log.Debugf("len:%d,start:%v.end:%v,end:%v", len(data), sourceObj.Data[start], sourceObj.Data[End], sourceObj.Data[End-1])
		// log.Debugf("partIndex:%d,start:%d,end:%d", partIndex, data[0], data[len(data)-1])
		ch <- &model.ChData{
			PartIndex: partIndex,
			Data:      data,
			Err:       nil,
		}
	}
	close(ch)
}

// 依据文件大小判断是否需要分片，实现文件的拷贝；
// profile 必须同时具有sourceBucket和targetBucket的权限，或者指定一方，另一方权限公开读;
// 全部在云上操作，不需要下载到本地。
func (c *bucketClient) CopyObjectServerSide(sourceProfile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) (bool, error) {

	if c.isSameFile(sourceObj, sourceProfile, targetBucket, targetKey) {
		return true, nil
	}
	sourcePart := model.GetPartsCount(sourceObj.ETag)

	if sourcePart >= 1 {
		// 大于1个分片的文件，直接分片拷贝
		var partIndex int64 = 0
		startIdx, endIdx, err := c.GetSourceSplit(sourceProfile, sourceBucket, sourceObj.Key, sourcePart)
		if err != nil {
			return false, err
		}
		upload_id, err := c.CreateMutiUpload(sourceProfile, targetBucket, targetKey)
		if err != nil {
			return false, err
		}
		var completed_parts []*s3.CompletedPart
		for j, start := range startIdx {
			partIndex++
			Start := start
			End := endIdx[j]
			copySource := fmt.Sprintf("/%s/%s", sourceBucket, sourceObj.Key)
			copySourceRange := fmt.Sprintf("bytes=%d-%d", Start, End)
			part, err := c.UploadPart(sourceProfile, targetBucket, targetKey, copySource, copySourceRange, upload_id, partIndex)
			if err != nil {
				break
			}
			completed_parts = append(completed_parts, part)
			log.Infof(`UploadPartCopy s3://%s/%s %d/%d`, targetBucket, targetKey, partIndex, len(startIdx))
		}
		err = c.ComplateMutiPartUpload(sourceProfile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
			c.AbortMutiPartUpload(sourceProfile, targetBucket, targetKey, upload_id)
			return false, err
		} else {
			log.Infof(`muticopy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, targetKey, model.FormatSize(sourceObj.Size))
			return false, nil
		}
	} else {
		// TODO:需要支持文件到目录的格式
		err := c.copyObject(sourceProfile, sourceBucket, sourceObj, targetBucket, targetKey)
		return false, err
	}
}

// 依据文件大小决定是用PutObject，还是分片上传；这里设置文件大小为5G，超过5G的文件使用分片上传；
func (c *bucketClient) CopyObjectClientSide(sourceProfile, targetProfile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) (bool, error) {
	isSameEtag := false
	if c.isSameFile(sourceObj, targetProfile, targetBucket, targetKey) {
		isSameEtag = true
		return isSameEtag, nil
	}
	sourcePart := model.GetPartsCount(sourceObj.ETag)

	if sourcePart >= 1 {
		upload_id, err := c.CreateMutiUpload(targetProfile, targetBucket, targetKey)
		if err != nil {
			return isSameEtag, err
		}
		log.Debugf("upload_id: %s", upload_id)
		defer c.AbortMutiPartUpload(targetProfile, targetBucket, targetKey, upload_id)
		threadCache := 2 // 对象切片的缓存数量，约大越占内存，但可以提高单个分片对象的完成速度。
		ch := make(chan *model.ChData, threadCache)
		threadNum := make(chan int, threadCache)
		go c.MutiDownloadObjectThread(sourceProfile, sourceBucket, sourceObj, sourcePart, ch)
		var completed_parts []*s3.CompletedPart
		var completed_map_parts map[int64]*s3.CompletedPart = make(map[int64]*s3.CompletedPart)
		for mutiData := range ch {
			var isOk bool = true
			if mutiData.Err != nil {
				return isSameEtag, mutiData.Err
			}
			threadNum <- 1
			go func(ok *bool, mutiDataI model.ChData) {
				defer func() {
					<-threadNum
				}()
				part, err := c.UploadPartWithData(targetProfile, targetBucket, targetKey, upload_id, mutiDataI.PartIndex, mutiDataI.Data)
				if err != nil {
					log.Errorf(err.Error())
					*ok = false
				} else {
					log.Infof(`UploadPart s3://%s/%s %d/%d/%d`, targetBucket, targetKey, mutiDataI.PartIndex, len(completed_map_parts)+1, sourcePart)
					completed_map_parts[mutiDataI.PartIndex] = part
				}
			}(&isOk, *mutiData)
			if !isOk {
				break
			}
		}
		for {
			if len(threadNum) == 0 {
				break
			}
			time.Sleep(time.Second * 1)
		}
		// completed_map_parts 排序生成 completed_parts
		for i := int64(1); i <= sourcePart; i++ {
			completed_parts = append(completed_parts, completed_map_parts[i])
		}

		err = c.ComplateMutiPartUpload(targetProfile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
			log.Errorf("ComplateMutiPartUpload error: %s", err.Error())
			return isSameEtag, err
		} else {
			log.Infof(`muticopy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, targetKey, model.FormatSize(sourceObj.Size))
			return isSameEtag, nil
		}
	} else {
		data, err := c.GetObject(sourceProfile, sourceBucket, sourceObj.Key)
		if err != nil {
			return isSameEtag, err
		}
		err = c.UploadObject(targetProfile, targetBucket, targetKey, data)
		if err != nil {
			return isSameEtag, err
		}
		log.Infof(`copy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, strings.TrimPrefix(targetKey, "/"), model.FormatSize(sourceObj.Size))
		return isSameEtag, nil
	}
}

// 本地文件到远程文件的拷贝
func (c *bucketClient) CopyObjectLocalToRemote(targetProfile string, sourceObj model.LocalFile, targetBucket, targetKey string) (bool, error) {
	isSameEtag := false
	if c.isSameFile(model.Object{
		ETag: sourceObj.ETag,
	}, targetProfile, targetBucket, targetKey) {
		isSameEtag = true
		return isSameEtag, nil
	}
	sourcePart := model.PartsRequired(sourceObj.Size)
	var partIndex int64 = 0
	if sourcePart > 1 {
		upload_id, err := c.CreateMutiUpload(targetProfile, targetBucket, targetKey)
		if err != nil {
			return isSameEtag, err
		}
		ch := make(chan *model.ChData, sourcePart)
		go c.MutiReadFile(sourceObj, sourcePart, ch)
		var completed_parts []*s3.CompletedPart
		for mutiData := range ch {
			partIndex++
			part, err := c.UploadPartWithData(targetProfile, targetBucket, targetKey, upload_id, partIndex, mutiData.Data)
			if err != nil {
				log.Errorf(err.Error())
				break
			}
			completed_parts = append(completed_parts, part)
			log.Infof(`UploadPart s3://%s/%s %d/%d`, targetBucket, targetKey, partIndex, sourcePart)
		}
		err = c.ComplateMutiPartUpload(targetProfile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
			return isSameEtag, err
		} else {
			log.Infof(`muticopy %s => s3://%s/%s %s`, sourceObj.Key, targetBucket, targetKey, model.FormatSize(sourceObj.Size))
			return isSameEtag, nil
		}
	} else {
		err := c.UploadObject(targetProfile, targetBucket, targetKey, sourceObj.Data)
		if err != nil {
			return isSameEtag, err
		}
		log.Infof(`copy %s => s3://%s/%s %s`, sourceObj.Key, targetBucket, strings.TrimPrefix(targetKey, "/"), model.FormatSize(sourceObj.Size))
		return isSameEtag, nil
	}
}

func (c *bucketClient) isSameFile(object model.Object, targetProfile, targetBucket, targetKey string) bool {
	// 没有覆盖要去检查目标文件的etag
	dstObject, err := c.HeadObject(targetProfile, targetBucket, targetKey)
	if err != nil {
		// except 404
		if !strings.Contains(err.Error(), "404") {
			log.Errorf("head object %s/%s error:%s", targetBucket, targetKey, err.Error())
			return false
		}
	}
	log.Debugf("etags: s:%s d:%s", object.ETag, dstObject.ETag)
	if dstObject.ETag == "" {
		return false
	}
	if object.ETag == dstObject.ETag {
		return true
	}
	{
		// 由于腾讯云目前不支持分片对象的源和目标保持一直 etag，为了保证不重复上传，需要对腾讯云的对象进行文件 size和时间校验；
		if c.isTencent(targetBucket) {
			if dstObject.LastModified.Before(object.LastModified) {
				return false
			}
			if object.Size == dstObject.Size {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

// isTencent 是否是腾讯云,依据 bucketName是否包含-1251949819 结尾
func (c *bucketClient) isTencent(bucket string) bool {
	return strings.HasSuffix(bucket, "-1251949819")
}

// 获取源分片的起止
func (c *bucketClient) GetSourceSplit(sourceProfile, sourceBucket, key string, sourcePart int64) (startIndex, endIndex []int64, err error) {
	if sess, ok := c.sessions[sourceProfile]; ok {
		// 依据源分片大小进行分片下载
		svc := s3.New(sess)
		input := &s3.HeadObjectInput{
			Bucket: aws.String(sourceBucket),
			Key:    aws.String(key),
		}
		start := int64(0)
		for i := int64(0); i < sourcePart; i++ {
			input.PartNumber = aws.Int64(i + 1)
			resp, err := svc.HeadObject(input)
			if err != nil {
				log.Errorf("get source split error:%s", err.Error())
				return nil, nil, err
			}
			// log.Debugf("%d", *resp.ContentLength)
			end := start + *resp.ContentLength - 1
			startIndex = append(startIndex, start)
			endIndex = append(endIndex, end)
			start = end + 1
		}
		return startIndex, endIndex, nil
	}
	return nil, nil, errors.New(fmt.Sprintf("profile:%s not found", sourceProfile))
}
