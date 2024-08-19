package common

import "github.com/spf13/viper"

type XrdConfigType struct {
	Host             string
	Port             int
	InitialDir       string
	XrdClientBinPath string
	ProcessTimeout   int
	StagingPath      string
}

var XrdConfig XrdConfigType

func ParseXrdConfig() {
	viper.SetDefault("xrd.host", "localhost")
	viper.SetDefault("xrd.port", 1094)
	viper.SetDefault("xrd.initial_dir", "/tmp/")
	viper.SetDefault("xrd.xrd_client_bin_path", "/opt/homebrew/bin/")
	viper.SetDefault("xrd.process_timeout", 5)
	viper.SetDefault("xrd.staging_path", "/tmp/delete_me")

	XrdConfig.Host = viper.GetString("xrd.host")
	XrdConfig.Port = viper.GetInt("xrd.port")
	XrdConfig.InitialDir = viper.GetString("xrd.initial_dir")
	XrdConfig.XrdClientBinPath = viper.GetString("xrd.xrd_client_bin_path")
	XrdConfig.ProcessTimeout = viper.GetInt("xrd.process_timeout")
	XrdConfig.StagingPath = viper.GetString("xrd.staging_path")
}
