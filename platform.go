package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// PlatformClient makes HTTP requests to the actuator over its unix socket.
type PlatformClient struct {
	client *http.Client
	token  string
}

// NewPlatformClient creates a client connected to the actuator socket from BRANCHKIT_SOCKET.
// Reads BRANCHKIT_PLUGIN_TOKEN for authenticated requests.
func NewPlatformClient() *PlatformClient {
	socketPath := os.Getenv("BRANCHKIT_SOCKET")
	if socketPath == "" {
		fmt.Fprintln(os.Stderr, "BRANCHKIT_SOCKET env var required")
		os.Exit(1)
	}
	return &PlatformClient{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: 10 * time.Second,
		},
		token: os.Getenv("BRANCHKIT_PLUGIN_TOKEN"),
	}
}

func (p *PlatformClient) get(path string, result any) error {
	req, err := http.NewRequest("GET", "http://localhost"+path, nil)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// GetJSON makes a GET request and decodes the JSON response.
func (p *PlatformClient) GetJSON(path string, result any) error {
	return p.get(path, result)
}

// Delete sends a DELETE request.
func (p *PlatformClient) Delete(path string) error {
	req, err := http.NewRequest("DELETE", "http://localhost"+path, nil)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DELETE %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}
	return nil
}

func (p *PlatformClient) post(path string, body any, result any) error {
	return p.PostJSON(path, body, "", result)
}

// PostJSON sends a JSON POST with an optional Bearer token. Exported for plugins
// that need authenticated requests (e.g., grammar push with plugin token).
// Pass empty token to use the stored plugin token; pass a specific token to override.
func (p *PlatformClient) PostJSON(path string, body any, token string, result any) error {
	effectiveToken := token
	if effectiveToken == "" {
		effectiveToken = p.token
	}
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest("POST", "http://localhost"+path, bodyReader)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if effectiveToken != "" {
		req.Header.Set("Authorization", "Bearer "+effectiveToken)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: HTTP %d: %s", path, resp.StatusCode, string(b))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// GetWorldModel fetches the current world model snapshot.
func (p *PlatformClient) GetWorldModel() (*WorldModel, error) {
	var wm WorldModel
	err := p.get("/v1/native/world-model", &wm)
	return &wm, err
}

// BatchSetFrames sets window frames. If readback is true, waits and returns updated frames.
func (p *PlatformClient) BatchSetFrames(frames []WindowFrame, readback bool) ([]WindowFrame, error) {
	req := struct {
		Frames   []WindowFrame `json:"frames"`
		Readback bool          `json:"readback"`
	}{Frames: frames, Readback: readback}

	var resp struct {
		Results []WindowFrame `json:"results"`
	}
	err := p.post("/v1/native/batch-set-frames", req, &resp)
	return resp.Results, err
}

// BatchSetFramesNoReadback sets frames without reading back.
func (p *PlatformClient) BatchSetFramesNoReadback(frames []WindowFrame) error {
	_, err := p.BatchSetFrames(frames, false)
	return err
}

// RaiseWindow brings a window to front.
func (p *PlatformClient) RaiseWindow(windowID string) error {
	req := struct {
		WindowID string `json:"window_id"`
	}{WindowID: windowID}
	return p.post("/v1/native/raise-window", req, nil)
}

// IsWindowTileable checks if a window can be tiled.
func (p *PlatformClient) IsWindowTileable(windowID string) (bool, error) {
	req := struct {
		WindowIDs []string `json:"window_ids"`
	}{WindowIDs: []string{windowID}}
	var resp struct {
		Results []struct {
			WindowID string `json:"window_id"`
			Tileable bool   `json:"tileable"`
		} `json:"results"`
	}
	err := p.post("/v1/native/batch-is-tileable", req, &resp)
	if err != nil {
		return false, err
	}
	if len(resp.Results) > 0 {
		return resp.Results[0].Tileable, nil
	}
	return false, nil
}

// BatchIsTileable checks tileability for multiple windows.
func (p *PlatformClient) BatchIsTileable(windowIDs []string) (map[string]bool, error) {
	req := struct {
		WindowIDs []string `json:"window_ids"`
	}{WindowIDs: windowIDs}
	var resp struct {
		Results []struct {
			WindowID string `json:"window_id"`
			Tileable bool   `json:"tileable"`
		} `json:"results"`
	}
	err := p.post("/v1/native/batch-is-tileable", req, &resp)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(resp.Results))
	for _, r := range resp.Results {
		result[r.WindowID] = r.Tileable
	}
	return result, nil
}

// ToggleFullscreen toggles native fullscreen for a window.
func (p *PlatformClient) ToggleFullscreen(windowID string) error {
	req := struct {
		WindowID string `json:"window_id"`
	}{WindowID: windowID}
	return p.post("/v1/native/toggle-fullscreen", req, nil)
}

// GetCursor returns the current cursor position.
func (p *PlatformClient) GetCursor() (int, int, error) {
	var resp struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	err := p.get("/v1/native/cursor", &resp)
	return resp.X, resp.Y, err
}

// WarpCursor moves the cursor to (x, y).
func (p *PlatformClient) WarpCursor(x, y int) error {
	req := struct {
		X int `json:"x"`
		Y int `json:"y"`
	}{X: x, Y: y}
	return p.post("/v1/native/warp-cursor", req, nil)
}

// IsAppHidden checks if an app is hidden.
func (p *PlatformClient) IsAppHidden(bundleID string) (bool, error) {
	req := struct {
		BundleID string `json:"bundle_id"`
	}{BundleID: bundleID}
	var resp struct {
		Result bool `json:"result"`
	}
	err := p.post("/v1/native/is-app-hidden", req, &resp)
	return resp.Result, err
}

// UnhideApp unhides an app.
func (p *PlatformClient) UnhideApp(bundleID string) error {
	req := struct {
		BundleID string `json:"bundle_id"`
	}{BundleID: bundleID}
	return p.post("/v1/native/unhide-app", req, nil)
}

// UpdateBorders sends border overlay rects to the actuator for display in the Swift shell.
func (p *PlatformClient) UpdateBorders(frames []WindowFrame) error {
	return p.post("/v1/native/borders", frames, nil)
}

// AudioDevice describes an audio input or output device.
type AudioDevice struct {
	ID             uint32 `json:"id"`
	UID            string `json:"uid"`
	Name           string `json:"name"`
	IsInput        bool   `json:"is_input"`
	IsOutput       bool   `json:"is_output"`
	IsDefaultInput bool   `json:"is_default_input"`
	IsDefaultOutput bool  `json:"is_default_output"`
}

// AudioDeviceList is the response from the audio-devices endpoint.
type AudioDeviceList struct {
	Devices []AudioDevice `json:"devices"`
}

// GetAudioDevices returns all audio devices from the actuator.
func (p *PlatformClient) GetAudioDevices() (*AudioDeviceList, error) {
	var resp AudioDeviceList
	err := p.get("/v1/native/audio-devices", &resp)
	return &resp, err
}

// SetAudioDevice sets the default audio device. deviceType is "output" or "input".
func (p *PlatformClient) SetAudioDevice(uid string, deviceType string) error {
	req := struct {
		UID        string `json:"uid"`
		DeviceType string `json:"device_type"`
	}{UID: uid, DeviceType: deviceType}
	return p.post("/v1/native/set-audio-device", req, nil)
}

// --- Polling ---

// PollBurst requests burst polling (200ms) for the next 2 seconds.
// Call after applying layouts so the centralized poller detects changes quickly.
func (p *PlatformClient) PollBurst() error {
	return p.post("/v1/native/poll-burst", nil, nil)
}

// --- AppleScript ---

// ApplescriptResult is the response from run-applescript.
type ApplescriptResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// RunApplescript executes an AppleScript via the actuator.
func (p *PlatformClient) RunApplescript(script string) (*ApplescriptResult, error) {
	req := struct {
		Script string `json:"script"`
	}{Script: script}
	var resp ApplescriptResult
	err := p.post("/v1/native/run-applescript", req, &resp)
	return &resp, err
}

// --- Command matching ---

// MatchCommands calls POST /v1/commands/match.
func (p *PlatformClient) MatchCommands(words []string, activeTags []string) (*MatchResponse, error) {
	var resp MatchResponse
	err := p.post("/v1/commands/match", MatchRequest{Words: words, ActiveTags: activeTags}, &resp)
	return &resp, err
}

// HasPartial calls POST /v1/commands/has-partial.
func (p *PlatformClient) HasPartial(words []string, activeTags []string) (bool, error) {
	var resp HasPartialResponse
	err := p.post("/v1/commands/has-partial", HasPartialRequest{Words: words, ActiveTags: activeTags}, &resp)
	return resp.HasPartial, err
}

// --- Action execution ---

// Execute calls POST /v1/execute.
func (p *PlatformClient) Execute(action json.RawMessage) (*ExecuteActionResponse, error) {
	var resp ExecuteActionResponse
	err := p.post("/v1/execute", ExecuteActionRequest{Action: action}, &resp)
	return &resp, err
}

// --- Tags ---

// GetTags calls GET /v1/tags.
func (p *PlatformClient) GetTags() ([]string, error) {
	var resp TagsResponse
	err := p.get("/v1/tags", &resp)
	return resp.Tags, err
}

// ModifyTags calls POST /v1/tags.
func (p *PlatformClient) ModifyTags(set, clear []string) ([]string, error) {
	var resp TagsResponse
	err := p.post("/v1/tags", TagsModifyRequest{Set: set, Clear: clear}, &resp)
	return resp.Tags, err
}

// --- Discovery ---

// OpenDiscovery calls POST /v1/discovery/open.
func (p *PlatformClient) OpenDiscovery(req DiscoveryOpenRequest) (*DiscoveryResponse, error) {
	var resp DiscoveryResponse
	err := p.post("/v1/discovery/open", req, &resp)
	return &resp, err
}

// CloseDiscovery calls POST /v1/discovery/close.
func (p *PlatformClient) CloseDiscovery(clearScopedTags, clearPluginTags bool) (*DiscoveryResponse, error) {
	var resp DiscoveryResponse
	err := p.post("/v1/discovery/close", DiscoveryCloseRequest{
		ClearScopedTags: clearScopedTags,
		ClearPluginTags: clearPluginTags,
	}, &resp)
	return &resp, err
}

// GetKeybindsStore fetches the keybinds shared store and returns the keyed-merged data
// as a map of plugin_id → combo_map.
func (p *PlatformClient) GetKeybindsStore() (map[string]map[string]string, error) {
	var resp struct {
		Data map[string]map[string]string `json:"data"`
	}
	err := p.get("/v1/plugins/stores/keybinds", &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// AlphabetEntry describes a single phonetic codeword.
type AlphabetEntry struct {
	Letter   string `json:"letter"`
	Codeword string `json:"codeword"`
	Code     uint16 `json:"code"`
}

// GetAlphabet fetches the shared alphabet store from the actuator.
// Returns nil slice on error (caller should fall back to defaults).
// The actuator merges contributions according to the store's declared merge strategy
// (authoritative for alphabet), so `data` is the merged result directly.
func (p *PlatformClient) GetAlphabet() ([]AlphabetEntry, error) {
	var resp struct {
		Data []AlphabetEntry `json:"data"`
	}
	err := p.get("/v1/plugins/stores/alphabet", &resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no alphabet data in store")
	}
	return resp.Data, nil
}
