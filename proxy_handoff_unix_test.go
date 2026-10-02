//go:build unix

package branchkit

import (
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
	"testing"
)

// A plugin whose BRANCHKIT_PROXY is fd://N asks over that channel for each
// connection and receives it as a passed socket. The fake broker here dials
// the test proxy itself and hands that connection over, so the SDK opens no
// socket of its own; ordinary HTTP then works through it, twice, and an
// undeclared host is still refused by the proxy.
func TestProxyOverHandoffChannel(t *testing.T) {
	srv := targetServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "127.0.0.1")

	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	brokerFile := os.NewFile(uintptr(pair[0]), "broker")
	brokerConn, err := net.FileConn(brokerFile)
	brokerFile.Close()
	if err != nil {
		t.Fatalf("broker conn: %v", err)
	}
	broker := brokerConn.(*net.UnixConn)
	t.Cleanup(func() { broker.Close() })
	go func() {
		ask := make([]byte, 1)
		for {
			if _, err := broker.Read(ask); err != nil {
				return
			}
			up, err := net.Dial("unix", sock)
			if err != nil {
				return
			}
			f, _ := up.(*net.UnixConn).File()
			up.Close()
			broker.WriteMsgUnix([]byte{'c'}, syscall.UnixRights(int(f.Fd())), nil)
			f.Close()
		}
	}()

	client := clientVia(t, "fd://"+strconv.Itoa(pair[1]))
	for i := 0; i < 2; i++ {
		resp, err := client.Get(srv.URL)
		if err != nil {
			t.Fatalf("GET %d through the handoff: %v", i, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "hello through the tunnel" {
			t.Fatalf("GET %d: unexpected body %q", i, body)
		}
	}
	if _, err := clientVia(t, "fd://"+strconv.Itoa(pair[1])).Get("http://evil.example.com/"); err == nil {
		t.Fatal("an undeclared host must be refused through the handoff too")
	}
}
