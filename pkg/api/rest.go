package api

import (
	"cbs/config"
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/noop/log"
)

var (
	managerIo     model.ManagerIo
	managerConfig config.ManagerConfig
)

func ApplyRoutes(routerGroup *gin.RouterGroup, managerio model.ManagerIo, managerconfig config.ManagerConfig) {
	// 注册managerio
	managerIo = managerio
	managerConfig = managerconfig
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

}
