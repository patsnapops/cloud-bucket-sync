package model

import "time"

type Status string

func (s Status) String() string {
	return string(s)
}

type Mode string

const (
	// pending, waiting, running, success, failed, "cancel", notallsuccess
	TaskPending       Status = "pending"       //任务创建后的状态
	TaskWaiting       Status = "waiting"       //任务等待执行的状态
	TaskRunning       Status = "running"       //任务执行中的状态
	TaskSuccess       Status = "success"       //任务执行成功的状态
	TaskFailed        Status = "failed"        //任务执行失败的状态
	TaskCancel        Status = "cancel"        //任务被取消的状态
	TaskNotAllSuccess Status = "notallsuccess" //任务执行成功但是有部分文件失败的状态

	// type: syncOnce,KeepSync
	ModeSyncOnce Mode = "syncOnce"
	ModeKeepSync Mode = "keepSync"
)

// 基于任务信息发出的实际执行任务
type Record struct {
	ID           string    `json:"id" gorm:"primary_key,unique_index,not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
	TaskID       string    `json:"task_id" gorm:"not null"`                       // 任务ID
	RunningMode  Mode      `json:"running_mode" gorm:"not null;default:syncOnce"` // 运行模式, syncOnce,KeepSync
	Operator     string    `json:"operator"`                                      // 操作人
	Status       Status    `json:"status" gorm:"not null;default:pending"`        // 任务状态 请走action接口去修改。
	Progress     int64     `json:"Progress" gorm:"not null;default:0"`            // 进度 0-100
	IsConfirm    *bool     `json:"is_confirm" gorm:"not null;default:false"`      // 是否需要确认
	TotalFiles   int64     `json:"total_files"`                                   // 总文件数
	TotalSize    int64     `json:"total_size"`                                    // 总大小 单位 B
	SuccessFiles int64     `json:"success_files"`                                 // 成功文件数
	SuccessSize  int64     `json:"success_size"`                                  // 走公网消耗流量的大小 单位 B
	FailedFiles  int64     `json:"failed_files"`                                  // 失败文件数
	CostTime     int64     `json:"cost_time"`                                     // 耗时 单位 s
	WorkerID     string    `json:"worker_id"`                                     // 执行任务的workerID
	ErrorS3Url   string    `json:"error_s3_url"`                                  // 错误文件列表的s3地址
	ErrorHttpUrl string    `json:"error_http_url"`                                // 错误文件列表的http下载地址
	Info         string    `json:"info"`                                          // 任务信息
}

func (Record) TableName() string {
	return "manager_exec_record"
}

type RecordInput struct {
	TaskID   string `json:"task_id"`
	RecordID string `json:"record_id"`
	Status   Status `json:"status"`
}

type TaskWithRecord struct {
	Task
	Record []Record `json:"record"`
}
