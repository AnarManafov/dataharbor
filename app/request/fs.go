package request

type DirectoryItemsRequest struct {
	Path     string `json:"path"`
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"pageSize"`
}
