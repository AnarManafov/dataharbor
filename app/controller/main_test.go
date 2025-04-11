package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
)

// Define functions as variables so they can be mocked
var (
	// Mock exec.CommandContext
	mockExecCommand = func(ctx context.Context, command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		// Set this environment variable to identify we're in the mock
		cmd.Env = []string{"GO_HELPER_PROCESS=1"}
		return cmd
	}

	// Original functions that we'll back up and restore
	originalRunXrdFs       RunXrdFsFunc
	originalStageFileLocal func(host string, port uint, file string) (string, error)
)

// TestMain sets up testing environment
func TestMain(m *testing.M) {
	// Initialize the logger and the configuration
	common.InitLogger()
	config.InitCmd()

	// Create a minimal common.XrdConfig for testing
	common.XrdConfig = common.XrdConfigType{
		Host:                "localhost",
		Port:                1094,
		InitialDir:          "/tmp/",
		XrdClientBinPath:    "/usr/bin/",
		ProcessTimeout:      1, // Short timeout for tests
		StagingPath:         os.TempDir(),
		StagingTmpDirPrefix: "test_",
	}

	// Backup original functions
	originalRunXrdFs = RunXrdFs
	originalStageFileLocal = stageFileLocal

	// Override functions with test implementations
	RunXrdFs = func(execCmd execCommandFunc, arg ...string) (string, error) {
		if os.Getenv("GO_USE_REAL_COMMANDS") == "1" {
			return originalRunXrdFs(execCmd, arg...)
		}
		// For tests, return mock data based on the args
		if len(arg) > 0 && arg[len(arg)-1] == "/valid/path" {
			return "drwxr-xr-x root root 0 2025-04-04 10:00:00 /valid/path/file1\ndrwxr-xr-x root root 0 2025-04-04 10:00:00 /valid/path/dir1", nil
		}
		return "", fmt.Errorf("mock command failed")
	}

	// Mock stageFileLocal function
	stageFileLocal = func(host string, port uint, file string) (string, error) {
		if os.Getenv("GO_USE_REAL_COMMANDS") == "1" {
			return originalStageFileLocal(host, port, file)
		}
		// Mock responses
		if file == "/valid/file" {
			return "/staged/file", nil
		}
		return "", fmt.Errorf("staging error")
	}

	// Run the tests
	code := m.Run()

	// Restore the original functions
	RunXrdFs = originalRunXrdFs
	stageFileLocal = originalStageFileLocal

	os.Exit(code)
}

// TestHelperProcess isn't a real test - it's a helper process for mocking exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_PROCESS") != "1" {
		return
	}
	// Get the command arguments after "--"
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		// No command to run
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(1)
	}

	// Mock different commands
	cmd, args := args[0], args[1:]
	switch cmd {
	case "/usr/bin/xrdfs":
		// Mock xrdfs command
		if len(args) >= 3 && args[1] == "ls" && args[2] == "/valid/path" {
			fmt.Println("drwxr-xr-x root root 0 2025-04-04 10:00:00 /valid/path/file1")
			fmt.Println("drwxr-xr-x root root 0 2025-04-04 10:00:00 /valid/path/dir1")
			os.Exit(0)
		}
	case "/usr/bin/xrdcp":
		// Mock xrdcp command
		if args[0] == "--force" && args[1] == "xroot://localhost:1094//valid/file" {
			os.Exit(0) // Successfully copied
		}
		// For error case
		if args[0] == "--force" && args[1] == "xroot://localhost:1094//error/file" {
			os.Exit(1) // Error copying
		}
	}

	// Default: command not recognized
	fmt.Fprintf(os.Stderr, "Command not recognized: %s %v\n", cmd, args)
	os.Exit(1)
}
