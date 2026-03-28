package response

type DirectoryItemResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	DateTime string `json:"date_time"`
	Size     uint64 `json:"size"`
}

type VirtualFSStatResponse struct {
	NodesRW            int `json:"nodesRW"`
	FreeSpaceMB        int `json:"freeSpaceMB"`
	UtilizationPercent int `json:"utilizationPercent"`
	NodesStaging       int `json:"nodesStaging"`
	FreeSpaceStagingMB int `json:"freeSpaceStagingMB"`
	UtilizationStaging int `json:"utilizationStaging"`
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
	QueryTimeMs        int64                   `json:"queryTimeMs"`
}

type PingResponse struct {
	LatencyMs float64 `json:"latencyMs"`
	ConnectMs float64 `json:"connectMs"`
	Status    string  `json:"status"`
	Server    string  `json:"server"`
}
