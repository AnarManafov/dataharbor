package controller

import (
	"regexp"
	"sync"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/gin-gonic/gin"

	"bufio"
	"context"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

type xrdDirEntry struct {
	name  string
	dt    time.Time
	size  uint64
	isDir bool
}

type cacheEntry struct {
	data      []xrdDirEntry
	timestamp time.Time
}

var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.Mutex

	// TODO: Add "60" to the configuration
	cacheTTL = 60 * time.Minute // Cache Time-To-Live
)

// Define a type for the command execution function
type execCommandFunc func(ctx context.Context, name string, arg ...string) *exec.Cmd

type RunXrdFsFunc func(execCmd execCommandFunc, arg ...string) (string, error)

func getCachedData(key string) ([]xrdDirEntry, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	entry, exists := cache[key]
	if !exists || time.Since(entry.timestamp) > cacheTTL {
		return nil, false
	}
	return entry.data, true
}

func setCachedData(key string, data []xrdDirEntry) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

func runCommand(execCmd execCommandFunc, timeout uint, name string, arg ...string) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := execCmd(ctx, name, arg...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func RunXrdFs(execCmd execCommandFunc, arg ...string) (string, error) {
	common.Logger.Info("RunXrdFs: ", arg)
	return runCommand(execCmd, common.XrdConfig.ProcessTimeout, path.Join(common.XrdConfig.XrdClientBinPath, "xrdfs"), arg...)
}

func RunXrdCp(xrdAddr string, src string, dest string) error {
	common.Logger.Info("XRD: Staging " + src + " to " + dest)
	_, err := runCommand(exec.CommandContext, common.XrdConfig.ProcessTimeout, path.Join(common.XrdConfig.XrdClientBinPath, "xrdcp"), "--force", "xroot://"+xrdAddr+"/"+src, dest)
	return err
}

func ReadDir(ctx *gin.Context, xrdFS RunXrdFsFunc, host string, port uint, dir string) ([]xrdDirEntry, error) {
	srdAddr := host + ":" + strconv.FormatUint(uint64(port), 10)
	cacheKey := srdAddr + ":" + dir

	// Check cache
	if data, found := getCachedData(cacheKey); found {
		return data, nil
	}

	// Run command and parse output
	output, err := xrdFS(exec.CommandContext, srdAddr, "ls", "-l", dir)
	if err != nil {
		return nil, err
	}

	var retVal []xrdDirEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		// Split the input on substrings:
		// Input format:
		// "drwxr-xr-x username staff   224 2024-02-14 12:14:47 /Users/Virtual Machines.localized"
		//
		pattern := `\s+`
		regex := regexp.MustCompile(pattern)
		// The input is split on 7 substrings
		columns := regex.Split(scanner.Text(), 7)

		var item xrdDirEntry
		// File name
		item.name = path.Base(columns[6])
		// Is Dir
		item.isDir = columns[0][0] == 'd'
		// Date/Time
		const layoutTime = "2006-01-02 15:04:05"
		tt, err := time.Parse(layoutTime, columns[4]+" "+columns[5])
		if err == nil {
			item.dt = tt
		}
		// Size
		s, err := strconv.ParseUint(columns[3], 10, 64)
		if err == nil {
			item.size = s
		}
		retVal = append(retVal, item)
	}

	// Cache the parsed data
	setCachedData(cacheKey, retVal)

	return retVal, nil
}

func StageFile(host string, port uint, file string) (string, error) {
	srdAddr := host + ":" + strconv.FormatUint(uint64(port), 10)
	// Create a random subdirectory to allow concurrent download files with the same name.
	tmpDir, err := os.MkdirTemp(common.XrdConfig.StagingPath, common.XrdConfig.StagingTmpDirPrefix)
	if err != nil {
		return "", err
	}
	stagedFilePath := path.Join(tmpDir, path.Base(file))
	// Request XRD to copy the file from XRD to a local location
	err = RunXrdCp(srdAddr, file, stagedFilePath)
	if err != nil {
		return "", err
	}

	return stagedFilePath, nil
}
