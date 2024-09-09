package controller

import (
	"regexp"

	"github.com/AnarManafov/app/common"

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

func RunXrdFs(arg ...string) (string, error) {
	common.Logger.Info("RunXrdFs: ", arg)
	timeout := common.XrdConfig.ProcessTimeout

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, path.Join(common.XrdConfig.XrdClientBinPath, "xrdfs"), arg...)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func RunXrdCp(_xrd_addr string, _src string, _dest string) error {
	timeout := common.XrdConfig.ProcessTimeout

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	_src = "xroot://" + _xrd_addr + "/" + _src
	common.Logger.Info("XRD: Staging " + _src + " to " + _dest)
	cmd := exec.CommandContext(ctx, path.Join(common.XrdConfig.XrdClientBinPath, "xrdcp"), "--force", _src, _dest)

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func ReadDir(host string, port uint, dir string) (retVal []xrdDirEntry, err error) {
	srd_addr := host + ":" + strconv.FormatUint(uint64(port), 10)
	output, err := RunXrdFs(srd_addr, "ls", "-l", dir)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
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
		if columns[0][0] == 'd' {
			item.isDir = true
		} else {
			item.isDir = false
		}
		// Date/Time
		var tt time.Time
		const layoutTime string = "2006-01-02 15:04:05"
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

	return retVal, nil
}

func StageFile(_host string, _port uint, _file string) (string, error) {
	srd_addr := _host + ":" + strconv.FormatUint(uint64(_port), 10)
	// Create a random subdirectory to allow concurrent download files with the same name.
	tmpDir, err := os.MkdirTemp(common.XrdConfig.StagingPath, common.XrdConfig.StagingTmpDirPrefix)
	if err != nil {
		return "", err
	}
	stagedFilePath := path.Join(tmpDir, path.Base(_file))
	// Request XRD to copy the file from XRD to a local location
	err = RunXrdCp(srd_addr, _file, stagedFilePath)
	if err != nil {
		return "", err
	}

	return stagedFilePath, nil
}
