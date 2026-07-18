package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Listener accepts inbound HTTP connections from an external service
// and provides access to the Plugin for forwarding data to the actuator.
//
// Usage:
//
//	plugin := shared.NewPlugin()
//	listener := shared.ListenLocal(plugin)
//	listener.HandleFunc("POST /push", func(w http.ResponseWriter, r *http.Request) {
//	    // use plugin.Call() to forward data to actuator
//	})
//	go listener.Serve()
//	plugin.Run()
//	listener.Shutdown(context.Background())
type Listener struct {
	ln     net.Listener
	token  string
	mux    *http.ServeMux
	server *http.Server
	plugin *Plugin
}

// ListenLocal binds a localhost TCP port for an external service to connect to.
// It generates a pairing token and writes a connect.json discovery file to
// BRANCHKIT_PLUGIN_DIR so the external service can find the port and token.
//
// When the actuator granted listener sockets (manifest `sockets.listen`,
// delivered per the LISTEN_FDS convention at fds 3+), the FIRST granted
// listener is used instead of self-binding. This is not an optimization:
// inside the Linux sandbox the plugin runs in an empty network namespace,
// where a self-bound "127.0.0.1" is a private dead loopback — the
// inherited host-loopback listener is the only reachable surface. See the
// actuator's notes/DESIGN_SANDBOX_LOOPBACK_FDPASS.md.
//
// The caller must register handlers with HandleFunc before calling Serve.
func ListenLocal(plugin *Plugin) (*Listener, error) {
	ln, err := inheritedListener(0)
	if err != nil {
		return nil, fmt.Errorf("inherited listener: %w", err)
	}
	if ln == nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("listen: %w", err)
		}
	}

	token, err := generateToken()
	if err != nil {
		ln.Close()
		return nil, err
	}

	mux := http.NewServeMux()
	l := &Listener{
		ln:     ln,
		token:  token,
		mux:    mux,
		plugin: plugin,
		server: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate pairing token on all requests
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if strings.TrimPrefix(auth, "Bearer ") != token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			mux.ServeHTTP(w, r)
		})},
	}

	// Write discovery file
	if err := l.writeDiscovery(); err != nil {
		ln.Close()
		return nil, err
	}

	return l, nil
}

// HandleFunc registers an HTTP handler on the listener.
// Patterns follow Go 1.22+ syntax: "POST /path", "GET /path", etc.
func (l *Listener) HandleFunc(pattern string, handler http.HandlerFunc) {
	l.mux.HandleFunc(pattern, handler)
}

// InheritedListeners returns every actuator-granted loopback listener
// declared in the manifest's `sockets.listen`, in declaration order
// (fd 3 = first entry). Returns an empty slice when none were granted —
// e.g. old actuators, unsandboxed dev runs, or no manifest declaration.
// Resolved ports are also published in BRANCHKIT_LISTEN_PORTS as
// comma-separated `id=port` pairs, but Addr() on the returned listeners
// is the simpler source of truth.
//
// Unlike systemd's convention, LISTEN_PID is deliberately NOT set or
// checked: the actuator cannot know the child pid before spawn, and
// plugin identity is already established by fd ownership.
func InheritedListeners() ([]net.Listener, error) {
	n, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || n <= 0 {
		return nil, nil
	}
	listeners := make([]net.Listener, 0, n)
	for i := range n {
		f := os.NewFile(uintptr(3+i), fmt.Sprintf("branchkit-listener-%d", i))
		if f == nil {
			return nil, fmt.Errorf("inherited fd %d is not open", 3+i)
		}
		ln, err := net.FileListener(f)
		// FileListener dups the fd; release our handle either way.
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("inherited fd %d: %w", 3+i, err)
		}
		listeners = append(listeners, ln)
	}
	return listeners, nil
}

// inheritedListener returns the i-th granted listener, or (nil, nil) when
// the actuator granted none (callers then self-bind).
func inheritedListener(i int) (net.Listener, error) {
	listeners, err := InheritedListeners()
	if err != nil {
		return nil, err
	}
	if i >= len(listeners) {
		// Close any listeners we're not returning to avoid fd leaks.
		for _, ln := range listeners {
			ln.Close()
		}
		return nil, nil
	}
	for j, ln := range listeners {
		if j != i {
			ln.Close()
		}
	}
	return listeners[i], nil
}

// Addr returns the listener's address (e.g., "127.0.0.1:52431").
func (l *Listener) Addr() string {
	return l.ln.Addr().String()
}

// Token returns the pairing token that external services must present.
func (l *Listener) Token() string {
	return l.token
}

// Serve starts accepting connections. Blocks until Shutdown is called.
func (l *Listener) Serve() error {
	err := l.server.Serve(l.ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully stops the listener and removes the discovery file.
func (l *Listener) Shutdown(ctx context.Context) {
	l.server.Shutdown(ctx)
	l.removeDiscovery()
}

// ConnectInfo is the discovery file format written to connect.json.
type ConnectInfo struct {
	Port  string `json:"port"`
	Token string `json:"token"`
}

func (l *Listener) writeDiscovery() error {
	pluginDir := os.Getenv("BRANCHKIT_PLUGIN_DIR")
	if pluginDir == "" {
		return nil // no plugin dir — skip discovery file (e.g., in tests)
	}

	_, port, err := net.SplitHostPort(l.ln.Addr().String())
	if err != nil {
		return err
	}

	info := ConnectInfo{
		Port:  port,
		Token: l.token,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal connect.json: %w", err)
	}
	path := filepath.Join(pluginDir, "connect.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func (l *Listener) removeDiscovery() {
	pluginDir := os.Getenv("BRANCHKIT_PLUGIN_DIR")
	if pluginDir != "" {
		os.Remove(filepath.Join(pluginDir, "connect.json"))
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
