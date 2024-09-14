package controller

import (
	"errors"
	"net/http"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/request"
	"github.com/AnarManafov/data_lake_ui/app/response"

	"github.com/gin-gonic/gin"
)

// ReadDirFunc is a function type that reads the directory and returns the list of files.
// This function definition is used for real and mock implementations.
type ReadDirFunc func(ctx *gin.Context, host string, port uint, dir string) ([]xrdDirEntry, error)

// TODO: Move the default value to the configuration
var pageSize uint32 = 500 // Default page size (a number of items per page)
const minPageSize = 100

func GetInitialDir(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.InitialDir)
}

func GetHostName(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.Host)
}

func GetDirItems(ctx *gin.Context) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("empty directory path to list")))
		return
	}

	files, err := ReadDir(ctx, common.XrdConfig.Host, common.XrdConfig.Port, dirPath)
	if err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	pageSizeTmp := req.PageSize
	if pageSizeTmp < minPageSize {
		pageSize = minPageSize
	} else {
		pageSize = pageSizeTmp
	}

	totalItems := uint32(len(files))
	totalPages := (totalItems + pageSize - 1) / pageSize // Calculate total pages

	common.Debugf(ctx, "Total Items: %d; Total Pages: %d\n", totalItems, totalPages)

	// Get the first page of items
	startIndex := 0
	endIndex := min(pageSize, totalItems)

	var items []response.DirItemResp
	var totalFileCount, totalFolderCount, cumulativeFileSize uint64

	for _, d := range files[startIndex:endIndex] {
		item := response.DirItemResp{
			Name:     d.name,
			DateTime: d.dt.Format("2006-01-02 15:04:05"),
			Size:     d.size,
		}
		if d.isDir {
			totalFolderCount++
			item.Type = "dir"
		} else {
			item.Type = "file"
			totalFileCount++
			cumulativeFileSize += d.size
		}

		items = append(items, item)
	}

	// This second loop is used to calculate the total number of files, folders, and the cumulative file size.
	for i := endIndex; i < totalItems; i++ {
		d := files[i]
		if d.isDir {
			totalFolderCount++
		} else {
			totalFileCount++
			cumulativeFileSize += d.size
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":               200,
		"items":              items,
		"totalItems":         totalItems,
		"pageSize":           pageSize,
		"totalPages":         totalPages,
		"totalFileCount":     totalFileCount,
		"totalFolderCount":   totalFolderCount,
		"cumulativeFileSize": cumulativeFileSize,
	})
}

func GetDirItemsByPage(ctx *gin.Context) {
	_GetDirItemsByPage(ctx, ReadDir, common.XrdConfig.Host, common.XrdConfig.Port)
}

func _GetDirItemsByPage(ctx *gin.Context, readDir ReadDirFunc, host string, port uint) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	page := req.Page
	if page < 1 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("invalid page number")))
		return
	}

	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("empty directory path to list")))
		return
	}

	files, err := readDir(ctx, host, port, dirPath)
	if err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	totalItems := uint32(len(files))
	totalPages := (totalItems + pageSize - 1) / pageSize // Calculate total pages

	common.Debugf(ctx, "Total Items: %d; Total Pages: %d", totalItems, totalPages)

	if page > uint32(totalPages) {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("page number out of range")))
		return
	}

	startIndex := (page - 1) * pageSize
	endIndex := min(startIndex+pageSize, uint32(totalItems))

	var items []response.DirItemResp
	for _, d := range files[startIndex:endIndex] {
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

	ctx.JSON(http.StatusOK, gin.H{
		"code":  200,
		"items": items,
	})
}

func GetFileStagedForDownload(ctx *gin.Context) {
	var req request.DirItemsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	filePath := req.Path
	if len(filePath) == 0 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("empty file path for staging")))
		return
	}

	// Stage the requested file:
	// Ask XRD to copy the requested file to the Server's public location, so that it can be downloaded.
	stagedFilePath, err := StageFile(common.XrdConfig.Host, common.XrdConfig.Port, filePath)
	if err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	// Send the response the to the server with the public location of the requested file.
	respondData := response.StageFileResp{Path: stagedFilePath}

	response.Success(ctx, respondData)
}
