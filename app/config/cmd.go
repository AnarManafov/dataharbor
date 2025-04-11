package config

import (
	"flag"
	"fmt"
)

// InitCmd initializes and parses command-line arguments
func InitCmd() {
	// Set up the command-line flag for config file path
	configPathPtr := flag.String("config", "./config/application.yaml", "Path to config file")

	// Parse command-line flags
	flag.Parse()

	// Set the global ConfigFile path from the command-line argument
	if configPathPtr != nil {
		ConfigFile = *configPathPtr
		fmt.Printf("Using config file from command line: %s\n", ConfigFile)
	}
}
