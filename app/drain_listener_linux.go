//go:build linux

package main

import (
	"net"

	"golang.org/x/sys/unix"
)

// sendQueueBytes returns the number of bytes written to conn that the peer has
// not acknowledged yet (SIOCOUTQ). ok is false if the socket cannot be queried,
// for example because the peer has already reset it.
func sendQueueBytes(conn *net.TCPConn) (queued int, ok bool) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, false
	}
	var ioctlErr error
	if err := raw.Control(func(fd uintptr) {
		queued, ioctlErr = unix.IoctlGetInt(int(fd), unix.SIOCOUTQ)
	}); err != nil || ioctlErr != nil {
		return 0, false
	}
	return queued, true
}
