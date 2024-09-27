package controller

import (
	"context"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetCachedData(t *testing.T) {
	key := "testKey"
	data := []xrdDirEntry{
		{name: "file1", dt: time.Now(), size: 123, isDir: false},
	}
	setCachedData(key, data)

	cachedData, found := getCachedData(key)
	assert.True(t, found)
	assert.Equal(t, data, cachedData)
}

func TestGetCachedData(t *testing.T) {
	// Test case: Cache hit
	t.Run("cache hit", func(t *testing.T) {
		key := "testKey"
		expectedData := []xrdDirEntry{
			{name: "file1", dt: time.Now(), size: 123, isDir: false},
			{name: "dir1", dt: time.Now(), size: 0, isDir: true},
		}
		setCachedData(key, expectedData)

		data, found := getCachedData(key)
		assert.True(t, found)
		assert.Equal(t, expectedData, data)
	})

	// Test case: Cache miss (key not found)
	t.Run("cache miss - key not found", func(t *testing.T) {
		key := "nonExistentKey"
		data, found := getCachedData(key)
		assert.False(t, found)
		assert.Nil(t, data)
	})

	// Test case: Cache miss (expired entry)
	t.Run("cache miss - expired entry", func(t *testing.T) {
		key := "expiredKey"
		expectedData := []xrdDirEntry{
			{name: "file1", dt: time.Now(), size: 123, isDir: false},
		}
		cache[key] = cacheEntry{
			data:      expectedData,
			timestamp: time.Now().Add(-2 * cacheTTL), // Set timestamp to be expired
		}

		data, found := getCachedData(key)
		assert.False(t, found)
		assert.Nil(t, data)
	})

	// Test case: Cache with 500K records
	t.Run("cache with 500K records", func(t *testing.T) {
		key := "largeCacheKey"
		var expectedData []xrdDirEntry
		for i := 0; i < 500000; i++ {
			expectedData = append(expectedData, xrdDirEntry{
				name:  "file" + strconv.Itoa(i),
				dt:    time.Now(),
				size:  uint64(i),
				isDir: i%2 == 0,
			})
		}
		setCachedData(key, expectedData)

		// Check if the data is cached.
		// Do it 2 times to ensure that the data is cached correctly.
		data1, found1 := getCachedData(key)
		assert.True(t, found1)
		assert.Equal(t, expectedData, data1)
		data2, found2 := getCachedData(key)
		assert.True(t, found2)
		assert.Equal(t, expectedData, data2)
	})
}

// Mock function for exec.CommandContext
func mockExecCommand(ctx context.Context, name string, arg ...string) *exec.Cmd {
	output := `drwxr-xr-x user staff    96 2023-05-09 09:26:08 /Users/user/Development
drwx------ user staff   320 2023-05-09 06:47:54 /Users/user/Documents
drwx------ user staff   608 2023-10-06 07:55:55 /Users/user/Downloads
dr-x------ user staff   224 2023-05-11 08:24:48 /Users/user/Google Drive`
	return exec.Command("echo", "-n", output)
}

func TestRunXrdFs(t *testing.T) {
	expectedOutput := `drwxr-xr-x user staff    96 2023-05-09 09:26:08 /Users/user/Development
drwx------ user staff   320 2023-05-09 06:47:54 /Users/user/Documents
drwx------ user staff   608 2023-10-06 07:55:55 /Users/user/Downloads
dr-x------ user staff   224 2023-05-11 08:24:48 /Users/user/Google Drive`
	output, err := RunXrdFs(mockExecCommand, "xrdfs", "ls", "-l")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}
