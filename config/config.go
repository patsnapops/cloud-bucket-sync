package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/patsnapops/noop/log"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type ApiConfig struct {
	PG PostgresConfig `mapstructure:"pg"`
}
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

func (c *PostgresConfig) GetUrl() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		c.Host,
		c.User,
		c.Password,
		c.Database,
		cast.ToString(c.Port),
	)
}

type CliConfig struct {
	Host     string    `mapstructure:"host"`
	Cli      string    `mapstructure:"cli"`
	Profiles []Profile `mapstructure:"profiles"`
}

func GetProfile(profiles []Profile, name string) Profile {
	for _, profile := range profiles {
		if profile.Name == name {
			return profile
		}
	}
	return Profile{}
}

type WorkerConfig struct {
	Worker string `mapstructure:"worker"`
}

type Profile struct {
	Name     string `mapstructure:"name"`
	AK       string `mapstructure:"ak"`
	SK       string `mapstructure:"sk"`
	Region   string `mapstructure:"region"`
	Endpoint string `mapstructure:"endpoint"`
}

func loadConfig(configFile string) error {
	homedir := os.Getenv("HOME")
	if strings.Contains(configFile, "~") {
		configFile = strings.Replace(configFile, "~", homedir, 1)
	}
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file error: %s", err)
	}
	return nil
}

func LoadApiConfig(configDir string) *ApiConfig {
	if !strings.HasSuffix(configDir, "/") {
		configDir = configDir + "/"
	}
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
	if !strings.HasSuffix(configDir, "/") {
		configDir = configDir + "/"
	}
	err := loadConfig(configDir + "manager.yaml")
	if err != nil {
		panic(err)
	}
	viper.Unmarshal(cliConfig)
	return cliConfig
}

func LoadWorkerConfig(configDir string) *WorkerConfig {
	workerConfig := &WorkerConfig{}
	if !strings.HasSuffix(configDir, "/") {
		configDir = configDir + "/"
	}
	err := loadConfig(configDir + "worker.yaml")
	if err != nil {
		log.Fatalf(err.Error())
	}
	viper.Unmarshal(workerConfig)
	return workerConfig
}
