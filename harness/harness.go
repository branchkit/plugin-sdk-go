// Package harness provides test helpers for BranchKit plugin behavioral tests.
//
// It spawns the branchkit-test-harness binary in --server mode and communicates
// via JSON-RPC 2.0 over stdin/stdout. Plugin developers use this to write tests
// that set up state, exercise the plugin, and assert results without running the
// full BranchKit app.
//
//	func TestMyCommand(t *testing.T) {
//	    h := harness.Start(t, ".")
//	    h.SetTag("plugin.voice.mode.command")
//	    result := h.SimulateCommand("switch to dictation mode")
//	    if !result.Matched {
//	        t.Fatal("expected command to match")
//	    }
//	    tags := h.GetTags("plugin.voice.mode.*")
//	    if _, ok := tags["plugin.voice.mode.dictation"]; !ok {
//	        t.Error("expected dictation mode tag")
//	    }
//	}
package harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Harness wraps a running branchkit-test-harness process in --server mode.
type Harness struct {
	t       testing.TB
	cmd     *exec.Cmd
	writer  *json.Encoder
	scanner *bufio.Scanner
	mu      sync.Mutex
	nextID  atomic.Uint64
	timeout time.Duration
}

// Start spawns the test harness and loads the plugin at dir.
// The harness binary is located via BRANCHKIT_TEST_HARNESS env var, or by
// searching common build output paths relative to the workspace root.
// Cleanup is registered via t.Cleanup — no need to call Stop manually.
func Start(t testing.TB, dir string) *Harness {
	t.Helper()

	binary := findHarnessBinary(t)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("harness: cannot resolve dir %q: %v", dir, err)
	}

	cmd := exec.Command(binary, "--server")
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("harness: stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("harness: stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("harness: start: %v", err)
	}

	h := &Harness{
		t:       t,
		cmd:     cmd,
		writer:  json.NewEncoder(stdin),
		scanner: bufio.NewScanner(stdout),
		timeout: 30 * time.Second,
	}

	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	})

	// Send test.start to load the plugin
	var startResult struct {
		PluginID string `json:"plugin_id"`
	}
	h.call("test.start", map[string]any{"dir": absDir}, &startResult)
	return h
}

// Stop shuts down the plugin. Normally unnecessary — cleanup is automatic.
func (h *Harness) Stop() {
	h.t.Helper()
	h.call("test.stop", map[string]any{}, nil)
}

// Reset clears all state and restarts the plugin.
func (h *Harness) Reset() {
	h.t.Helper()
	h.call("test.reset", map[string]any{}, nil)
}

// SetTag sets a tag in the active gates.
func (h *Harness) SetTag(tag string) {
	h.t.Helper()
	h.call("test.set_tag", map[string]any{"tag": tag}, nil)
}

// ClearTag removes a tag from the active gates.
func (h *Harness) ClearTag(tag string) {
	h.t.Helper()
	h.call("test.clear_tag", map[string]any{"tag": tag}, nil)
}

// GetTags returns all active tags matching a glob pattern.
// The pattern supports trailing wildcards: "plugin.voice.*" matches all tags
// under that prefix. Use "*" to get all tags.
func (h *Harness) GetTags(pattern string) []string {
	h.t.Helper()
	var result struct {
		Tags []string `json:"tags"`
	}
	h.call("test.get_tags", map[string]any{"pattern": pattern}, &result)
	return result.Tags
}

// SimulateResult holds the outcome of a command simulation.
type SimulateResult struct {
	Matched        bool              `json:"matched"`
	Action         json.RawMessage   `json:"action,omitempty"`
	Args           []json.RawMessage `json:"args,omitempty"`
	ConsumedCount  int               `json:"consumed_count,omitempty"`
	SetsTags       []string          `json:"sets_tags,omitempty"`
	ClearsTags     []string          `json:"clears_tags,omitempty"`
	OwnerPlugin    string            `json:"owner_plugin,omitempty"`
	ActionResponse *ActionResponse   `json:"action_response,omitempty"`
}

// ActionResponse holds the plugin's on_action reply when the matched command
// has a Plugin action. Nil when the command has no action or the action is
// not a Plugin type.
type ActionResponse struct {
	Status         string          `json:"status,omitempty"`
	ControlMessage string          `json:"control_message,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
}

// SimulateCommand feeds a phrase to the matching engine, executes the matched
// action's tag writes, and returns the match result.
func (h *Harness) SimulateCommand(phrase string) SimulateResult {
	h.t.Helper()
	var result SimulateResult
	h.call("test.simulate_command", map[string]any{"phrase": phrase}, &result)
	return result
}

// CollectionResult holds the data returned by GetCollection.
type CollectionResult struct {
	Name          string                        `json:"name"`
	Introducer    string                        `json:"introducer"`
	Contributions map[string]json.RawMessage    `json:"contributions"`
}

// GetCollection reads a collection's contributions.
func (h *Harness) GetCollection(name string) CollectionResult {
	h.t.Helper()
	var result CollectionResult
	h.call("test.get_collection", map[string]any{"name": name}, &result)
	return result
}

// WriteCollection writes data to a collection. The contributor defaults to
// "_test_harness" if empty.
func (h *Harness) WriteCollection(name string, data any, contributor string) {
	h.t.Helper()
	params := map[string]any{"name": name, "data": data}
	if contributor != "" {
		params["contributor"] = contributor
	}
	h.call("test.write_collection", params, nil)
}

// CallPlugin sends a JSON-RPC method call directly to the plugin process and
// unmarshals the response into result. Pass nil for result to discard it.
func (h *Harness) CallPlugin(method string, params any, result any) {
	h.t.Helper()
	rpcParams := map[string]any{"method": method, "params": params}
	h.call("test.call_plugin_method", rpcParams, result)
}

// PluginState holds process health and RPC statistics.
type PluginState struct {
	Alive         bool     `json:"alive"`
	PluginID      string   `json:"plugin_id"`
	RPCCallCount  int      `json:"rpc_call_count"`
	RPCErrorCount int      `json:"rpc_error_count"`
	RPCMethodsSeen []string `json:"rpc_methods_seen"`
}

// GetPluginState returns process liveness and RPC call statistics.
func (h *Harness) GetPluginState() PluginState {
	h.t.Helper()
	var result PluginState
	h.call("test.get_plugin_state", map[string]any{}, &result)
	return result
}

// --- JSON-RPC transport ---

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RpcError represents an error returned by the harness RPC server.
type RpcError struct {
	Code    int
	Message string
}

func (e *RpcError) Error() string {
	return fmt.Sprintf("harness: RPC error %d: %s", e.Code, e.Message)
}

func (h *Harness) call(method string, params any, result any) {
	h.t.Helper()
	if err := h.tryCall(method, params, result); err != nil {
		h.t.Fatalf("harness: %s: %v", method, err)
	}
}

func (h *Harness) tryCall(method string, params any, result any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID.Add(1)
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	req := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  paramsBytes,
	}
	if err := h.writer.Encode(req); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	done := make(chan struct{})
	var scanOK bool
	go func() {
		scanOK = h.scanner.Scan()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(h.timeout):
		return fmt.Errorf("timeout after %s", h.timeout)
	}

	if !scanOK {
		if err := h.scanner.Err(); err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		return fmt.Errorf("EOF reading response")
	}

	var resp rpcMessage
	if err := json.Unmarshal(h.scanner.Bytes(), &resp); err != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", err, h.scanner.Text())
	}

	if resp.Error != nil {
		return &RpcError{Code: resp.Error.Code, Message: resp.Error.Message}
	}

	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}

// TrySimulateCommand is like SimulateCommand but returns an error instead of
// failing the test. Useful for testing that a method returns an expected error.
func (h *Harness) TrySimulateCommand(phrase string) (SimulateResult, error) {
	var result SimulateResult
	err := h.tryCall("test.simulate_command", map[string]any{"phrase": phrase}, &result)
	return result, err
}

// TryCallPlugin is like CallPlugin but returns an error instead of failing the test.
func (h *Harness) TryCallPlugin(method string, params any, result any) error {
	return h.tryCall("test.call_plugin_method", map[string]any{"method": method, "params": params}, result)
}

func findHarnessBinary(t testing.TB) string {
	t.Helper()

	if env := os.Getenv("BRANCHKIT_TEST_HARNESS"); env != "" {
		return env
	}

	// Walk up from CWD looking for a Cargo target directory
	candidates := []string{
		"target/debug/branchkit-test-harness",
		"target/release/branchkit-test-harness",
		"../target/debug/branchkit-test-harness",
		"../target/release/branchkit-test-harness",
		"../../target/debug/branchkit-test-harness",
		"../../target/release/branchkit-test-harness",
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	// Try PATH
	if p, err := exec.LookPath("branchkit-test-harness"); err == nil {
		return p
	}

	t.Fatal("harness: cannot find branchkit-test-harness binary. " +
		"Set BRANCHKIT_TEST_HARNESS or run 'cargo build -p branchkit-test-harness'")
	return ""
}

// StartWithTimeout is like Start but allows a custom RPC call timeout.
// The default Start uses 30 seconds.
func StartWithTimeout(t testing.TB, dir string, timeout time.Duration) *Harness {
	t.Helper()
	h := Start(t, dir)
	h.timeout = timeout
	return h
}

// MustSimulateCommand is like SimulateCommand but fails the test if no match.
func (h *Harness) MustSimulateCommand(phrase string) SimulateResult {
	h.t.Helper()
	result := h.SimulateCommand(phrase)
	if !result.Matched {
		h.t.Fatalf("harness: expected %q to match a command, but it didn't", phrase)
	}
	return result
}

// SetWorld configures the synthetic world model (windows, displays, active app).
func (h *Harness) SetWorld(world any) {
	h.t.Helper()
	h.call("test.set_world", world, nil)
}

// RequireTag is a test assertion: fails if the tag is not currently active.
func (h *Harness) RequireTag(tag string) {
	h.t.Helper()
	tags := h.GetTags(tag)
	if !slices.Contains(tags, tag) {
		h.t.Fatalf("harness: expected tag %q to be active, but it was not", tag)
	}
}

// RequireNoTag is a test assertion: fails if the tag IS currently active.
func (h *Harness) RequireNoTag(tag string) {
	h.t.Helper()
	tags := h.GetTags(tag)
	if slices.Contains(tags, tag) {
		h.t.Fatalf("harness: expected tag %q to NOT be active, but it was", tag)
	}
}

// Reload re-reads the manifest from disk and restarts the plugin. Use this to
// test hot reload behavior after modifying plugin.json or the binary.
func (h *Harness) Reload() {
	h.t.Helper()
	h.call("test.reload", map[string]any{}, nil)
}

// LoadManifest loads a dependency plugin's manifest without spawning its
// binary. Accepts a local directory path or a plugin name (resolved via
// installed plugins, then the catalog).
func (h *Harness) LoadManifest(dirOrName string) {
	h.t.Helper()
	absDir, err := filepath.Abs(dirOrName)
	if err == nil {
		if _, statErr := os.Stat(filepath.Join(absDir, "plugin.json")); statErr == nil {
			h.call("test.load_manifest", map[string]any{"dir": absDir}, nil)
			return
		}
	}
	h.call("test.load_manifest", map[string]any{"name": dirOrName}, nil)
}

// DepStatus represents the resolution status of a single dependency.
type DepStatus struct {
	Plugin  string  `json:"plugin"`
	Status  string  `json:"status"`
	Version *string `json:"version,omitempty"`
	Source  *string `json:"source,omitempty"`
	Reason  *string `json:"reason,omitempty"`
}

// ResolveDeps resolves all depends_on entries for the running plugin and
// reports their status.
func (h *Harness) ResolveDeps() []DepStatus {
	h.t.Helper()
	var result struct {
		Deps []DepStatus `json:"deps"`
	}
	h.call("test.resolve_deps", map[string]any{}, &result)
	return result.Deps
}

// InjectEvent fires an event on the event bus. For plugin events, use any
// event_type name. For platform events, use a valid _platform.* event type.
func (h *Harness) InjectEvent(eventType string, data any) {
	h.t.Helper()
	h.call("test.inject_event", map[string]any{
		"event_type": eventType,
		"data":       data,
	}, nil)
}

// HUDResult holds the content of a single HUD channel.
type HUDResult struct {
	Channel  string `json:"channel"`
	HTML     string `json:"html"`
	TargetID string `json:"target_id,omitempty"`
	Raw      bool   `json:"raw,omitempty"`
}

// GetHUD reads the last fragment pushed to a HUD channel.
func (h *Harness) GetHUD(channel string) HUDResult {
	h.t.Helper()
	var result HUDResult
	h.call("test.get_hud", map[string]any{"channel": channel}, &result)
	return result
}

// HUDChannelInfo describes a registered HUD channel.
type HUDChannelInfo struct {
	Channel     string `json:"channel"`
	PluginID    string `json:"plugin_id"`
	Description string `json:"description"`
}

// ListHUDChannels returns all registered HUD channels.
func (h *Harness) ListHUDChannels() []HUDChannelInfo {
	h.t.Helper()
	var result struct {
		Channels []HUDChannelInfo `json:"channels"`
	}
	h.call("test.get_hud", map[string]any{}, &result)
	return result.Channels
}

// ConformancePhase holds the results of a conformance test phase.
type ConformancePhase struct {
	Phase string           `json:"phase"`
	Tests []ConformanceTest `json:"tests"`
}

// ConformanceTest holds a single test result within a phase.
type ConformanceTest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// RpcLogEntry records a single RPC call made by the plugin during its lifecycle.
type RpcLogEntry struct {
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	OK      bool            `json:"ok"`
	Stubbed bool            `json:"stubbed"`
}

// GetRpcLog returns the full ordered log of RPC calls made by the plugin.
func (h *Harness) GetRpcLog() []RpcLogEntry {
	h.t.Helper()
	var result struct {
		Entries []RpcLogEntry `json:"entries"`
	}
	h.call("test.get_rpc_log", map[string]any{}, &result)
	return result.Entries
}

// RunStaticAnalysis runs Phase 1 conformance checks against the loaded plugin.
func (h *Harness) RunStaticAnalysis() ConformancePhase {
	h.t.Helper()
	var result ConformancePhase
	h.call("test.run_static_analysis", map[string]any{}, &result)
	return result
}

// RunStartupCheck runs grammar/command registration conformance checks.
func (h *Harness) RunStartupCheck() ConformancePhase {
	h.t.Helper()
	var result ConformancePhase
	h.call("test.run_startup_check", map[string]any{}, &result)
	return result
}

// RunRPCContract runs RPC contract conformance checks.
func (h *Harness) RunRPCContract() ConformancePhase {
	h.t.Helper()
	var result ConformancePhase
	h.call("test.run_rpc_contract", map[string]any{}, &result)
	return result
}

// RunSettingsCheck runs settings tab rendering conformance checks.
func (h *Harness) RunSettingsCheck() ConformancePhase {
	h.t.Helper()
	var result ConformancePhase
	h.call("test.run_settings_check", map[string]any{}, &result)
	return result
}

// RunDependencyCheck runs dependency validation conformance checks.
func (h *Harness) RunDependencyCheck() ConformancePhase {
	h.t.Helper()
	var result ConformancePhase
	h.call("test.run_dependency_check", map[string]any{}, &result)
	return result
}

// ConformanceResult holds all phases from a full conformance run.
type ConformanceResult struct {
	Phases []ConformancePhase `json:"phases"`
}

// RunAll runs all conformance phases against the loaded plugin and returns
// structured results. Does not fail the test on conformance failures — inspect
// the returned phases to assert specific outcomes.
func (h *Harness) RunAll() ConformanceResult {
	h.t.Helper()
	var result ConformanceResult
	h.call("test.run_all", map[string]any{}, &result)
	return result
}

// ActionType extracts the action type string from a SimulateResult's Action field.
// Returns empty string if the action is nil or doesn't have a type field.
func (r *SimulateResult) ActionType() string {
	if r.Action == nil {
		return ""
	}
	var action struct {
		ActionType string `json:"action_type"`
	}
	if json.Unmarshal(r.Action, &action) != nil {
		return ""
	}
	return action.ActionType
}

// ActionParams unmarshals the action's params into the given struct.
func (r *SimulateResult) ActionParams(v any) error {
	if r.Action == nil {
		return fmt.Errorf("no action in result")
	}
	var action struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(r.Action, &action); err != nil {
		return err
	}
	return json.Unmarshal(action.Params, v)
}
