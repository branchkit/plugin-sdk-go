//go:build unix

package branchkit

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// The proxy handoff (BRANCHKIT_PROXY=fd://N): the actuator hands each
// connection to the plugin's filtering proxy over an inherited channel, so
// the plugin never opens a socket or names a path (a sandbox may forbid
// both). Ask with one byte; the reply is one byte carrying a connected
// socket (SCM_RIGHTS), already served with this plugin's proxy rules. Every
// reply is an equivalent fresh connection, so asks only need serialising.
//
// A read that gives up (the dial's deadline passed) leaves its reply to
// arrive later. owed counts those: each is read and discarded before the
// next ask, so a dial always takes the reply to its own ask, never an
// earlier one.
var handoff struct {
	once sync.Once
	mu   sync.Mutex
	conn *net.UnixConn
	err  error
	owed int
}

func dialHandoff(ctx context.Context, fdText string) (net.Conn, error) {
	handoff.once.Do(func() {
		n, err := strconv.Atoi(fdText)
		if err != nil || n < 0 {
			handoff.err = fmt.Errorf("bad proxy channel fd %q", fdText)
			return
		}
		f := os.NewFile(uintptr(n), "branchkit-proxy-channel")
		c, err := net.FileConn(f)
		f.Close()
		if err != nil {
			handoff.err = fmt.Errorf("proxy channel fd %d: %w", n, err)
			return
		}
		uc, ok := c.(*net.UnixConn)
		if !ok {
			c.Close()
			handoff.err = fmt.Errorf("proxy channel fd %d is not a unix socket", n)
			return
		}
		handoff.conn = uc
	})
	if handoff.err != nil {
		return nil, handoff.err
	}
	handoff.mu.Lock()
	defer handoff.mu.Unlock()
	if dl, ok := ctx.Deadline(); ok {
		_ = handoff.conn.SetDeadline(dl)
		defer handoff.conn.SetDeadline(time.Time{})
	}
	for handoff.owed > 0 {
		fd, consumed, err := readHandoffReply()
		if consumed {
			handoff.owed--
		}
		if fd >= 0 {
			syscall.Close(fd)
		}
		if err != nil && !consumed {
			return nil, fmt.Errorf("the proxy channel has not yet answered an earlier ask: %w", err)
		}
	}
	if _, err := handoff.conn.Write([]byte{'c'}); err != nil {
		return nil, fmt.Errorf("ask the proxy channel: %w", err)
	}
	handoff.owed++
	fd, consumed, err := readHandoffReply()
	if consumed {
		handoff.owed--
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "branchkit-proxy")
	c, err := net.FileConn(f)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("proxy connection: %w", err)
	}
	return c, nil
}

// readHandoffReply reads one reply off the channel and returns the
// descriptor it carries (-1 if none) and whether a reply was taken off the
// channel at all: a read that times out takes none.
func readHandoffReply() (int, bool, error) {
	buf := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4))
	n, oobn, _, _, err := handoff.conn.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, false, fmt.Errorf("read the proxy channel: %w", err)
	}
	if n == 0 {
		return -1, false, fmt.Errorf("the proxy channel closed")
	}
	msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil || len(msgs) == 0 {
		return -1, true, fmt.Errorf("the proxy channel replied without a connection")
	}
	fds, err := syscall.ParseUnixRights(&msgs[0])
	if err != nil || len(fds) == 0 {
		return -1, true, fmt.Errorf("the proxy channel replied without a connection")
	}
	for _, extra := range fds[1:] {
		syscall.Close(extra)
	}
	return fds[0], true, nil
}
