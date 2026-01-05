package request

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectoryItemsRequest_DefaultValues(t *testing.T) {
	req := DirectoryItemsRequest{}

	assert.Empty(t, req.Path, "Default Path should be empty")
	assert.Equal(t, uint32(0), req.Page, "Default Page should be 0")
	assert.Equal(t, uint32(0), req.PageSize, "Default PageSize should be 0")
}

func TestDirectoryItemsRequest_JSONMarshal(t *testing.T) {
	req := DirectoryItemsRequest{
		Path:     "/home/user/documents",
		Page:     1,
		PageSize: 50,
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err, "JSON marshaling should not fail")

	expected := `{"path":"/home/user/documents","page":1,"pageSize":50}`
	assert.JSONEq(t, expected, string(data))
}

func TestDirectoryItemsRequest_JSONUnmarshal(t *testing.T) {
	jsonData := `{"path":"/tmp/test","page":2,"pageSize":100}`

	var req DirectoryItemsRequest
	err := json.Unmarshal([]byte(jsonData), &req)

	assert.NoError(t, err, "JSON unmarshaling should not fail")
	assert.Equal(t, "/tmp/test", req.Path)
	assert.Equal(t, uint32(2), req.Page)
	assert.Equal(t, uint32(100), req.PageSize)
}

func TestDirectoryItemsRequest_PartialJSON(t *testing.T) {
	// Test with partial JSON (only path)
	jsonData := `{"path":"/data"}`

	var req DirectoryItemsRequest
	err := json.Unmarshal([]byte(jsonData), &req)

	assert.NoError(t, err, "Partial JSON should unmarshal without error")
	assert.Equal(t, "/data", req.Path)
	assert.Equal(t, uint32(0), req.Page, "Missing Page should default to 0")
	assert.Equal(t, uint32(0), req.PageSize, "Missing PageSize should default to 0")
}

func TestDirectoryItemsRequest_EmptyJSON(t *testing.T) {
	jsonData := `{}`

	var req DirectoryItemsRequest
	err := json.Unmarshal([]byte(jsonData), &req)

	assert.NoError(t, err, "Empty JSON should unmarshal without error")
	assert.Empty(t, req.Path)
	assert.Equal(t, uint32(0), req.Page)
	assert.Equal(t, uint32(0), req.PageSize)
}

func TestDirectoryItemsRequest_LargeValues(t *testing.T) {
	req := DirectoryItemsRequest{
		Path:     "/very/long/path/to/some/nested/directory/structure",
		Page:     999999,
		PageSize: 4294967295, // Max uint32
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded DirectoryItemsRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, req.Path, decoded.Path)
	assert.Equal(t, req.Page, decoded.Page)
	assert.Equal(t, req.PageSize, decoded.PageSize)
}

func TestDirectoryItemsRequest_SpecialCharacters(t *testing.T) {
	req := DirectoryItemsRequest{
		Path:     "/path/with spaces/and-dashes/and_underscores/file.txt",
		Page:     1,
		PageSize: 20,
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var decoded DirectoryItemsRequest
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, req.Path, decoded.Path)
}
