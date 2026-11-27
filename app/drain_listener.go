package main

import (
	"net"
	"sync/atomic"
	"time"
)

// drainListener hands out connections whose Close waits until the kernel has
// delivered what was written to them.
//
// A handler returns as soon as its last byte is in the socket's send buffer,
// which can hold several megabytes for a slow client. A plain process exit
// leaves the kernel to deliver that tail, but a container's network namespace
// is destroyed with the process, and the tail of every such response with it.
// Holding the connection open until its queue is empty lets a shutdown know
// when "the request completed" is true for the client too.
type drainListener struct {
	net.Listener

	// deadline is zero until a shutdown starts; from then on no Close waits
	// beyond that instant (UnixNano).
	deadline atomic.Int64
	// closing counts connections that are waiting in Close.
	closing atomic.Int64
	// unsent counts connections closed with data still queued because the
	// shutdown deadline arrived.
	unsent atomic.Int64
}

func (l *drainListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		return &drainConn{TCPConn: tc, owner: l}, nil
	}
	return c, nil
}

// startDrain bounds every Close, running or future, by deadline.
func (l *drainListener) startDrain(deadline time.Time) {
	l.deadline.Store(deadline.UnixNano())
}

// waitClosed blocks until no connection is waiting in Close any more. Every
// such wait ends at the shutdown deadline, so this returns shortly after it at
// the latest. Call it after startDrain.
func (l *drainListener) waitClosed() {
	for l.closing.Load() > 0 {
		time.Sleep(sendQueuePollMin)
	}
}

// drainConn embeds *net.TCPConn so the server keeps its fast paths (ReadFrom).
type drainConn struct {
	*net.TCPConn
	owner *drainListener
	// closeStarted makes a second Close a no-op: the server may close a
	// connection from its own goroutine and again from Server.Close. The first
	// call owns the wait and the real close, which the shutdown deadline bounds.
	closeStarted atomic.Bool
}

const (
	// The send queue is sampled with a growing interval: a small response is
	// acknowledged within a millisecond or two and must not be held up.
	sendQueuePollMin = time.Millisecond
	sendQueuePollMax = 50 * time.Millisecond

	// maxCloseWait bounds the wait outside a shutdown, so a client that has
	// stopped reading cannot pin a connection for ever.
	maxCloseWait = 2 * time.Minute
)

func (c *drainConn) Close() error {
	if !c.closeStarted.CompareAndSwap(false, true) {
		return nil
	}
	l := c.owner
	l.closing.Add(1)
	defer l.closing.Add(-1)

	giveUp := time.Now().Add(maxCloseWait)
	for wait := sendQueuePollMin; ; wait = min(wait*2, sendQueuePollMax) {
		queued, ok := sendQueueBytes(c.TCPConn)
		if !ok || queued == 0 {
			break
		}
		now := time.Now()
		if ns := l.deadline.Load(); ns != 0 && !now.Before(time.Unix(0, ns)) {
			l.unsent.Add(1)
			// Reset instead of a normal close: the queued data is lost either
			// way, and a reset reaches the peer before the process exits. With
			// a normal close a proxy in front would wait for its read timeout,
			// and its client with it, for a response that can no longer come.
			_ = c.SetLinger(0)
			break
		}
		if !now.Before(giveUp) {
			break
		}
		time.Sleep(wait)
	}
	return c.TCPConn.Close()
}
