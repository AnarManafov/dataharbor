package common

import "github.com/spf13/viper"

type XrdConfigType struct {
	Host           string
	Port           int
	InitialDir     string
	XrdfsPath      string
	ProcessTimeout int
}

var XrdConfig XrdConfigType

func ParseXrdConfig() {
	viper.SetDefault("xrd.host", "localhost")
	viper.SetDefault("xrd.port", 1094)
	viper.SetDefault("xrd.initial_dir", "/tmp")
	viper.SetDefault("xrd.xrdfs_path", "/opt/homebrew/bin/xrdfs")
	viper.SetDefault("xrd.process_timeout", 5)

	XrdConfig.Host = viper.GetString("xrd.host")
	XrdConfig.Port = viper.GetInt("xrd.port")
	XrdConfig.InitialDir = viper.GetString("xrd.initial_dir")
	XrdConfig.XrdfsPath = viper.GetString("xrd.xrdfs_path")
	XrdConfig.ProcessTimeout = viper.GetInt("xrd.process_timeout")
}
