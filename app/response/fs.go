package response

type DirItemResp struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	DateTime string `json:"date_time"`
	Size     uint64 `json:"size"`
}

type DirResp struct {
	Code               int           `json:"code"`
	Items              []DirItemResp `json:"items"`
	TotalItems         int           `json:"totalItems"`
	PageSize           int           `json:"pageSize"`
	TotalPages         int           `json:"totalPages"`
	TotalFileCount     int           `json:"totalFileCount"`
	TotalFolderCount   int           `json:"totalFolderCount"`
	CumulativeFileSize int           `json:"cumulativeFileSize"`
}

type StageFileResp struct {
	Path string `json:"path"`
}
