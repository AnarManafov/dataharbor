package util

import (
	"net"
	"testing"

	"github.com/bwmarrin/snowflake"
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
