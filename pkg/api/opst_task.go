package api

import (
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/noop/log"
)

// @Summary get task list
// @Description get all task list
// @Tags task
// @Accept  json
// @Produce  json
// @Param task_id query string false "task id"
// @Param name query string false "task name"
// @Param worker_tag query string false "worker_tag"
// @Success 200 {object} []model.Task
// @Failure 500 {object} string
// @Router /api/v1/task [get]
func GetTaskList(c *gin.Context) {
	resp, err := managerIo.QueryTask(model.TaskInput{
		ID:        c.Query("task_id"),
		Name:      c.Query("name"),
		WorkerTag: c.Query("worker_tag"),
	})
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp)
}

// @Summary get task detail
// @Description get task detail
// @Tags  task
// @Accept  json
// @Produce  json
// @Param id path string true "task id"
// @Success 200 {object} []model.Task
// @Failure 500 {object} string
// @Router /api/v1/task/{id} [get]
func GetTaskDetail(c *gin.Context) {
	resp, err := managerIo.QueryTask(model.TaskInput{
		ID: c.Param("id"),
	})
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp)
}

// @Summary create task
// @Description create task, sourceurl 和 targeturl 支持目录，targeturl 不支持文件，如果写文件默认当作dir处理。
// @Tags  task
// @Accept  json
// @Produce  json
// @Param task body model.Task true "task"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/task [post]
func CreateTask(c *gin.Context) {
	var req model.Task
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	taskID, err := managerIo.CreateTask(&req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	// 任务创建成功后，判断是否有定时任务配置决定是否立即启动任务
	if req.Corn == "" {
		recordID, err := managerIo.ExecuteTask(taskID, req.Submitter, "")
		if err != nil {
			log.Errorf("execute task error: %v", err)
			c.JSON(500, err.Error())
			return
		}
		log.Infof("execute task success, taskID: %s, recordID: %s", taskID, recordID)
	}

	c.JSON(200, taskID)
}

// @Summary update task
// @Description update task
// @Tags  task
// @Accept  json
// @Produce  json
// @Param id path string true "task id"
// @Param task body model.Task true "task"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/task/{id} [put]
func UpdateTask(c *gin.Context) {
	var req model.Task
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	err := managerIo.UpdateTask(&req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, "ok")
}

// @Summary delete task
// @Description delete task
// @Tags  task
// @Accept  json
// @Produce  json
// @Param id path string true "task id"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/task/{id} [delete]
func DeleteTask(c *gin.Context) {
	err := managerIo.DeleteTask(c.Param("id"))
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, "ok")
}

type ChangeRecordStatusRequest struct {
	Operator string `json:"operator"`
	RecordID string `json:"record_id"`
	Status   string `json:"status"`
}

// @Summary chang record status
// @Description only this api can change record status.
// @Tags record
// @Accept  json
// @Produce  json
// @Param action body ChangeRecordStatusRequest  true "record chang"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/status [post]
func ChangeRecordStatus(c *gin.Context) {
	var req ChangeRecordStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	err := managerIo.UpdateRecord(&model.Record{
		ID:     req.RecordID,
		Status: model.Status(req.Status),
	})
	if err != nil {
		log.Errorf("execute task error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, "ok")
}

type ExecuteTaskRequest struct {
	Operator string `json:"operator"`
	TaskID   string `json:"task_id"`
	RunMode  string `json:"run_mode"` // syncOnce keepSync
}

// @Summary execute task
// @Description execute task
// @Tags task
// @Accept  json
// @Produce  json
// @Param action body ExecuteTaskRequest true "task execute"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/execute [post]
func ExecuteTask(c *gin.Context) {
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	recordID, err := managerIo.ExecuteTask(req.TaskID, req.Operator, model.Mode(req.RunMode))
	if err != nil {
		log.Errorf("execute task error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, recordID)
}
