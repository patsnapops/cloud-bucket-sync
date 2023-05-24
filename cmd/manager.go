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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	port            string
	disableSchedule bool
)

func init() {
	apiServerCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&port, "port", "p", "8080", "manager server port")
	startCmd.Flags().BoolVarP(&disableSchedule, "disable-schedule", "", false, "disable schedule")
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
	Long: "start manager server, default port is 8080",
	Run: func(cmd *cobra.Command, args []string) {
		initApp()
		managerIo := io.NewManagerClient(initDB(*managerConfig))
		managerC := service.NewManagerService(managerIo)
		go startSchedule(managerC)
		startGin(managerIo)
	},
}

func startGin(managerIo model.ManagerIo) {
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
	api.ApplyRoutes(r, managerIo, *managerConfig)
	log.Infof("manager server start at 0.0.0.0:%s success", port)
	err := ginEngine.Run(":" + port)
	if err != nil {
		log.Panicf("gin start failed %s", err.Error())
	}
}

func startSchedule(managerC model.ManagerContract) {
	if disableSchedule {
		log.Infof("schedule is disabled.")
		return
	}
	c := cron.New()
	c.AddFunc("*/10 * * * * *", func() {
		managerC.CheckWorker()
	})
	c.Start()
	select {}
}

func initDB(apiConfig config.ManagerConfig) *gorm.DB {
	dbConfig := &gorm.Config{}
	if !debug {
		dbConfig = &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		}
	}
	rdb, err := gorm.Open(postgres.Open(apiConfig.PG.GetUrl()), dbConfig)
	if err != nil {
		log.Panicf("gorm with postgres initialization failed, check config.PostgresUrl")
	}
	rdb.AutoMigrate(
		model.Task{}, model.Record{}, model.Worker{},
	)
	return rdb
}
