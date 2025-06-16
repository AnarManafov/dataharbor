package controller

import (
	"errors"
	"net/http"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/request"
	"github.com/AnarManafov/dataharbor/app/response"

	"github.com/gin-gonic/gin"
)

// ReadDirFunc enables dependency injection for unit testing
type ReadDirFunc func(xrdFS RunXrdFsFunc, host string, port uint, dir string) ([]xrdDirEntry, error)
type StageFileFunc func(host string, port uint, filePath string) (string, error)

// Pagination constants to balance performance with user experience
const (
	defaultPageSize uint32 = 500 // Large enough for most views while maintaining performance
	minPageSize     uint32 = 5   // Prevents excessive API calls that could impact server performance
)

func FetchInitialDir(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.InitialDir)
}

func FetchHostName(ctx *gin.Context) {
	response.Success(ctx, common.XrdConfig.Host)
}

func FetchDirItems(ctx *gin.Context) {
	fetchDirItems(ctx, ReadDir, common.XrdConfig.Host, common.XrdConfig.Port, false)
}

func FetchDirItemsByPage(ctx *gin.Context) {
	fetchDirItems(ctx, ReadDir, common.XrdConfig.Host, common.XrdConfig.Port, true)
}

// Handles directory listing with unified business logic for both paginated
// and non-paginated views to ensure consistent behavior
func fetchDirItems(ctx *gin.Context, readDir ReadDirFunc, host string, port uint, paginate bool) {
	var req request.DirectoryItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	// Validate the page number when pagination is enabled
	if paginate && req.Page < 1 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("invalid page number")))
		return
	}

	// Validate the directory path
	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("empty directory path to list")))
		return
	}

	// Fetch the list of files from the requested directory
	files, err := readDir(RunXrdFs, host, port, dirPath)
	if err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}
	common.Debugf(ctx, "Fetched %d items from the directory: %s\n", len(files), dirPath)

	// Enforce minimum page size to prevent performance issues
	pageSize := req.PageSize
	if pageSize < minPageSize {
		pageSize = minPageSize
	}

	totalItems := uint32(len(files))
	totalPages := (totalItems + pageSize - 1) / pageSize // Ceiling division ensures partial pages are counted

	common.Debugf(ctx, "Requested Page: %d; Page size: %d; Total Items: %d; Total Pages: %d\n", req.Page, pageSize, totalItems, totalPages)

	if paginate && req.Page > uint32(totalPages) {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("page number out of range")))
		return
	}

	// Calculate slice boundaries based on pagination settings
	startIndex := uint32(0)
	if paginate {
		startIndex = (req.Page - 1) * pageSize
	}
	endIndex := min(startIndex+pageSize, totalItems)

	var items []response.DirectoryItemResponse
	var totalFileCount, totalFolderCount, cumulativeFileSize uint64

	// Process visible items for the current page
	for _, d := range files[startIndex:endIndex] {
		item := response.DirectoryItemResponse{
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

	// Calculate totals for non-visible items to provide accurate statistics
	for i := endIndex; i < totalItems; i++ {
		d := files[i]
		if d.isDir {
			totalFolderCount++
		} else {
			totalFileCount++
			cumulativeFileSize += d.size
		}
	}

	// Return different response formats depending on pagination mode
	if !paginate {
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
	} else {
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
}

func FetchFileStagedForDownload(ctx *gin.Context) {
	fetchFileStagedForDownload(ctx, stageFileLocal, common.XrdConfig.Host, common.XrdConfig.Port)
}

func fetchFileStagedForDownload(ctx *gin.Context, stageFile StageFileFunc, host string, port uint) {
	var req request.DirectoryItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	filePath := req.Path
	if len(filePath) == 0 {
		response.FailWithErr(ctx, *response.SystemErr(errors.New("empty file path for staging")))
		return
	}

	// Stage the file for download
	result, err := stageFile(host, port, filePath)
	if err != nil {
		response.FailWithErr(ctx, *response.SystemErr(err))
		return
	}

	// Send response with the public location of the staged file
	respondData := response.StagedFileResponse{Path: result}
	response.Success(ctx, respondData)
}

// min returns the smaller of two uint32 values
func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
