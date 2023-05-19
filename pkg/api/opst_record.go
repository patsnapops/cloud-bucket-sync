package api

import (
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/noop/log"
)

// @Summary get task record list
// @Description get task record list
// @Tags  opst/record
// @Accept  json
// @Produce  json
// @Param task_id query string false "task id"
// @Param status query string false "status"
// @Param record_id query string false "record id"
// @success 200 {object} []model.SyncTaskRecord
// @Failure 500 {object} string
// @Router /v2023-03/opst/record [get]
func GetTaskRecordList(c *gin.Context) {
	taskID := c.Query("task_id")
	status := c.Query("status")
	recordID := c.Query("record_id")
	input := model.RecordInput{
		TaskID:   taskID,
		Status:   model.Status(status),
		RecordID: recordID,
	}
	log.Debugf("input: %+v", input)
	res, err := managerIo.QueryRecord(input)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, res)
}

// @Summary update task record
// @Description update task record;不支持status的修改，修改status需要调用接口 action接口
// @Tags  opst/record
// @Accept  json
// @Produce  json
// @Param id path string true "task id"
// @Param record body model.RecordRequest true "task record"
// @Success 200 {object} model.TaskResponse
// @Failure 500 {object} string
// @Router /v2023-03/opst/record/{id} [put]
func UpdateTaskRecord(c *gin.Context) {
	var req model.Record
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	message := ""
	if req.Status != "" {
		message += "(status is not support to update by this api, please use action api to update status.)"
	}
	req.Status = "" // 不支持status的修改，修改status需要调用接口 action接口，过滤掉status
	err := managerIo.UpdateRecord(&req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, message+"update success")
}
