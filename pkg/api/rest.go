package api

import (
	"cbs/pkg/model"

	"github.com/gin-gonic/gin"
)

var (
	managerIo model.ManagerIo
)

func ApplyRoutes(routerGroup *gin.RouterGroup, managerIo model.ManagerIo) {
	managerIo = managerIo
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
	a := routerGroup.Group("/status")
	{
		a.POST("", ChangeRecordStatus)
	}
	// execute
	e := routerGroup.Group("/execute")
	{
		e.POST("", ExecuteTask)
	}

}
