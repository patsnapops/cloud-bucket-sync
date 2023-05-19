package model

import (
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	ID        string    `json:"id" gorm:"primary_key"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
	IsDeleted bool      `json:"is_deleted" gorm:"not null;default:false"`
	Cloud     string    `json:"cloud" gorm:"not null;default:'';"`
	Region    string    `json:"region" gorm:"not null;default:'';"`
	Hc        time.Time `json:"hc" gorm:"not null;type:timestamp;default:CURRENT_TIMESTAMP;"`
}

// to response
func (w *Worker) ToResponse() WorkerResponse {
	return WorkerResponse{
		Id:        w.ID,
		Cloud:     w.Cloud,
		Region:    w.Region,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
		Hc:        w.Hc,
	}
}

func (Worker) TableName() string {
	return "opst_worker"
}

type WorkerRequest struct {
	Cloud  string `json:"cloud"`
	Region string `json:"region"`
}

// new worker
func (w *WorkerRequest) NewWorker() Worker {
	return Worker{
		ID:     uuid.New().String(),
		Cloud:  w.Cloud,
		Region: w.Region,
	}
}

type WorkerResponse struct {
	Id        string    `json:"id"`
	Cloud     string    `json:"cloud"`
	Region    string    `json:"region"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Hc        time.Time `json:"hc"`
}

type WorkerInput struct {
	WorkerID string `json:"worker_id"`
	Cloud    string `json:"cloud"`
	Region   string `json:"region"`
}
