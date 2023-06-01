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
		// log.Debugf("new session %s", param.Name)
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
	log.Debugf("list objects %s/%s", bucketName, prefix)
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
			// PartNumber: aws.Int64(7),
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

func (c *bucketClient) MutiDownloadObject(profileFrom, sourceBucket string, sourceObj model.Object, sourcePart, contentLength int64, ch chan<- *model.ChData) {
	var partIndex int64 = 0
	startIdx, endIdx := model.CalculateEvenSplitsByPartSize(sourceObj.Size, contentLength)
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
	close(ch)
}

// 依据文件大小判断是否需要分片，实现文件的拷贝；
// profile 必须同时具有sourceBucket和targetBucket的权限，或者指定一方，另一方权限公开读;
// 全部在云上操作，不需要下载到本地。
func (c *bucketClient) CopyObjectServerSide(profile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) (bool, error) {

	if c.isSameFile(sourceObj, profile, targetBucket, targetKey) {
		return true, nil
	}
	sourcePart, err := model.GetPartsCount(sourceObj.ETag)
	if err != nil {
		sourcePart = 1
	}
	contentLength, err := c.getSourceContentLength(profile, sourceBucket, sourceObj.Key)
	if err != nil {
		return false, err
	}
	if sourcePart > 1 {
		// 大于1个分片的文件，直接分片拷贝
		var partIndex int64 = 0
		upload_id, err := c.CreateMutiUpload(profile, targetBucket, targetKey)
		if err != nil {
			return false, err
		}
		startIdx, endIdx := model.CalculateEvenSplitsByPartSize(sourceObj.Size, contentLength)
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
			log.Infof(`UploadPartCopy s3://%s/%s %d/%d`, targetBucket, targetKey, partIndex, sourcePart)
		}
		err = c.ComplateMutiPartUpload(profile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
			return false, err
		} else {
			log.Infof(`muticopy s3://%s/%s => s3://%s/%s %s`, sourceBucket, sourceObj.Key, targetBucket, targetKey, model.FormatSize(sourceObj.Size))
			return false, nil
		}
	} else {
		// TODO:需要支持文件到目录的格式
		err := c.copyObject(profile, sourceBucket, sourceObj, targetBucket, targetKey)
		return false, err
	}
}

// 判断是否进行分片传输，是返回true，否返回false，并且返回需要分片的大小
func (c *bucketClient) getSourceContentLength(profile, bucketName, object string) (int64, error) {
	if sess, ok := c.sessions[profile]; ok {
		svc := s3.New(sess)
		input := &s3.HeadObjectInput{
			Bucket:     aws.String(bucketName),
			Key:        aws.String(object),
			PartNumber: aws.Int64(1),
		}
		resp, err := svc.HeadObject(input)
		if err != nil {
			return 0, err
		}
		log.Debugf(tea.Prettify(resp))
		return *resp.ContentLength, nil
	}
	return 0, fmt.Errorf("profile %s not found,please check cli.yaml config.", profile)
}

// 依据文件大小决定是用PutObject，还是分片上传；这里设置文件大小为5G，超过5G的文件使用分片上传；
func (c *bucketClient) CopyObjectClientSide(sourceProfile, targetProfile, sourceBucket string, sourceObj model.Object, targetBucket, targetKey string) (bool, error) {
	isSameEtag := false
	if c.isSameFile(sourceObj, targetProfile, targetBucket, targetKey) {
		isSameEtag = true
		return isSameEtag, nil
	}
	sourcePart, err := model.GetPartsCount(sourceObj.ETag)
	if err != nil {
		sourcePart = 1
	}
	contentLength, err := c.getSourceContentLength(sourceProfile, sourceBucket, sourceObj.Key)
	if err != nil {
		return isSameEtag, err
	}
	if sourcePart > 1 {
		var partIndex int64 = 0
		upload_id, err := c.CreateMutiUpload(targetProfile, targetBucket, targetKey)
		if err != nil {
			return isSameEtag, err
		}
		ch := make(chan *model.ChData, 100)
		go c.MutiDownloadObject(sourceProfile, sourceBucket, sourceObj, sourcePart, contentLength, ch)
		var completed_parts []*s3.CompletedPart
		for mutiData := range ch {
			partIndex++
			part, err := c.UploadPartWithData(targetProfile, targetBucket, targetKey, upload_id, partIndex, mutiData.Data)
			if err != nil {
				log.Fatal(err.Error())
				break
			}
			completed_parts = append(completed_parts, part)
			log.Infof(`UploadPart s3://%s/%s %d/%d`, targetBucket, targetKey, partIndex, sourcePart)
		}
		err = c.ComplateMutiPartUpload(targetProfile, targetBucket, targetKey, upload_id, completed_parts)
		if err != nil {
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

// 建议在使用 ETag 进行文件比较时，结合其他属性如文件大小和最后修改时间等进行综合判断，以确保准确性。
func (c *bucketClient) isSameFile(object model.Object, targetProfile, targetBucket, targetKey string) bool {
	// 没有覆盖要去检查目标文件的etag
	dstObject, err := c.HeadObject(targetProfile, targetBucket, targetKey)
	if err != nil {
		// except 404
		if !strings.Contains(err.Error(), "404") {
			log.Errorf("head object error:%s", err.Error())
			return false
		}
	}
	// 判断修改时间，如果源文件修改时间大于目标文件，那么就需要覆盖
	if object.LastModified.After(dstObject.LastModified) {
		log.Debugf("source file is  %s/%s modified. overwrite", targetBucket, targetKey)
		return false
	}

	if object.ETag == dstObject.ETag {
		return true
	}

	return false
}
