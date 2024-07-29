package controller

import (
	"bufio"
	"context"
	"os/exec"
	"path"
	"strings"
	"time"
)

type xrdDirEntry struct {
	name  string
	isDir bool
}

func RunXrdfs(arg ...string) (string, error) {
	timeout := 5

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "/opt/homebrew/bin/xrdfs", arg...)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// TODO: add error code
func ReadDir(host string, dir string) (retVal []xrdDirEntry, err error) {
	output, err := RunXrdfs(host, "ls", "-l", dir)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		columns := strings.Fields(scanner.Text())
		var item xrdDirEntry
		item.name = path.Base(columns[6])
		if columns[0][0] == 'd' {
			item.isDir = true
		} else {
			item.isDir = false
		}
		retVal = append(retVal, item)
	}

	return retVal, nil
}
