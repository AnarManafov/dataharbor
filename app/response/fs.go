package response

type DirectoryItemResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	DateTime string `json:"date_time"`
	Size     uint64 `json:"size"`
}

type DirectoryResponse struct {
	Code               int                     `json:"code"`
	Items              []DirectoryItemResponse `json:"items"`
	TotalItems         int                     `json:"totalItems"`
	PageSize           int                     `json:"pageSize"`
	TotalPages         int                     `json:"totalPages"`
	TotalFileCount     int                     `json:"totalFileCount"`
	TotalFolderCount   int                     `json:"totalFolderCount"`
	CumulativeFileSize int                     `json:"cumulativeFileSize"`
}

type StagedFileResponse struct {
	Path string `json:"path"`
}
