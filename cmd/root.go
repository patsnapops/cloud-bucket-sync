package cmd

import (
	"cbs/config"
	"cbs/pkg/io"
	"cbs/pkg/model"
	"cbs/pkg/service"
	"time"

	"github.com/patsnapops/noop/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	debug      bool
	configPath string
	logPath    string

	cliConfig     *config.CliConfig
	managerConfig *config.ManagerConfig
	workerConfig  *config.WorkerConfig

	requestC model.RequestContract
	bucketC  model.BucketContract
)

func init() {
	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(bucketCmd)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "~/.cbs/", "config file dir,default is ~/.cbs/")
	rootCmd.PersistentFlags().StringVarP(&logPath, "log", "", "./cbs.log", "log file dir,default is ./cbs.log")
}

var (
	rootCmd = &cobra.Command{
		Use:   "cbs",
		Short: "Welcome to use cbs",
		Long:  "CloudBucketSync is a powerful tool for syncing data between cloud storage buckets. Built in Go.\nyou should use `cbs [command] --help` to see the usage of each command.\n",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		Version: "v0.0.1-beta",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func initApp() {
	logLevel := log.InfoLevel
	if debug {
		logLevel = log.DebugLevel
	}
	// init log
	log.Default().WithLevel(logLevel).WithFilename(logPath).WithFields([]zap.Field{{
		Key:    "LocalTime",
		Type:   zapcore.StringType,
		String: time.Now().In(time.Local).Format("2006-01-02 15:04:05.000"),
	}}).Init()
	// init config
	cliConfig = config.LoadCliConfig(configPath)
	managerConfig = config.LoadManagerConfig(configPath)
	workerConfig = config.LoadWorkerConfig(configPath)
	// log.Debugf(tea.Prettify(workerConfig))
	// init Service
	requestC = service.NewRequestService(cliConfig.Manager)
	bucketC = service.NewBucketService(io.NewBucketClient(cliConfig.Profiles))
	log.Debugf("init app success")
}
