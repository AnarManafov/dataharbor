package controller

import (
	"os"
	"os/user"

	"github.com/AnarManafov/app/request"
	"github.com/AnarManafov/app/response"

	"github.com/gin-gonic/gin"
)

func GetHomeDir(ctx *gin.Context) {
	// Get the current user
	u, err := user.Current()
	if err != nil {
		response.FailWithErr(ctx, response.SystemErr)
		return
	}

	response.Success(ctx, u.HomeDir)
}

func GetDirItems(ctx *gin.Context) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, response.SystemErr)
		return
	}

	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(ctx, response.SystemErr)
		return
	}

	var items []response.DirItemResp

	files, err := os.ReadDir(dirPath)
	if err != nil {
		response.FailWithErr(ctx, response.SystemErr)
		return
	}

	for _, d := range files {
		item := response.DirItemResp{
			Name: d.Name(),
		}
		if d.IsDir() { // If you want to list both files and dirs, remove this check.
			item.Type = "dir"
		} else {
			item.Type = "file"
		}
		items = append(items, item)
	}

	if err != nil {
		response.FailWithErr(ctx, response.SystemErr)
		return
	}

	response.Success(ctx, items)
}
