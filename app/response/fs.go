package response

type DirItemResp struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	DateTime string `json:"date_time"`
	Size     uint64 `json:"size"`
}

type StageFileResp struct {
	Path string `json:"path"`
}
