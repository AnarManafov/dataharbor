package config

import (
	"flag"
)

func InitCmd() {
	flag.StringVar(&ConfigFile, "config", "./config/application.yaml", "config file path")
	flag.Parse()
}
