package controller

import (
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
