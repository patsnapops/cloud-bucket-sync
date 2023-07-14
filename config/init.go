package config

import (
	"cbs/pkg/model"

	dt "github.com/patsnapops/go-dingtalk-sdk-wrapper"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/cast"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(apiConfig ManagerConfig, debug bool) *gorm.DB {
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
	log.Infof("connect db %s:%s", apiConfig.PG.Host, apiConfig.PG.Database)
	rdb.AutoMigrate(
		model.Task{}, model.Record{}, model.Worker{},
	)
	return rdb
}

func InitDt(managerConfig ManagerConfig, withDingtalkApprove bool) *dt.DingTalkClient {
	config := dt.DingTalkConfig{
		AppKey:    managerConfig.Dingtalk.AppKey,
		AppSecret: managerConfig.Dingtalk.AppSecret,
		CorpId:    managerConfig.Dingtalk.CorpId,
		AgentId:   managerConfig.Dingtalk.AgentId,
	}
	client, err := dt.NewDingTalkClient(&config)
	if err != nil {
		panic(err)
	}
	client.WithRobotClient()
	if withDingtalkApprove {
		client.WithWorkflowClientV2() // enable 工作流审批
	}
	client.WithMiniProgramClient(cast.ToInt64(config.AgentId)) // enable 小程序通知等
	return client
}
