package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListenLocalBindsPort(t *testing.T) {
	plugin := NewPlugin()
	listener, err := ListenLocal(plugin)
	if err != nil {
		t.Fatalf("ListenLocal: %v", err)
	}
	defer listener.Shutdown(context.Background())

	addr := listener.Addr()
	if addr == "" {
		t.Fatal("expected non-empty address")
	}

	token := listener.Token()
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("token length=%d, want 64", len(token))
	}
}

func TestListenLocalAuthRequired(t *testing.T) {
	plugin := NewPlugin()
	listener, err := ListenLocal(plugin)
	if err != nil {
		t.Fatalf("ListenLocal: %v", err)
	}
	go listener.Serve()
	defer listener.Shutdown(context.Background())

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + listener.Addr()

	// Register a handler so we can tell if auth was bypassed
	listener.HandleFunc("POST /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No token → 401
	req, _ := http.NewRequest("POST", base+"/test", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token: status=%d, want 401", resp.StatusCode)
	}

	// Wrong token → 403
	req, _ = http.NewRequest("POST", base+"/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("wrong token: status=%d, want 403", resp.StatusCode)
	}

	// Correct token → 200
	req, _ = http.NewRequest("POST", base+"/test", nil)
	req.Header.Set("Authorization", "Bearer "+listener.Token())
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("correct token: status=%d, want 200", resp.StatusCode)
	}
}

func TestListenLocalDiscoveryFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRANCHKIT_PLUGIN_DIR", dir)

	plugin := NewPlugin()
	listener, err := ListenLocal(plugin)
	if err != nil {
		t.Fatalf("ListenLocal: %v", err)
	}

	// Verify connect.json was written
	data, err := os.ReadFile(filepath.Join(dir, "connect.json"))
	if err != nil {
		t.Fatalf("read connect.json: %v", err)
	}

	var info ConnectInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("parse connect.json: %v", err)
	}

	if info.Port == "" {
		t.Error("expected non-empty port")
	}
	if info.Token != listener.Token() {
		t.Errorf("token mismatch: file=%q, listener=%q", info.Token, listener.Token())
	}

	// Shutdown should remove the file
	listener.Shutdown(context.Background())

	if _, err := os.Stat(filepath.Join(dir, "connect.json")); !os.IsNotExist(err) {
		t.Error("connect.json not removed after shutdown")
	}
}

func TestListenLocalDiscoveryWriteFailure(t *testing.T) {
	// Point to a nonexistent directory — writeDiscovery should fail
	t.Setenv("BRANCHKIT_PLUGIN_DIR", "/nonexistent/path/that/does/not/exist")

	plugin := NewPlugin()
	_, err := ListenLocal(plugin)
	if err == nil {
		t.Fatal("expected error for nonexistent plugin dir")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("error=%q, expected to mention 'write'", err.Error())
	}
}

func TestListenLocalNoPluginDir(t *testing.T) {
	// Ensure BRANCHKIT_PLUGIN_DIR is unset
	t.Setenv("BRANCHKIT_PLUGIN_DIR", "")

	plugin := NewPlugin()
	listener, err := ListenLocal(plugin)
	if err != nil {
		t.Fatalf("ListenLocal: %v", err)
	}
	defer listener.Shutdown(context.Background())

	// Should succeed without writing a discovery file
	if listener.Addr() == "" {
		t.Error("expected non-empty address")
	}
}
