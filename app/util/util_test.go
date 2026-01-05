package util

import (
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/assert"
)

func TestIpv4ToLong(t *testing.T) {
	tests := []struct {
		ip       string
		expected uint
		hasError bool
	}{
		{"192.168.1.1", 3232235777, false},
		{"255.255.255.255", 4294967295, false},
		{"0.0.0.0", 0, false},
		{"invalid_ip", 0, true},
	}

	for _, test := range tests {
		result, err := Ipv4ToLong(test.ip)
		if (err != nil) != test.hasError {
			t.Errorf("Ipv4ToLong(%s) error = %v, expected error = %v", test.ip, err, test.hasError)
		}
		if result != test.expected {
			t.Errorf("Ipv4ToLong(%s) = %d, expected %d", test.ip, result, test.expected)
		}
	}
}

func TestGetClientIp(t *testing.T) {
	ip, err := getClientIp()
	if err != nil {
		t.Errorf("getClientIp() error = %v", err)
	}
	if net.ParseIP(ip) == nil {
		t.Errorf("getClientIp() returned invalid IP: %s", ip)
	}
}

func TestNextUid(t *testing.T) {
	// Initialize snowNode for testing
	var err error
	snowNode, err = snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("Failed to initialize snowflake node: %v", err)
	}

	uid := NextUid()
	if uid == "" {
		t.Errorf("NextUid() returned an empty string")
	}
}

func TestInitSnowflake(t *testing.T) {
	// Reset the once and snowNode for this test
	// Note: We need to be careful since once.Do only runs once
	// For testing purposes, we verify InitSnowflake returns no error
	// when called (since it may have already been initialized)

	// If snowNode is nil, InitSnowflake should initialize it
	// If already initialized, it should be a no-op due to sync.Once
	err := InitSnowflake()

	// Should not return an error on a normal system
	assert.NoError(t, err, "InitSnowflake should not return an error")
	assert.NotNil(t, snowNode, "snowNode should be initialized after InitSnowflake")
}

func TestInitSnowflake_AlreadyInitialized(t *testing.T) {
	// Initialize first
	_ = InitSnowflake()

	// Second call should be a no-op
	err := InitSnowflake()
	assert.NoError(t, err, "Second call to InitSnowflake should not fail")
}

func TestNextUid_Uniqueness(t *testing.T) {
	// Reset for clean test
	snowNode, _ = snowflake.NewNode(1)

	// Generate multiple UIDs and check uniqueness
	uids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		uid := NextUid()
		assert.NotEmpty(t, uid, "NextUid should not return empty string")

		if uids[uid] {
			t.Errorf("Duplicate UID generated: %s", uid)
		}
		uids[uid] = true
	}

	assert.Equal(t, count, len(uids), "All generated UIDs should be unique")
}

func TestNextUid_Concurrent(t *testing.T) {
	// Reset for clean test
	snowNode, _ = snowflake.NewNode(1)

	var wg sync.WaitGroup
	uidChan := make(chan string, 100)
	goroutines := 10
	uidsPerGoroutine := 10

	// Generate UIDs concurrently
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < uidsPerGoroutine; j++ {
				uid := NextUid()
				uidChan <- uid
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(uidChan)

	// Collect all UIDs and check for uniqueness
	uids := make(map[string]bool)
	for uid := range uidChan {
		assert.NotEmpty(t, uid)
		if uids[uid] {
			t.Errorf("Duplicate UID generated in concurrent test: %s", uid)
		}
		uids[uid] = true
	}

	expectedCount := goroutines * uidsPerGoroutine
	assert.Equal(t, expectedCount, len(uids), "All concurrent UIDs should be unique")
}

func TestIpv4ToLong_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint
		hasError bool
	}{
		{"minimum IP", "0.0.0.0", 0, false},
		{"maximum IP", "255.255.255.255", 4294967295, false},
		{"localhost", "127.0.0.1", 2130706433, false},
		{"common private", "10.0.0.1", 167772161, false},
		{"IPv6 address", "::1", 0, true},
		{"empty string", "", 0, true},
		{"partial IP", "192.168", 0, true},
		{"too many octets", "192.168.1.1.1", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Ipv4ToLong(tt.ip)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetClientIp_ReturnsValidIP(t *testing.T) {
	// Test that getClientIp returns a valid non-loopback IPv4 address
	ip, err := getClientIp()
	// On most systems, this should succeed
	// Only skip if there's no network interface
	if err != nil {
		t.Skip("No non-loopback network interface available")
	}

	assert.NotEmpty(t, ip, "IP should not be empty")

	// Verify the IP is valid IPv4
	parsedIP := net.ParseIP(ip)
	assert.NotNil(t, parsedIP, "Returned IP should be parseable")
	assert.NotNil(t, parsedIP.To4(), "Returned IP should be IPv4")
}

func TestNextUid_NotEmpty(t *testing.T) {
	// Ensure snowNode is initialized
	snowNode, _ = snowflake.NewNode(1)

	uid := NextUid()
	assert.NotEmpty(t, uid, "NextUid should return non-empty string")
	assert.NotEqual(t, "0", uid, "NextUid should return non-zero ID")
}

func TestNextUid_Format(t *testing.T) {
	// Ensure snowNode is initialized
	snowNode, _ = snowflake.NewNode(1)

	uid := NextUid()

	// Snowflake IDs are numeric strings
	_, err := strconv.ParseInt(uid, 10, 64)
	assert.NoError(t, err, "NextUid should return a valid numeric string")
}

func TestNextUid_AutoInitializes(t *testing.T) {
	// Reset snowNode to nil to test auto-initialization path
	// Note: This is testing the scenario where NextUid is called before InitSnowflake
	// Since we can't truly reset sync.Once, we just verify NextUid works

	// This tests the path where snowNode might be nil
	// The function should either already be initialized or will panic on init failure
	uid := NextUid()
	assert.NotEmpty(t, uid, "NextUid should return non-empty after auto-init")
}

func TestNextUid_UniqueGeneration(t *testing.T) {
	// Ensure snowNode is initialized
	snowNode, _ = snowflake.NewNode(1)

	ids := make(map[string]bool)
	numIds := 1000

	for i := 0; i < numIds; i++ {
		uid := NextUid()
		assert.False(t, ids[uid], "NextUid should generate unique IDs")
		ids[uid] = true
	}

	assert.Len(t, ids, numIds, "Should have generated %d unique IDs", numIds)
}
