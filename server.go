package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// RunPlugin starts a plugin server on the unix socket specified by
// the BRANCHKIT_PLUGIN_SOCKET env var. It handles SIGTERM for graceful shutdown.
func RunPlugin(handler http.Handler) {
	socketPath := os.Getenv("BRANCHKIT_PLUGIN_SOCKET")
	if socketPath == "" {
		fmt.Fprintln(os.Stderr, "BRANCHKIT_PLUGIN_SOCKET env var required")
		os.Exit(1)
	}

	// Remove stale socket
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to bind unix socket: %v\n", err)
		os.Exit(1)
	}

	pluginID := os.Getenv("BRANCHKIT_PLUGIN_ID")
	if pluginID == "" {
		pluginID = "unknown"
	}
	Log(pluginID, "listening on "+socketPath)

	server := &http.Server{Handler: handler}

	// Graceful shutdown on SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		Log(pluginID, "shutting down")
		server.Shutdown(context.Background())
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "[%s] server error: %v\n", pluginID, err)
		os.Exit(1)
	}
}

// HealthHandler returns an http.HandlerFunc that responds with {"ready": true}.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ready": true})
	}
}

// Log prints a message to stderr with the plugin ID prefix.
func Log(pluginID, msg string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", pluginID, msg)
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ReadJSON reads a JSON request body into v.
func ReadJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
