package api

import (
	"cbs/pkg/model"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/noop/log"
)

// @Summary get task list
// @Description get all task list
// @Tags task
// @Accept  json
// @Produce  json
// @Param id query string false "task id"
// @Param name query string false "task name 支持模糊匹配"
// @Param worker_tag query string false "worker_tag"
// @Success 200 {object} []model.Task
// @Failure 500 {object} string
// @Router /api/v1/task [get]
func GetTaskList(c *gin.Context) {
	resp, err := managerIo.QueryTask(model.TaskInput{
		ID:        c.Query("id"),
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
	if len(resp) != 1 {
		err = fmt.Errorf("task not found")
	}
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp[0])
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
	taskID, err := managerIo.CreateTask(&req, managerConfig)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	// 任务创建成功后，判断是否有定时任务配置决定是否立即启动任务
	if req.Corn == "" {
		recordID, err := managerIo.ExecuteTask(taskID, req.Submitter, req.SyncMode)
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
	log.Infof("change record status, recordID: %s, status: %s by %s", req.RecordID, req.Status, req.Operator)
	err := managerContract.UpdateRecordStatus(req.RecordID, model.Status(req.Status))
	if err != nil {
		log.Errorf("execute task error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, "ok")
}

// @Summary execute task
// @Description execute task
// @Tags task
// @Accept  json
// @Produce  json
// @Param action body model.TaskExecInput true "task execute"
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /api/v1/execute [post]
func ExecuteTask(c *gin.Context) {
	var req model.TaskExecInput
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	log.Debugf("execute task: %+v", req)
	// 获取任务信息
	task, err := managerIo.GetTaskById(req.TaskID)
	if err != nil {
		log.Errorf("execute task error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	if req.SyncMode == "" {
		req.SyncMode = task.SyncMode
	}
	if req.SyncMode != "syncOnce" && req.SyncMode != "keepSync" {
		c.JSON(500, fmt.Sprintf("syncMode error %s", req.SyncMode))
		return
	}
	recordID, err := managerIo.ExecuteTask(req.TaskID, req.Operator, req.SyncMode)
	if err != nil {
		log.Infof("execute task error: %v", err)
		c.JSON(500, err.Error())
		return
	}
	log.Infof("execute task success, taskID: %s, recordID: %s", req.TaskID, recordID)
	c.JSON(200, recordID)
}
