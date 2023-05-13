package config

import (
	"fmt"

	"github.com/patsnapops/noop/log"
	"github.com/spf13/viper"
)

type ApiConfig struct {
	Host string `mapstructure:"host"`
}
type CliConfig struct {
	Host string `mapstructure:"host"`
	Cli  string `mapstructure:"cli"`
}
type WorkerConfig struct {
	Worker string `mapstructure:"worker"`
}

func loadConfig(configFile string) error {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file error: %s", err)
	}
	return nil
}

func LoadApiConfig(configDir string) *ApiConfig {
	log.Debugf("config dir %s", configDir)
	apiConfig := &ApiConfig{}
	err := loadConfig(configDir + "api.yaml")
	if err != nil {
		log.Fatalf(err.Error())
	}
	viper.Unmarshal(apiConfig)
	return apiConfig
}

func LoadCliConfig(configDir string) *CliConfig {
	cliConfig := &CliConfig{}
	err := loadConfig(configDir + "cli.yaml")
	if err != nil {
		log.Fatalf(err.Error())
	}
	viper.Unmarshal(cliConfig)
	return cliConfig
}

func LoadWorkerConfig(configDir string) *WorkerConfig {
	workerConfig := &WorkerConfig{}
	err := loadConfig(configDir + "worker.yaml")
	if err != nil {
		log.Fatalf(err.Error())
	}
	viper.Unmarshal(workerConfig)
	return workerConfig
}
