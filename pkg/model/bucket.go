package model

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aws/aws-sdk-go/service/s3"
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

type SyncInput struct {
	Input
	Force  bool
	DryRun bool
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
	log.Debugf(tea.Prettify(input))
	return input
}

type ChData struct {
	Data      []byte
	PartIndex int64
	Err       error
}

type BucketIo interface {
	Presign(profile, bucketName, key string, expiresIn int64) (string, error)
	HeadObject(profile, bucketName, key string) (Object, error)

	GetObject(profile, bucketName, object string) ([]byte, error)
	GetSourceSplit(sourceProfile, sourceBucket, key string, sourcePart int64) (startIndex, endIndex []int64, err error)
	UploadObject(profile, bucketName, object string, data []byte) error

	ListObjects(profile, bucketName, prefix string, input Input) ([]string, []Object, error)
	ListObjectsWithChan(profile, bucketName, prefix string, input Input, objectsChan chan *ChanObject) //使用chan的方式降低内存占用并降低大量数据的等待时间

	RmObject(profile, bucketName, object string) error

	CreateMutiUpload(profile, bucketName, object string) (string, error)
	UploadPart(profile, bucketName, object, copySource, copySourceRange, uploadId string, partNumber int64) (*s3.CompletedPart, error)
	UploadPartWithData(profile, bucketName, object, uploadId string, partNumber int64, data []byte) (*s3.CompletedPart, error)
	MutiDownloadObject(profileFrom, sourceBucket string, sourceObj Object, sourcePart int64, ch chan<- *ChData)
	MutiDownloadObjectThread(profileFrom, sourceBucket string, sourceObj Object, sourcePart int64, ch chan<- *ChData)

	MutiReadFile(sourceObj LocalFile, sourcePart int64, ch chan *ChData)
	ComplateMutiPartUpload(profile, bucketName, object, uploadId string, completed_parts []*s3.CompletedPart) error

	// 高级封装的接口
	// target和source profile要一致，否则要保证目标段和源段的profile有权限
	CopyObjectServerSide(profile, sourceBucket string, sourceObj Object, targetBucket, targetKey string) (bool, error)
	CopyObjectClientSide(profileFrom, profileTo, sourceBucket string, sourceObj Object, targetBucket, targetKey string) (bool, error)

	CopyObjectLocalToRemote(targetProfile string, sourceObj LocalFile, targetBucket, targetKey string) (bool, error)
}

type ChanObject struct {
	Obj *Object
	Dir *string
	Err error
}

type ErrorKeys []ErrorKey

// fmt ErrorKeys to jsonFile
func (e ErrorKeys) ToJsonFile(record_id string) string {
	// 本地创建json文件
	jsonFile := path.Join("/opt/data/", record_id+"_error.json")
	// 创建文件
	f, err := os.Create(jsonFile)
	if err != nil {
		log.Errorf("create jsonFile error: %v", err)
		return err.Error()
	}
	defer f.Close()
	// 写入文件
	for _, v := range e {
		f.WriteString(v.Func + " " + v.Key + " " + strings.ReplaceAll(v.Err.Error(), "\n", "") + "\n")
	}
	log.Infof("write error keys to jsonFile: %v", jsonFile)
	return jsonFile
}

type ErrorKey struct {
	Func string
	Key  string
	Err  error
}

// 过滤对象，符合条件返回true 默认都符合
func ListObjectsWithFilter(key Object, input Input) bool {

	contain := false
	if len(input.Include) != 0 {
		for _, include := range input.Include {
			// log.Debugf("key: %v, include: %v", key.Key, include)
			if include == "" {
				// log.Debugf("%v", include == "")
				contain = true
				continue
			}
			if strings.Contains(key.Key, include) {
				contain = true
			}
		}
	} else {
		contain = true
	}
	// 默认对象都是不剔除的
	excludeB := false
	if len(input.Exclude) != 0 {
		for _, exclude := range input.Exclude {
			log.Debugf("key: %v, exclude: %v", key.Key, exclude)
			if exclude == "" {
				log.Debugf("%v", exclude == "")
				excludeB = false
				continue
			}
			if strings.Contains(key.Key, exclude) {
				excludeB = true
			}
		}
	} else {
		excludeB = false
	}

	timeAfterB := false
	if input.TimeAfter != nil {
		if key.LastModified.Before(*input.TimeAfter) {
			timeAfterB = true
		}
	}
	timeBeforeB := false
	if input.TimeBefore != nil {
		if key.LastModified.After(*input.TimeBefore) {
			timeBeforeB = true
		}
	}
	log.Debugf("key: %v, contain: %v, excludeB: %v, timeAfterB: %v, timeBeforeB: %v", key, contain, excludeB, timeAfterB, timeBeforeB)
	if !contain || excludeB || timeAfterB || timeBeforeB {
		return false
	}
	return true
}

// turn s3://bucket/prefix to bucket and prefix
func ParseBucketAndPrefix(s3Path string) (bucket, prefix string) {
	bucketS := strings.Split(strings.TrimPrefix(s3Path, "s3://"), "/")
	bucket = bucketS[0]
	if len(bucketS) > 1 {
		prefix = strings.Join(bucketS[1:], "/")
	} else {
		prefix = ""
	}
	log.Debugf("bucket: %s, prefix: %s", bucket, prefix)
	return
}

// 同步模式计算目标对象的key，依据原本的KEY和原本的前缀，以及目标前缀
// targetPrefix 如果没有以/结尾，则源对象的key会直接加在后面
func GetTargetKey(key, prefix, targetPrefix string) string {
	if prefix == "" {
		return targetPrefix + key
	}
	fileName := path.Base(prefix)
	prefix = strings.TrimSuffix(prefix, fileName)
	if strings.HasSuffix(prefix, "/") {
		// 目录同步
		return targetPrefix + strings.TrimPrefix(key, prefix)
	} else {
		// 文件同步，支持自动补上文件名字
		// targetPrefix = strings.TrimPrefix(targetPrefix, "/")
		if strings.HasPrefix(strings.TrimPrefix(key, prefix), "/") {
			// 目标前缀是目录，且key是以/开头的，需要去掉/
			return targetPrefix + strings.TrimPrefix(key, prefix)[1:]
		}
		return targetPrefix + strings.TrimPrefix(key, prefix)
	}
}

func ListObjectsWithChanLocalRecursive(
	localPath string, recursive bool, input SyncInput, objectChan chan *LocalFile) {
	// 判断 localPath 是文件还是目录
	info, err := os.Stat(localPath)
	if err != nil {
		log.Panicf("stat localPath error: %v", err)
	}
	if info.IsDir() {
		if recursive {
			// 递归获取目录下的所有文件
			ListObjectsLocalRecursive(localPath, input, objectChan)
		} else {
			// 只获取目录下的文件
			ListObjectsWithChanLocal(localPath, input, objectChan)
		}
	} else {
		// 文件
		data, _ := os.ReadFile(localPath)
		// log.Panicf("localPath: %v, data_len: %d", localPath, len(data))
		etag, _ := CalculateHashForLocalFile(localPath, "md5")
		objectChan <- &LocalFile{
			Key:     path.Base(localPath),
			ETag:    etag,
			ModTime: info.ModTime(),
			Size:    info.Size(),
			Data:    data,
		}
	}
	defer close(objectChan)
}

// 获取本地目录下的所有文件，不递归
func ListObjectsWithChanLocal(localPath string, input SyncInput, objectChan chan *LocalFile) {
	// 只获取目录下的文件
	files, err := os.ReadDir(localPath)
	if err != nil {
		log.Errorf("read dir error: %v", err)
	} else {
		for _, file := range files {
			if !file.IsDir() {
				localFile := new(LocalFile)
				localFile.Dir = localPath
				localFile.Key = file.Name()
				path := localPath + localFile.Key
				info, err := os.Stat(path)
				if err != nil {
					log.Errorf("stat file error: %v", err)
					continue
				}
				localFile.ModTime = info.ModTime()
				localFile.Size = info.Size()
				data, _ := os.ReadFile(path)
				localFile.Data = data
				etag, _ := CalculateHashForLocalFile(path, "md5")
				localFile.ETag = etag
				// log.Panicf(tea.Prettify(localFile))
				objectChan <- localFile
			} else {
				log.Warnf("skip dir: %v", file.Name())
			}
		}
	}
}

// 获取本地目录下的所有文件，递归
func ListObjectsLocalRecursive(localPath string, input SyncInput, objectChan chan *LocalFile) {
	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("walk path error: %v", err)
			return err
		}
		if !info.IsDir() {
			localFile := new(LocalFile)
			localFile.Dir = localPath
			localFile.Key = strings.TrimPrefix(path, strings.TrimPrefix(localPath, "./"))
			etag, _ := CalculateHashForLocalFile(path, "md5")
			localFile.ETag = etag
			data, _ := os.ReadFile(path)
			localFile.Data = data
			localFile.ModTime = info.ModTime()
			localFile.Size = info.Size()
			// log.Panicf(tea.Prettify(localFile))
			objectChan <- localFile
		}
		return nil
	})
	if err != nil {
		log.Errorf("walk path error: %v", err)
	}
}
