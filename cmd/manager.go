package cmd

import (
	"cbs/config"
	_ "cbs/docs"
	"cbs/pkg/api"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"

	"github.com/gin-gonic/gin"
	"github.com/patsnapops/ginx/middleware"
	hh "github.com/patsnapops/http-headers"
	"github.com/patsnapops/noop/log"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	port                string
	withOutSchedule     bool
	withDingtalkApprove bool // 任务是否开启钉钉审批
)

var (
	managerIo model.ManagerIo
	managerC  model.ManagerContract
)

func init() {
	apiServerCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&port, "port", "p", "8012", "指定端口号(默认8012))")
	startCmd.Flags().BoolVarP(&withOutSchedule, "without-schedule", "", false, "是否禁用定时任务(默认不禁用)")
	startCmd.Flags().BoolVarP(&withDingtalkApprove, "with-dingtalk-approve", "", false, "任务是否开启钉钉审批(默认不开启)")
}

var apiServerCmd = &cobra.Command{
	Use:     "manager",
	Aliases: []string{"m"},
	Long:    "this is manager server",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:  "start",
	Long: "start manager server, default port is 8012",
	Run: func(cmd *cobra.Command, args []string) {
		initConfig()
		managerConfig = config.LoadManagerConfig(configPath)
		log.Infof("withDingtalkApprove: %v", withDingtalkApprove)
		managerIo = io.NewManagerClient(config.InitDB(*managerConfig, debug))
		dtc := io.NewDingtalkClient(config.InitDt(*managerConfig, withDingtalkApprove), managerConfig.Dingtalk)
		managerC = service.NewManagerService(managerIo, dtc, withDingtalkApprove)
		go startSchedule(managerC)
		startGin()
	},
}

func startGin() {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	ginEngine := gin.Default()
	// TODO JWT
	middleware.AttachTo(ginEngine).
		WithCacheDisabled().
		WithCORS().
		WithRecover().
		WithRequestID(hh.XRequestID).
		WithSecurity()
	// add swagger
	ginEngine.GET("/swagger/*any", func(c *gin.Context) {
		c.Next()
	}, ginSwagger.WrapHandler(swaggerFiles.Handler))

	r := ginEngine.Group("/api/v1")
	api.ApplyRoutes(r, managerIo, *managerConfig, managerC)
	log.Infof("manager server start at 0.0.0.0:%s success", port)
	err := ginEngine.Run(":" + port)
	if err != nil {
		log.Panicf("gin start failed %s", err.Error())
	}
}

func startSchedule(managerC model.ManagerContract) {
	if withOutSchedule {
		log.Infof("schedule is disabled.")
		return
	}
	c := cron.New()
	c.AddFunc("*/30 * * * * *", func() {
		// 检查worker状态
		managerC.CheckWorker()
	})
	c.AddFunc("1 * * * * *", func() {
		// 检查task的cron表达式，符合条件的task会执行生成pending状态record去跑
		managerC.CheckTaskCorn()
	})
	c.Start()
	select {}
}
