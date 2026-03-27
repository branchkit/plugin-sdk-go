package shared

import "encoding/json"

// Hand-written type extensions and convenience types.
// Generated types are in types_gen.go.

// Rect is an integer rectangle (SDK convenience, not part of the RPC protocol).
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// IsEnabled returns true if the app is enabled (defaults to true if nil).
func (a AppData) IsEnabled() bool {
	if a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

// ExecuteActionRequest is for plugins that call the execute HTTP endpoint.
// Not part of the JSON-RPC protocol (execute is HTTP-only via TCP :21551).
type ExecuteActionRequest struct {
	Action json.RawMessage `json:"action"`
}

// ExecuteActionResponse is the response from the execute endpoint.
type ExecuteActionResponse struct {
	Status      string  `json:"status"`
	Message     *string `json:"message,omitempty"`
	ShellAction *string `json:"shell_action,omitempty"`
	Mode        *string `json:"mode,omitempty"`
}
