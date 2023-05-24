package model

import "time"

type Task struct {
	ID            string    `json:"id" gorm:"primary_key,unique_index,not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted     bool      `json:"is_deleted" gorm:"not null;default:false"`
	WorkerTag     string    `json:"worker" gorm:"not null;default:''"`                                                                                            // worker节点,task创建的时候创建
	Name          string    `json:"name" gorm:"not null" binding:"required"`                                                                                      // 任务名称
	SourceUrl     string    `json:"source_url" gorm:"not null" binding:"required"`                                                                                // S3URL s3://sourceBucket/key 支持文件和目录结尾
	TargetUrl     string    `json:"target_url" gorm:"not null" binding:"required"`                                                                                // S3URL s3://destBucket/dir/ 不支持文件结尾 没有/的目录看作目录处理
	SourceProfile string    `json:"source_profile" gorm:"not null" binding:"required" example:"cn3977"`                                                           // 源Profile配置 可选 cn9554,cn3977,cn0536,us7478,us0066,us1549,tx-cn,tx-us
	TargetProfile string    `json:"target_profile" gorm:"not null" binding:"required" example:"us7478"`                                                           // 目标Profile配置 可选 cn9554,cn3977,cn0536,us7478,us0066,us1549,tx-cn,tx-us
	SyncMode      Mode      `json:"sync_mode" gorm:"not null;default:syncOnce" binding:"required" example:"syncOnce"`                                             // 默认运行模式 syncOnce 一次性任务, KeepSync 持续同步
	Submitter     string    `json:"submitter" binding:"required"`                                                                                                 // 提交人
	Corn          string    `json:"corn"  gorm:"not null;default:''" example:"0 */8 * * 1,2,3,4,5" `                                                              // 格式为 分、时、日、月、周                                                      // cron表达式 用于定时任务 ’分 时 日 月 周‘
	KeysUrl       string    `json:"keys_url" gorm:"not null;default:''" example:"s3://bucket/key"`                                                                // S3URL s3://bucket/key 支持提供文件列表去同步
	IsSilence     *bool     `json:"is_silence" gorm:"not null;default:false" example:"false"`                                                                     // 是否静默 默认false 静默不发送通知
	IsOverwrite   *bool     `json:"is_overwrite,omitempty" gorm:"not null;default:false"`                                                                         // 是否覆盖 默认覆盖 true
	TimeBefore    string    `json:"time_before"  example:"1992-03-01 21:26:30"`                                                                                   // 在某个时间之前 UTC时间格式：“1992-03-01 21:26:30”
	TimeAfter     string    `json:"time_after"  example:"1992-03-01 21:26:30"`                                                                                    // 在某个时间之后 UTC时间格式：“1992-03-01 21:26:30”
	Include       string    `json:"include" gorm:"type:text"`                                                                                                     // 包含
	Exclude       string    `json:"exclude" gorm:"type:text"`                                                                                                     // 排除
	StorageClass  string    `json:"storage_class" gorm:"not null;default:STANDARD"`                                                                               // 存储类型 STANDARD,STANDARD_IA,ONEZONE_IA,INTELLIGENT_TIERING,REDUCED_REDUNDANCY,STANDARD_IA,ONEZONE_IA,INTELLIGENT_TIERING,REDUCED_REDUNDANCY
	Meta          string    `json:"meta" gorm:"type:text" example:"Expires:2022-10-12T00:00:00.000Z#Cache-Control:no-cache#Content-Encoding:gzip#x-cos-meta-x:x"` // 任务元信息
}

func (Task) TableName() string {
	return "manager_task"
}

type TaskInput struct {
	ID        string `json:"id" gorm:"primary_key,unique_index,not null"`
	Name      string `json:"name" gorm:"not null" binding:"required"` // 任务名称
	WorkerTag string `json:"worker_tag" gorm:"not null;default:''"`   // worker节点
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

// args to ObjectFilter
func (t Task) ToFilter() *ObjectFilter {
	return &ObjectFilter{
		Include:    t.Include,
		Exclude:    t.Exclude,
		TimeBefore: t.TimeBefore,
		TimeAfter:  t.TimeAfter,
	}
}
