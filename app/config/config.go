package config

import (
	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/spf13/viper"
)

// ConfigFile is the path to the configuration file
var ConfigFile string

func Init() {
	loadConfig()
	common.ParseSystemConfig()
	if common.ServerConfig.Debug {
		common.Logger.Info(viper.AllSettings())
	}

	// Set the default values for the configuration.
	common.ParseDatabaseConfig()
	common.ParseRedisConf()
	common.ParseXrdConfig()
}

func loadConfig() {
	if ConfigFile != "" {
		viper.SetConfigFile(ConfigFile)
		err := viper.ReadInConfig()
		if err != nil {
			// Log the error but continue with defaults
			common.Logger.Warn("Config file not found, using defaults")
		}
	} else {
		// Log the absence of a config file but continue with defaults
		common.Logger.Warn("No config file provided, using defaults")
	}
}
