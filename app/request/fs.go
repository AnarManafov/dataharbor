package request

type DirectoryItemsRequest struct {
	Path     string `json:"path"`
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"pageSize"`
}

// BatchDownloadRequest defines the request body for multi-file tar streaming download
type BatchDownloadRequest struct {
	BasePath string   `json:"basePath" binding:"required"`
	Files    []string `json:"files" binding:"required"`
}
