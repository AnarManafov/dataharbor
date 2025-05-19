package test

import (
	"context"
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
)

// BenchmarkXRDClientCreation benchmarks the native XRD client creation
func BenchmarkXRDClientCreation(b *testing.B) {
	initTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := common.GetXRDNativeClient()
		if client == nil {
			b.Fatal("client should not be nil")
		}
	}
}

// BenchmarkClientConnections benchmarks native client connection operations
func BenchmarkClientConnections(b *testing.B) {
	initTestConfig()
	client := common.GetXRDNativeClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test filesystem access
		fs, cleanup, err := client.GetFileSystem(context.Background(), "")
		if err != nil {
			b.Skip("XRootD server not available for benchmark")
		}
		_ = fs // Use the filesystem
		cleanup()
	}
}

// BenchmarkConfigCompatibility benchmarks configuration compatibility
func BenchmarkConfigCompatibility(b *testing.B) {
	initTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test that both client access methods work
		client1 := common.GetXRDClient()
		client2 := common.GetXRDNativeClient()
		if client1 != client2 {
			b.Fatal("clients should be the same instance")
		}
	}
}
