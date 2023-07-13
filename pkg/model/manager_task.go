package model

import "time"

type Task struct {
	Id                 string    `json:"id" gorm:"primary_key,unique_index,not null"`
	CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted          bool      `json:"is_deleted" gorm:"not null;default:false"`
	WorkerTag          string    `json:"worker_tag" gorm:"not null;default:''" binding:"required"`                                                                     // 任务执行节点, 用于标记任务归属于哪个worker,会涉及到费用，需要注意选择正确的workerTag。借鉴与gitlab CICD runner.
	IsServerSide       *bool     `json:"is_server_side" gorm:"not null;default:false" binding:"required"`                                                              // 是否使用云服务商后台执行，决定是否在本地产生流量。只有同一个云厂商后台才可能支持。默认为 false。走后台传输会省流量费用。
	Name               string    `json:"name" gorm:"not null" binding:"required"`                                                                                      // 任务名称
	SourceUrl          string    `json:"source_url" gorm:"not null" binding:"required"`                                                                                // S3URL s3://sourceBucket/key 支持文件和目录结尾
	TargetUrl          string    `json:"target_url" gorm:"not null" binding:"required"`                                                                                // S3URL s3://destBucket/dir/ 不支持文件结尾 没有/的目录看作目录处理
	SourceProfile      string    `json:"source_profile" gorm:"not null" binding:"required" default:"proxy" example:"cn3977"`                                           // 源Profile配置 可选 cn9554,cn3977,cn0536,us7478,us0066,us1549,tx-cn,tx-us
	TargetProfile      string    `json:"target_profile" gorm:"not null" binding:"required" default:"proxy" example:"us7478"`                                           // 目标Profile配置 可选 cn9554,cn3977,cn0536,us7478,us0066,us1549,tx-cn,tx-us
	SyncMode           string    `json:"sync_mode" gorm:"not null;default:syncOnce" binding:"required" example:"syncOnce"`                                             // 默认运行模式 syncOnce 一次性任务, KeepSync 持续同步
	Submitter          string    `json:"submitter" binding:"required"`                                                                                                 // 提交人
	Corn               string    `json:"corn"  gorm:"not null;default:''" example:"0 */8 * * 1,2,3,4,5" `                                                              // 格式为 分、时、日、月、周                                                      // cron表达式 用于定时任务 ’分 时 日 月 周‘
	KeysUrl            string    `json:"keys_url" gorm:"not null;default:''" example:"s3://bucket/key"`                                                                // S3URL s3://bucket/key 支持提供文件列表去同步
	IsSilence          *bool     `json:"is_silence" gorm:"not null;default:false" example:"false"`                                                                     // 是否静默 默认false 静默不发送通知
	TimeBefore         string    `json:"time_before"  example:"1992-03-01 21:26:30"`                                                                                   // 在某个时间之前 UTC时间格式：“1992-03-01 21:26:30”
	TimeAfter          string    `json:"time_after"  example:"1992-03-01 21:26:30"`                                                                                    // 在某个时间之后 UTC时间格式：“1992-03-01 21:26:30”
	Include            string    `json:"include" gorm:"type:text"`                                                                                                     // 包含
	Exclude            string    `json:"exclude" gorm:"type:text"`                                                                                                     // 排除
	StorageClass       string    `json:"storage_class" gorm:"not null;default:STANDARD"`                                                                               // 存储类型 STANDARD,STANDARD_IA,ONEZONE_IA,INTELLIGENT_TIERING,REDUCED_REDUNDANCY,STANDARD_IA,ONEZONE_IA,INTELLIGENT_TIERING,REDUCED_REDUNDANCY
	Meta               string    `json:"meta" gorm:"type:text" example:"Expires:2022-10-12T00:00:00.000Z#Cache-Control:no-cache#Content-Encoding:gzip#x-cos-meta-x:x"` // 任务元信息
	ApproveResult      string    `json:"approve_result" gorm:"not null;default:''" example:"agree"`                                                                    // 审批结果 agree,refuse
	DingtalkInstanceId string    `json:"dingtalk_instance_id" gorm:"not null;default:''" `
}

func (t *Task) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             t.Id,
		"created_at":     t.CreatedAt,
		"updated_at":     t.UpdatedAt,
		"is_deleted":     t.IsDeleted,
		"worker_tag":     t.WorkerTag,
		"is_server_side": t.IsServerSide,
		"name":           t.Name,
		"source_url":     t.SourceUrl,
		"target_url":     t.TargetUrl,
		"source_profile": t.SourceProfile,
		"target_profile": t.TargetProfile,
		"sync_mode":      t.SyncMode,
		"submitter":      t.Submitter,
		"corn":           t.Corn,
		"keys_url":       t.KeysUrl,
		"is_silence":     t.IsSilence,
		"time_before":    t.TimeBefore,
		"time_after":     t.TimeAfter,
		"include":        t.Include,
		"exclude":        t.Exclude,
		"storage_class":  t.StorageClass,
		"meta":           t.Meta,
	}
}

func StringToTime(str string) *time.Time {
	if str == "" {
		return nil
	}
	t, _ := time.Parse("2006-01-02 15:04:05", str)
	return &t
}

func (Task) TableName() string {
	return "manager_task"
}

type TaskInput struct {
	ID         string `json:"id" gorm:"primary_key,unique_index,not null"`
	Name       string `json:"name" gorm:"not null" binding:"required"` // 任务名称，支持模糊匹配
	WorkerTag  string `json:"worker_tag" gorm:"not null;default:''"`   // worker节点
	InstanceId string `json:"instance_id" gorm:"not null;default:''"`  // 钉钉审批实例id
}

func (t *TaskInput) ToQuery() string {
	var query []string
	if t.Name != "" {
		query = append(query, "name="+t.Name)
	}
	if t.WorkerTag != "" {
		query = append(query, "worker_tag="+t.WorkerTag)
	}
	if t.ID != "" {
		query = append(query, "id="+t.ID)
	}
	return joinQuery(query)
}
