package config

import (
	"flag"
	"fmt"
	"os"
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
	return initCmd(flag.CommandLine, os.Args[1:])
}

// initCmd registers the backend's flags on fs and parses args. It is separate
// from InitCmd so tests can use a private FlagSet instead of replacing the
// process-wide flag.CommandLine (which also holds the test binary's -test.*
// flags; swapping it out breaks `go test -count=N`).
func initCmd(fs *flag.FlagSet, args []string) bool {
	// Set up command-line flags
	configPathPtr := fs.String("config", "./config/application.yaml", "Path to config file")
	versionFlag := fs.Bool("version", false, "Print version information")

	// Parse command-line flags. With flag.ExitOnError (as flag.CommandLine
	// uses) a parse error exits the process, so the error is only seen by
	// callers that pass a ContinueOnError FlagSet.
	if err := fs.Parse(args); err != nil {
		return false
	}

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
