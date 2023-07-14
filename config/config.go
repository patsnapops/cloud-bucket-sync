package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/patsnapops/noop/log"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type ManagerConfig struct {
	PG       PostgresConfig `mapstructure:"pg"`
	Dingtalk DingtalkConfig `mapstructure:"dingtalk"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type DingtalkConfig struct {
	RobotToken  string      `mapstructure:"robot_token"`
	AppKey      string      `mapstructure:"app_key"`
	AppSecret   string      `mapstructure:"app_secret"`
	CorpId      string      `mapstructure:"corp_id"`
	AgentId     string      `mapstructure:"agent_id"`
	ApproveInfo ApproveInfo `mapstructure:"approve_info"`
}

type ApproveInfo struct {
	ProcessCode string `mapstructure:"process_code"`
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
	Manager  CliManager `mapstructure:"manager"`
	Profiles []Profile  `mapstructure:"profiles"`
}

type CliManager struct {
	Endpoint   string `mapstructure:"endpoint"`
	ApiVersion string `mapstructure:"api_version"`
}

func GetProfile(profiles []Profile, name string) Profile {
	for _, profile := range profiles {
		if profile.Name == name {
			return profile
		}
	}
	return Profile{}
}

type Profile struct {
	Name     string `mapstructure:"name"`
	AK       string `mapstructure:"ak"`
	SK       string `mapstructure:"sk"`
	Region   string `mapstructure:"region"`   // 桶所在的区域
	Endpoint string `mapstructure:"endpoint"` // 云服务的endpoint，支持配置自己的endpoint
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

func LoadManagerConfig(configDir string) *ManagerConfig {
	if !strings.HasSuffix(configDir, "/") {
		configDir = configDir + "/"
	}
	apiConfig := &ManagerConfig{}
	err := loadConfig(configDir + "manager.yaml")
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
	err := loadConfig(configDir + "cli.yaml")
	if err != nil {
		panic(err)
	}
	viper.Unmarshal(cliConfig)
	return cliConfig
}
