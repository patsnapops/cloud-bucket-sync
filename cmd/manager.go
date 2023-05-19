package cmd

import (
	"cbs/config"
	"cbs/pkg/api"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/patsnapops/ginx/middleware"
	hh "github.com/patsnapops/http-headers"
	"github.com/patsnapops/noop/log"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	port string
)

func init() {
	apiServerCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&port, "port", "p", "8080", "manager server port")
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
		log.Debugf(tea.Prettify(managerConfig))
		managerIo := io.NewManagerClient(initDB(*managerConfig))
		managerC := service.NewManagerService(managerIo)
		go startGin(managerIo)
		startSchedule(managerC)
	},
}

func startGin(managerIo model.ManagerIo) {
	ginEngine := gin.Default()
	// TODO JWT
	middleware.AttachTo(ginEngine).
		WithCacheDisabled().
		WithCORS().
		WithRecover().
		WithRequestID(hh.XRequestID).
		WithSecurity()
	r := ginEngine.Group("/api/v1")
	api.ApplyRoutes(r, managerIo)

	if !debug {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	err := ginEngine.Run(":" + port)
	if err != nil {
		log.Panicf("gin start failed %s", err.Error())
	}
}

func startSchedule(managerC model.ManagerContract) {
	c := cron.New()
	c.AddFunc("*/10 * * * * *", func() {
		managerC.CheckWorker()
	})
	c.Start()
	log.Infof("manager server start at success")
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
