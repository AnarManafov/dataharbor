package common

import "github.com/spf13/viper"

type XrdConfigType struct {
	Host                  string
	Port                  uint
	InitialDir            string
	XrdClientBinPath      string
	ProcessTimeout        uint
	StagingPath           string
	SanitationJobInterval uint
	StagingTmpDirPrefix   string
}

var XrdConfig XrdConfigType

func ParseXrdConfig() {
	viper.SetDefault("xrd.host", "localhost")
	viper.SetDefault("xrd.port", 1094)
	viper.SetDefault("xrd.initial_dir", "/tmp/")
	viper.SetDefault("xrd.xrd_client_bin_path", "/opt/homebrew/bin/")
	viper.SetDefault("xrd.process_timeout", 60)
	viper.SetDefault("xrd.staging_path", "/tmp/delete_me")
	viper.SetDefault("xrd.sanitation_job_interval", 30)
	viper.SetDefault("xrd.staging_tmp_dir_prefix", "stg_")

	XrdConfig.Host = viper.GetString("xrd.host")
	XrdConfig.Port = viper.GetUint("xrd.port")
	XrdConfig.InitialDir = viper.GetString("xrd.initial_dir")
	XrdConfig.XrdClientBinPath = viper.GetString("xrd.xrd_client_bin_path")
	XrdConfig.ProcessTimeout = viper.GetUint("xrd.process_timeout")
	XrdConfig.StagingPath = viper.GetString("xrd.staging_path")
	XrdConfig.SanitationJobInterval = viper.GetUint("xrd.sanitation_job_interval")
	XrdConfig.StagingTmpDirPrefix = viper.GetString("xrd.staging_tmp_dir_prefix")
}
