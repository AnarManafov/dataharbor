package controller

import (
	"github.com/AnarManafov/app/common"
	"github.com/AnarManafov/app/request"
	"github.com/AnarManafov/app/response"

	"github.com/gin-gonic/gin"
)

func GetInitialDir(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.InitialDir)
}

func GetHostName(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.Host)
}

func GetDirItems(ctx *gin.Context) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, response.SystemErr.AppendErrMsg(err))
		return
	}

	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(ctx, response.SystemErr.Append("Empty directory path to list."))
		return
	}

	var items []response.DirItemResp

	files, err := ReadDir(common.XrdConfig.Host, common.XrdConfig.Port, dirPath)
	if err != nil {
		response.FailWithErr(ctx, response.SystemErr.AppendErrMsg(err))
		return
	}

	for _, d := range files {
		item := response.DirItemResp{
			Name:     d.name,
			DateTime: d.dt.Format("2006-01-02 15:04:05"),
			Size:     d.size,
		}
		if d.isDir { // If you want to list both files and dirs, remove this check.
			item.Type = "dir"
		} else {
			item.Type = "file"
		}

		items = append(items, item)
	}

	response.Success(ctx, items)
}

func GetFileStagedForDownload(ctx *gin.Context) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, response.SystemErr.AppendErrMsg(err))
		return
	}

	filePath := req.Path
	if len(filePath) == 0 {
		response.FailWithErr(ctx, response.SystemErr.Append("Empty file path for staging"))
		return
	}

	// Stage the requested file:
	// Ask XRD to copy the requested file to the Server's public location, so that it can be downloaded.
	stagedFilePath, err := StageFile(common.XrdConfig.Host, common.XrdConfig.Port, filePath)
	if err != nil {
		response.FailWithErr(ctx, response.SystemErr.AppendErrMsg(err))
		return
	}

	// Send the response the to the server with the public location of the requested file.
	respondData := response.StageFileResp{Path: stagedFilePath}

	response.Success(ctx, respondData)
}
