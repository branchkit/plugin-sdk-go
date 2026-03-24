package shared

import "encoding/json"

// WorldModel is the full snapshot from the actuator's world model endpoint.
type WorldModel struct {
	Windows        []WindowInfo  `json:"windows"`
	Displays       []DisplayInfo `json:"displays"`
	ActiveWindowID *string       `json:"active_window_id"`
	ActiveApp      *string       `json:"active_app,omitempty"`
}

// WindowInfo describes a single window.
type WindowInfo struct {
	ID      string `json:"id"`
	AppID   string `json:"app_id"`
	AppName string `json:"app_name"`
	Title   string `json:"title"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	W       int    `json:"w"`
	H       int    `json:"h"`
}

// DisplayInfo describes a connected display.
type DisplayInfo struct {
	ID       int `json:"id"`
	X        int `json:"x"`
	Y        int `json:"y"`
	W        int `json:"w"`
	H        int `json:"h"`
	VisibleX int `json:"visible_x"`
	VisibleY int `json:"visible_y"`
	VisibleW int `json:"visible_w"`
	VisibleH int `json:"visible_h"`
}

// Rect is an integer rectangle.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// WindowFrame pairs a window ID with a target frame.
type WindowFrame struct {
	WindowID string `json:"window_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	W        int    `json:"w"`
	H        int    `json:"h"`
}

// HudResponse is the standard response for render-hud hooks.
type HudResponse struct {
	Title       string       `json:"title"`
	Footer      string       `json:"footer"`
	ContentHTML *string      `json:"content_html,omitempty"`
	Sections    []HudSection `json:"sections"`
}

// HudSection is a group of items in a HUD response.
type HudSection struct {
	Title string    `json:"title"`
	Items []HudItem `json:"items"`
}

// HudItem is a single item in a HUD section.
type HudItem struct {
	ID       string  `json:"id"`
	Tag      *string `json:"tag,omitempty"`
	Title    string  `json:"title"`
	Subtitle *string `json:"subtitle,omitempty"`
	Icon     *string `json:"icon,omitempty"`
}

// SettingsResponse is the standard response for render-settings hooks.
type SettingsResponse struct {
	HTML string `json:"html"`
}

// --- Command matching types ---

// MatchRequest is the request for POST /v1/commands/match.
type MatchRequest struct {
	Words      []string `json:"words"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// MatchResponse is the response from POST /v1/commands/match.
type MatchResponse struct {
	Matched       bool            `json:"matched"`
	Action        json.RawMessage `json:"action,omitempty"`
	Args          []string        `json:"args"`
	ConsumedCount int             `json:"consumed_count"`
	SetsTags      []string        `json:"sets_tags"`
	ClearsTags    []string        `json:"clears_tags"`
	OwnerPlugin   *string         `json:"owner_plugin,omitempty"`
}

// HasPartialRequest is the request for POST /v1/commands/has-partial.
type HasPartialRequest struct {
	Words      []string `json:"words"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// HasPartialResponse is the response from POST /v1/commands/has-partial.
type HasPartialResponse struct {
	HasPartial bool `json:"has_partial"`
}

// --- Execute types ---

// ExecuteActionRequest is the request for POST /v1/execute.
type ExecuteActionRequest struct {
	Action json.RawMessage `json:"action"`
}

// ExecuteActionResponse is the response from POST /v1/execute.
type ExecuteActionResponse struct {
	Status      string  `json:"status"`
	Message     *string `json:"message,omitempty"`
	ShellAction *string `json:"shell_action,omitempty"`
	Mode        *string `json:"mode,omitempty"`
}

// --- Tags types ---

// TagsResponse is the response from GET/POST /v1/tags.
type TagsResponse struct {
	Tags []string `json:"tags"`
}

// TagsModifyRequest is the request for POST /v1/tags.
type TagsModifyRequest struct {
	Set   []string `json:"set,omitempty"`
	Clear []string `json:"clear,omitempty"`
}

// --- Discovery types ---

// RenderHudRequest is the request for POST /hooks/render-hud.
type RenderHudRequest struct {
	HudMode  string       `json:"hud_mode"`
	Apps     []AppData    `json:"apps,omitempty"`
	Title    *string      `json:"title,omitempty"`
	Footer   *string      `json:"footer,omitempty"`
	Sections []HudSection `json:"sections,omitempty"`
}

// AppData describes a registered application.
type AppData struct {
	Name     string   `json:"name"`
	BundleID string   `json:"bundle_id"`
	Aliases  []string `json:"aliases"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

// IsEnabled returns true if the app is enabled (defaults to true if nil).
func (a AppData) IsEnabled() bool {
	if a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

// DiscoveryOpenRequest is the request for POST /v1/discovery/open.
type DiscoveryOpenRequest struct {
	RequireTag *string  `json:"require_tag,omitempty"`
	Words      []string `json:"words,omitempty"`
	Countdown  bool     `json:"countdown"`
}

// DiscoveryResponse is the response from POST /v1/discovery/open and /close.
type DiscoveryResponse struct {
	Opened      bool    `json:"opened"`
	ShellAction *string `json:"shell_action,omitempty"`
}

// DiscoveryCloseRequest is the request for POST /v1/discovery/close.
type DiscoveryCloseRequest struct {
	ClearScopedTags bool `json:"clear_scoped_tags"`
	ClearPluginTags bool `json:"clear_plugin_tags"`
}
