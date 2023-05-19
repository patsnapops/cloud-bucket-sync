package api

import (
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
)

// @Summary get worker list
// @Description get all worker list
// @Tags opst/worker
// @Accept  json
// @Produce  json
// @Success 200 {object} []model.WorkerResponse
// @Failure 500 {object} string
// @Router /v2023-03/opst/worker [get]
func GetWorkerList(c *gin.Context) {
	resp, err := managerIo.ListWorkers()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp)
}

// @Summary get worker detail
// @Description get worker detail
// @Tags opst/worker
// @Accept  json
// @Produce  json
// @Param id path string true "worker id"
// @Success 200 {object} model.WorkerResponse
// @Failure 500 {object} string
// @Router /v2023-03/opst/worker/{id} [get]
func GetWorkerDetail(c *gin.Context) {
	resp, err := managerIo.QueryWorker(model.WorkerInput{
		WorkerID: c.Param("id"),
	})
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp)
}

// @Summary create worker
// @Description 带上cloud region 注册worker
// @Tags opst/worker
// @Accept  json
// @Produce  json
// @Param worker body model.WorkerRequest true "worker"
// @Success 200 {object} model.WorkerResponse
// @Failure 500 {object} string
// @Router /v2023-03/opst/worker [post]
func CreateWorker(c *gin.Context) {
	var req model.WorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	resp, err := managerIo.CreateWorker(req.Cloud, req.Region)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, resp)
}

// @Summary update worker
// @Description 只用来更新worker的hc状态
// @Tags opst/worker
// @Accept  json
// @Produce  json
// @Param id path string true "worker id"
// @Success 200 {object} model.WorkerResponse
// @Failure 500 {object} string
// @Router /v2023-03/opst/worker/{id} [put]
func UpdateWorker(c *gin.Context) {
	err := managerIo.UpdateWorker(c.Param("id"))
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, "success update worker hc")
}
