package cmd

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/patsnapops/noop/log"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var apiServerCmd = &cobra.Command{
	Use:     "manager",
	Aliases: []string{"m"},
	Long:    "this is manager server",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	apiServerCmd.AddCommand(startCmd)
}

func initDB(apiConfig config.ApiConfig) *gorm.DB {
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

var startCmd = &cobra.Command{
	Use:  "start",
	Long: "start manager server, default port is 8080",
	Run: func(cmd *cobra.Command, args []string) {
		if debug {
			log.Default().WithLevel(log.DebugLevel).WithFilename("cbs.log").Init()
		} else {
			log.Default().WithLevel(log.InfoLevel).WithFilename("cbs.log").Init()
		}
		apiConfig := config.LoadApiConfig(configPath)
		log.Debugf(tea.Prettify(apiConfig))
		managerClient := io.NewManagerClient(initDB(*apiConfig))
		manager := service.NewManagerService(managerClient)
		c := cron.New()
		c.AddFunc("*/10 * * * * *", func() {
			manager.CheckWorker()
		})
		c.Start()
		log.Infof("manager server start at success")
		select {}
	},
}
