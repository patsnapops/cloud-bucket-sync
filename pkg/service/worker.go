package service

import (
	"cbs/pkg/model"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/patsnapops/noop/log"
)

type WorkerService struct {
	BucketIo  model.BucketIo
	RequestC  model.RequestContract
	ThreadNum int64
}

func NewWorkerService(bucketIo model.BucketIo, requestC model.RequestContract, threadNum int64) model.WorkerContract {
	return &WorkerService{
		BucketIo:  bucketIo,
		RequestC:  requestC,
		ThreadNum: threadNum,
	}
}

// 一次同步任务，不考虑增量，对象来源于源桶调用的ls接口
func (w *WorkerService) SyncOnce(task model.Task, record *model.Record) {
	errorKeys := make(model.ErrorKeys, 0)
	startTime := time.Now()
	objectsChan := make(chan *model.ChanObject, 1000)
	sourceBucket, sourcePrefix := model.ParseBucketAndPrefix(task.SourceUrl)
	targetBucket, targetPrefix := model.ParseBucketAndPrefix(task.TargetUrl)
	go func(*model.Record) {
		for {
			// update record.
			record.CostTime = int64(time.Now().Sub(startTime).Seconds())
			log.Debugf("update.%v", *record)
			err := w.RequestC.RecordUpdate(record)
			if err != nil {
				log.Errorf("record update error: %v", err)
			}
			if record.Status != model.TaskRunning {
				log.Infof("stop sync record %v", *record)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}(record)
	go w.BucketIo.ListObjectsWithChan(task.SourceProfile, sourceBucket, sourcePrefix, model.Input{
		Recursive:  true,
		Include:    strings.Split(task.Include, ","),
		Exclude:    strings.Split(task.Exclude, ","),
		TimeBefore: model.StringToTime(task.TimeBefore),
		TimeAfter:  model.StringToTime(task.TimeAfter),
		Limit:      0,
	}, objectsChan)
	log.Infof("start sync task %v", task)
	// 设置并发数
	threadNumChan := make(chan int8, 10)
	for object := range objectsChan {
		log.Debugf("object: %s", tea.Prettify(object))
		threadNumChan <- 1
		go func(object *model.ChanObject) {
			targetKey := model.GetTargetKey(object.Obj.Key, sourcePrefix, targetPrefix)
			if task.IsServerSide {
				err := w.BucketIo.CopyObjectServerSide(task.SourceProfile, sourceBucket, *object.Obj, targetBucket, targetKey)
				if err != nil {
					errorKeys = append(errorKeys, model.ErrorKey{
						Func: "CopyObjectServerSide",
						Key:  object.Obj.Key,
						Err:  err,
					})
					record.FailedFiles++
				}
				log.Infof("%s - copy object %s/%s success", record.Id, targetBucket, object.Obj.Key)
			} else {
				err := w.BucketIo.CopyObjectClientSide(task.SourceProfile, task.TargetProfile, sourceBucket, *object.Obj, targetBucket, targetKey)
				if err != nil {
					errorKeys = append(errorKeys, model.ErrorKey{
						Func: "CopyObjectClientSide",
						Key:  object.Obj.Key,
						Err:  err,
					})
					record.FailedFiles++
				}
				log.Infof("%s upload object %s/%s success", record.Id, targetBucket, object.Obj.Key)
			}
			record.TotalFiles++
			record.TotalSize += object.Obj.Size
			<-threadNumChan
		}(object)
	}
	for {
		if len(threadNumChan) == 0 {
			record.Status = model.TaskSuccess
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 汇总结果
	if len(errorKeys) > 0 {
		// not all success
		record.Status = model.TaskNotAllSuccess
		log.Infof("sync task %s record %s not all success totalFile:%d,totalSize:%s", task.Id, record.Id, record.TotalFiles, model.FormatSize(record.TotalSize))
		filePath := errorKeys.ToJsonFile(record.Id)
		record.Info = *filePath //TODO: upload to server.
	} else {
		// all success
		record.Status = model.TaskSuccess
		log.Infof("sync task %s record %s success totalFile:%d,totalSize:%s", task.Id, record.Id, record.TotalFiles, model.FormatSize(record.TotalSize))
	}
	w.RequestC.RecordUpdateStatus(record.Id, record.Status)
}

// $0.0025 per 1 million objects listed in S3 Inventory
// 实现方法主要考虑到2个：
// 1. 循环跑一次同步，第一次跑的时候，记录下所有的对象（大小，md5），然后下次跑的时候，对比记录，如果有变化，就同步.目标端对象直接覆盖（缺点，每次都是全量ls，aws 底层ls接口不支持时间条件过滤，增加API接口调用的费用。1,662,553,993对象 4.12刀）
// 1.1 关于本地记录，可行性不大。因为对象数量太多，本地记录的IO查询开销大。如果是Reids 1亿个对象入到内存，1亿对象 * 174字节/对象 ≈ 17.4 GB 内存消耗也很大。如果是kafka，感觉应该可以。
// 2. 使用桶事件来获取对象，然后同步。（需要额外的桶去配置，人工成本大，配置多，繁琐。）
func (w *WorkerService) KeepSync(task model.Task, record *model.Record) {
	// 修正首次执行的时间限制，往前推1个月的对象
	task.TimeAfter = time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 15:04:05")
	for {
		log.Infof("start sync task %s", tea.Prettify(task))
		w.SyncOnce(task, record)
		// 修正下次执行的时间限制，往后推10分钟的对象
		time.Sleep(time.Minute * 10)
		task.TimeAfter = time.Now().UTC().Add(time.Minute * 5).Format("2006-01-02 15:04:05")
	}
}
