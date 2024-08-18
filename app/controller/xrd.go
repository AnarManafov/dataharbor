package controller

import (
	"github.com/AnarManafov/app/common"

	"bufio"
	"context"
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

func RunXrdfs(arg ...string) (string, error) {
	timeout := common.XrdConfig.ProcessTimeout

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, common.XrdConfig.XrdfsPath, arg...)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// TODO: add error code
func ReadDir(host string, port int, dir string) (retVal []xrdDirEntry, err error) {
	srd_addr := host + ":" + strconv.Itoa(port)
	output, err := RunXrdfs(srd_addr, "ls", "-l", dir)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		columns := strings.Fields(scanner.Text())
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
