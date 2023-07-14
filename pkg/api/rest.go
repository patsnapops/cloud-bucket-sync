package api

import (
	"cbs/config"
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/noop/log"
)

var (
	managerIo       model.ManagerIo
	managerContract model.ManagerContract
	managerConfig   config.ManagerConfig
)

func ApplyRoutes(routerGroup *gin.RouterGroup, managerio model.ManagerIo, managerconfig config.ManagerConfig, mc model.ManagerContract) {
	// 注册managerio
	managerIo = managerio
	managerConfig = managerconfig
	managerContract = mc
	if managerIo == nil {
		log.Panic("managerIo is nil")
	}
	log.Debugf("managerIo: %v", managerIo)
	// worker
	w := routerGroup.Group("/worker")
	{
		// worker需要注册，更新，查询
		w.GET("", GetWorkerList)
		w.GET("/:id", GetWorkerDetail)
		w.POST("", CreateWorker)
		w.PUT("/:id", UpdateWorker)
	}
	// task
	t := routerGroup.Group("/task")
	{
		t.GET("", GetTaskList)
		t.GET("/:id", GetTaskDetail)
		t.POST("", CreateTask)
		t.PUT("/:id", UpdateTask)
		t.DELETE("/:id", DeleteTask)
	}
	// record
	r1 := routerGroup.Group("/record")
	{
		r1.GET("", GetTaskRecordList)
		r1.GET("/:id", GetTaskRecordDetail)
		r1.PUT("/:id", UpdateTaskRecord)
	}
	// action
	a := routerGroup.Group("/action")
	{
		a.POST("", ChangeRecordStatus)
	}
	// execute
	e := routerGroup.Group("/execute")
	{
		e.POST("", ExecuteTask)
	}
	// dingtalk webhook
	{
		routerGroup.POST("/webhook", DingTalkWebHook)
	}

}

type WebhookRequest struct {
	ProcessInstanceId string `json:"processInstanceId" binding:"required"`
	Result            string `json:"result" binding:"required"`
}

// @Summary https://github.com/patsnapops/dingtalk_miniprogram_webhook的回调接口
// @Description 用于接收钉钉机器人的回调
// @Tags webhook
// @Accept  json
// @Produce  json
// @Param webhook body WebhookRequest true "webhook"
// @Success 200 {string} string	"ok"
// @Router /api/v1/webhook [post]
func DingTalkWebHook(c *gin.Context) {
	var req WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %v", err)
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	log.Debugf("req: %v", req)
	tasks, err := managerIo.QueryTask(model.TaskInput{
		InstanceId: req.ProcessInstanceId,
	})
	if err != nil {
		log.Errorf("query task error: %v", err)
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	if len(tasks) > 1 {
		c.JSON(500, "more than one task matched, please check")
		return
	}
	if len(tasks) == 0 {
		c.JSON(500, "no task matched, please check")
		return
	}
	task := tasks[0]
	task.ApproveResult = req.Result
	if managerIo.UpdateTask(task) != nil {
		log.Errorf("update task error: %v", err)
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	log.Infof("update task %s approve result to %s success", task.Id, req.Result)
	c.JSON(200, "ok")
}
