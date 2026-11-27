//go:build !linux

package main

import "net"

// sendQueueBytes is only implemented on Linux, the platform the backend is
// deployed on. Elsewhere connections are closed without waiting.
func sendQueueBytes(_ *net.TCPConn) (queued int, ok bool) {
	return 0, false
}
