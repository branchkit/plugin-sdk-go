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

// DispatchActionRequest is for plugins that call the dispatch HTTP endpoint.
// Not part of the JSON-RPC protocol (dispatch is HTTP-only via TCP :21551).
type DispatchActionRequest struct {
	Action json.RawMessage `json:"action"`
}

// DispatchActionResponse is the response from the dispatch endpoint.
type DispatchActionResponse struct {
	Status         string  `json:"status"`
	Message        *string `json:"message,omitempty"`
	ControlMessage *string `json:"control_message,omitempty"`
	Mode           *string `json:"mode,omitempty"`
}
