package config

import (
	"flag"
	"fmt"
)

var (
	// Version of the application (set via -ldflags at build time)
	// Example: go build -ldflags="-X github.com/AnarManafov/dataharbor/app/config.Version=1.0.0"
	Version = "dev"

	// BuildTime of the application (set via -ldflags at build time)
	BuildTime = "unknown"

	// GitCommit hash (set via -ldflags at build time)
	GitCommit = "unknown"
)

// InitCmd initializes and parses command-line arguments
// Returns true if the application should continue, false if it should exit
func InitCmd() bool {
	// Set up command-line flags
	configPathPtr := flag.String("config", "./config/application.yaml", "Path to config file")
	versionFlag := flag.Bool("version", false, "Print version information")

	// Parse command-line flags
	flag.Parse()

	// Handle --version flag
	if *versionFlag {
		fmt.Printf("dataharbor-backend version %s\n", Version)
		if BuildTime != "unknown" {
			fmt.Printf("Build time: %s\n", BuildTime)
		}
		if GitCommit != "unknown" {
			fmt.Printf("Git commit: %s\n", GitCommit)
		}
		return false
	}

	// Set the global ConfigFile path from the command-line argument
	if configPathPtr != nil {
		ConfigFile = *configPathPtr
		fmt.Printf("Using config file from command line: %s\n", ConfigFile)
	}

	return true
}
