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
var handoff struct {
	once sync.Once
	mu   sync.Mutex
	conn *net.UnixConn
	err  error
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
	if _, err := handoff.conn.Write([]byte{'c'}); err != nil {
		return nil, fmt.Errorf("ask the proxy channel: %w", err)
	}
	buf := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4))
	n, oobn, _, _, err := handoff.conn.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("read the proxy channel: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("the proxy channel closed")
	}
	msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil || len(msgs) == 0 {
		return nil, fmt.Errorf("the proxy channel replied without a connection")
	}
	fds, err := syscall.ParseUnixRights(&msgs[0])
	if err != nil || len(fds) == 0 {
		return nil, fmt.Errorf("the proxy channel replied without a connection")
	}
	for _, extra := range fds[1:] {
		syscall.Close(extra)
	}
	f := os.NewFile(uintptr(fds[0]), "branchkit-proxy")
	c, err := net.FileConn(f)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("proxy connection: %w", err)
	}
	return c, nil
}
