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
// The caller must register handlers with HandleFunc before calling Serve.
func ListenLocal(plugin *Plugin) (*Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
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
