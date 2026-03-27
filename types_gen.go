// AUTO-GENERATED from contracts/*.json — do not edit.
// Run: python3 contracts/generate_types.py

package shared

import "encoding/json"

// Ensure json import is used.
var _ json.RawMessage

// ===== Shared types (from components/schemas) =====

// AppData is auto-generated from the OpenRPC spec.
type AppData struct {
	Name string `json:"name"`
	BundleID string `json:"bundle_id"`
	Aliases []string `json:"aliases,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}

// ApplescriptResult is auto-generated from the OpenRPC spec.
type ApplescriptResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	ExitCode int `json:"exit_code"`
}

// AudioDevice is auto-generated from the OpenRPC spec.
type AudioDevice struct {
	ID int `json:"id"`
	UID string `json:"uid"`
	Name string `json:"name"`
	IsInput bool `json:"is_input"`
	IsOutput bool `json:"is_output"`
	IsDefaultInput bool `json:"is_default_input"`
	IsDefaultOutput bool `json:"is_default_output"`
}

// DiscoveryResult is auto-generated from the OpenRPC spec.
type DiscoveryResult struct {
	Opened bool `json:"opened"`
	ShellAction *string `json:"shell_action,omitempty"`
}

// DisplayInfo is auto-generated from the OpenRPC spec.
type DisplayInfo struct {
	ID int `json:"id"`
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
	VisibleX int `json:"visible_x"`
	VisibleY int `json:"visible_y"`
	VisibleW int `json:"visible_w"`
	VisibleH int `json:"visible_h"`
}

// Frame is auto-generated from the OpenRPC spec.
type Frame struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// HudItem is auto-generated from the OpenRPC spec.
type HudItem struct {
	ID string `json:"id"`
	Tag *string `json:"tag,omitempty"`
	Title string `json:"title"`
	Subtitle *string `json:"subtitle,omitempty"`
	Icon *string `json:"icon,omitempty"`
}

// HudResponse is auto-generated from the OpenRPC spec.
type HudResponse struct {
	Title string `json:"title"`
	Footer string `json:"footer"`
	ContentHTML *string `json:"content_html,omitempty"`
	Sections []HudSection `json:"sections,omitempty"`
}

// HudSection is auto-generated from the OpenRPC spec.
type HudSection struct {
	Title string `json:"title"`
	Items []HudItem `json:"items"`
}

// MatchResult is auto-generated from the OpenRPC spec.
type MatchResult struct {
	Matched bool `json:"matched"`
	Action json.RawMessage `json:"action,omitempty"`
	Args []string `json:"args,omitempty"`
	ConsumedCount *int `json:"consumed_count,omitempty"`
	SetsTags []string `json:"sets_tags,omitempty"`
	ClearsTags []string `json:"clears_tags,omitempty"`
	RequiresTags []string `json:"requires_tags,omitempty"`
	OwnerPlugin *string `json:"owner_plugin,omitempty"`
}

// Point is auto-generated from the OpenRPC spec.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// VoiceCommand is auto-generated from the OpenRPC spec.
type VoiceCommand struct {
	Phrase string `json:"phrase"`
	Action json.RawMessage `json:"action"`
	RequiresTags []string `json:"requires_tags,omitempty"`
	SetsTags []string `json:"sets_tags,omitempty"`
	ClearsTags []string `json:"clears_tags,omitempty"`
}

// WindowFrame is auto-generated from the OpenRPC spec.
type WindowFrame struct {
	WindowID string `json:"window_id"`
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// WindowInfo is auto-generated from the OpenRPC spec.
type WindowInfo struct {
	ID string `json:"id"`
	AppID string `json:"app_id"`
	AppName string `json:"app_name"`
	Title string `json:"title"`
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// WorldModel is auto-generated from the OpenRPC spec.
type WorldModel struct {
	Windows []WindowInfo `json:"windows,omitempty"`
	Displays []DisplayInfo `json:"displays,omitempty"`
	ActiveWindowID *string `json:"active_window_id,omitempty"`
	ActiveApp *string `json:"active_app,omitempty"`
}

// ===== Plugin → Actuator request/response types =====

// StorePushRequest is the request type for store.push.
type StorePushRequest struct {
	Name string `json:"name"`
	Data json.RawMessage `json:"data"`
}

// StorePushResponse is the response type for store.push.
type StorePushResponse struct {
	Ok bool `json:"ok"`
}

// StoreGetRequest is the request type for store.get.
type StoreGetRequest struct {
	Name string `json:"name"`
}

// StoreGetResponse is the response type for store.get.
type StoreGetResponse struct {
	Data json.RawMessage `json:"data,omitempty"`
}

// TagsGetResponse is the response type for tags.get.
type TagsGetResponse struct {
	Tags []string `json:"tags"`
}

// TagsModifyRequest is the request type for tags.modify.
type TagsModifyRequest struct {
	Set []string `json:"set,omitempty"`
	Clear []string `json:"clear,omitempty"`
}

// TagsModifyResponse is the response type for tags.modify.
type TagsModifyResponse struct {
	Tags []string `json:"tags"`
}

// CommandsMatchRequest is the request type for commands.match.
type CommandsMatchRequest struct {
	Words []string `json:"words"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// CommandsHasPartialRequest is the request type for commands.has_partial.
type CommandsHasPartialRequest struct {
	Words []string `json:"words"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// CommandsHasPartialResponse is the response type for commands.has_partial.
type CommandsHasPartialResponse struct {
	HasPartial bool `json:"has_partial"`
	NextList *string `json:"next_list,omitempty"`
}

// ListsUpdateRequest is the request type for lists.update.
type ListsUpdateRequest struct {
	Name string `json:"name"`
	Entries json.RawMessage `json:"entries"`
	Merge *bool `json:"merge,omitempty"`
	Label *string `json:"label,omitempty"`
}

// ListsUpdateResponse is the response type for lists.update.
type ListsUpdateResponse struct {
	Name *string `json:"name,omitempty"`
	Entries json.RawMessage `json:"entries,omitempty"`
}

// ListsDeleteRequest is the request type for lists.delete.
type ListsDeleteRequest struct {
	Name string `json:"name"`
}

// ListsDeleteResponse is the response type for lists.delete.
type ListsDeleteResponse struct {
	Ok bool `json:"ok"`
}

// GrammarPushRequest is the request type for grammar.push.
type GrammarPushRequest struct {
	Commands []VoiceCommand `json:"commands"`
}

// GrammarPushResponse is the response type for grammar.push.
type GrammarPushResponse struct {
	Ok bool `json:"ok"`
	Count int `json:"count"`
}

// SelectionSetRequest is the request type for selection.set.
type SelectionSetRequest struct {
	Title *string `json:"title,omitempty"`
	Items []HudItem `json:"items,omitempty"`
}

// SelectionSetResponse is the response type for selection.set.
type SelectionSetResponse struct {
	Ok bool `json:"ok"`
}

// KeybindsRegisterRequest is the request type for keybinds.register.
type KeybindsRegisterRequest struct {
	Snapshot json.RawMessage `json:"snapshot"`
}

// KeybindsRegisterResponse is the response type for keybinds.register.
type KeybindsRegisterResponse struct {
	Ok bool `json:"ok"`
	Count *int `json:"count,omitempty"`
}

// KeyNamesSetRequest is the request type for key_names.set.
type KeyNamesSetRequest struct {
	Names map[string]int `json:"names"`
}

// KeyNamesSetResponse is the response type for key_names.set.
type KeyNamesSetResponse struct {
	Ok bool `json:"ok"`
	Count *int `json:"count,omitempty"`
}

// DiscoveryOpenRequest is the request type for discovery.open.
type DiscoveryOpenRequest struct {
	RequireTag *string `json:"require_tag,omitempty"`
	Words []string `json:"words,omitempty"`
	Countdown *bool `json:"countdown,omitempty"`
}

// DiscoveryCloseRequest is the request type for discovery.close.
type DiscoveryCloseRequest struct {
	ClearScopedTags *bool `json:"clear_scoped_tags,omitempty"`
	ClearPluginTags *bool `json:"clear_plugin_tags,omitempty"`
}

// NativeWorldModelRequest is the request type for native.world_model.
type NativeWorldModelRequest struct {
	OnScreen *bool `json:"on_screen,omitempty"`
}

// NativeBatchSetFramesRequest is the request type for native.batch_set_frames.
type NativeBatchSetFramesRequest struct {
	Frames []WindowFrame `json:"frames"`
	Readback *bool `json:"readback,omitempty"`
}

// NativeBatchSetFramesResponse is the response type for native.batch_set_frames.
type NativeBatchSetFramesResponse struct {
	Results []WindowFrame `json:"results"`
}

// NativeRaiseWindowRequest is the request type for native.raise_window.
type NativeRaiseWindowRequest struct {
	WindowID string `json:"window_id"`
}

// NativeBatchIsTileableRequest is the request type for native.batch_is_tileable.
type NativeBatchIsTileableRequest struct {
	WindowIds []string `json:"window_ids"`
}

// NativeBatchIsTileableResponse is the response type for native.batch_is_tileable.
type NativeBatchIsTileableResponse struct {
	Results []json.RawMessage `json:"results"`
}

// NativeToggleFullscreenRequest is the request type for native.toggle_fullscreen.
type NativeToggleFullscreenRequest struct {
	WindowID string `json:"window_id"`
}

// NativeWarpCursorRequest is the request type for native.warp_cursor.
type NativeWarpCursorRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NativeIsAppHiddenRequest is the request type for native.is_app_hidden.
type NativeIsAppHiddenRequest struct {
	BundleID string `json:"bundle_id"`
}

// NativeIsAppHiddenResponse is the response type for native.is_app_hidden.
type NativeIsAppHiddenResponse struct {
	Result bool `json:"result"`
}

// NativeUnhideAppRequest is the request type for native.unhide_app.
type NativeUnhideAppRequest struct {
	BundleID string `json:"bundle_id"`
}

// NativeBordersRequest is the request type for native.borders.
type NativeBordersRequest struct {
	Frames json.RawMessage `json:"frames"`
}

// NativeRunApplescriptRequest is the request type for native.run_applescript.
type NativeRunApplescriptRequest struct {
	Script string `json:"script"`
}

// NativeAudioDevicesResponse is the response type for native.audio_devices.
type NativeAudioDevicesResponse struct {
	Devices []AudioDevice `json:"devices"`
}

// NativeSetAudioDeviceRequest is the request type for native.set_audio_device.
type NativeSetAudioDeviceRequest struct {
	UID string `json:"uid"`
	DeviceType string `json:"device_type"`
}

// NativeKeyboardLayoutResponse is the response type for native.keyboard_layout.
type NativeKeyboardLayoutResponse struct {
	LayoutID *string `json:"layout_id,omitempty"`
	LayoutName *string `json:"layout_name,omitempty"`
	Mappings json.RawMessage `json:"mappings,omitempty"`
}

// ControlSignalRequest is the request type for control.signal.
type ControlSignalRequest struct {
	Signal string `json:"signal"`
}

// EventsEmitRequest is the request type for events.emit.
type EventsEmitRequest struct {
	EventType string `json:"event_type"`
	Data json.RawMessage `json:"data,omitempty"`
	CorrelationID *string `json:"correlation_id,omitempty"`
}

// ===== Actuator → Plugin request/response types =====

// RenderSettingsRequest is the request type for render_settings.
type RenderSettingsRequest struct {
	TabKey string `json:"tab_key"`
	Search *string `json:"search,omitempty"`
	Apps []AppData `json:"apps,omitempty"`
	Commands json.RawMessage `json:"commands,omitempty"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// RenderSettingsResponse is the response type for render_settings.
type RenderSettingsResponse struct {
	HTML string `json:"html"`
}

// RenderHUDRequest is the request type for render_hud.
type RenderHUDRequest struct {
	HudMode string `json:"hud_mode"`
	Apps []AppData `json:"apps,omitempty"`
	Title *string `json:"title,omitempty"`
	Footer *string `json:"footer,omitempty"`
	Sections []HudSection `json:"sections,omitempty"`
}

// OnActionRequest is the request type for on_action.
type OnActionRequest struct {
	Action string `json:"action"`
	Params json.RawMessage `json:"params,omitempty"`
	ActiveApp *string `json:"active_app,omitempty"`
	ActiveWindowID *string `json:"active_window_id,omitempty"`
}

// OnActionResponse is the response type for on_action.
type OnActionResponse struct {
	Status string `json:"status"`
	ShellAction *string `json:"shell_action,omitempty"`
	ControlMessage *string `json:"control_message,omitempty"`
}

// BuildCommandRegistryRequest is the request type for build_command_registry.
type BuildCommandRegistryRequest struct {
	CommandsByPlugin json.RawMessage `json:"commands_by_plugin"`
	UserCommands []json.RawMessage `json:"user_commands,omitempty"`
}

// BuildCommandRegistryResponse is the response type for build_command_registry.
type BuildCommandRegistryResponse struct {
	PhoneticsCount int `json:"phonetics_count"`
}

// SpeechPipelineRequest is the request type for speech_pipeline.
type SpeechPipelineRequest struct {
	Transcript string `json:"transcript"`
	IsFinal bool `json:"is_final"`
	Mode string `json:"mode"`
}

// SpeechPipelineResponse is the response type for speech_pipeline.
type SpeechPipelineResponse struct {
	Action string `json:"action"`
}

// SpeechOrchestrateRequest is the request type for speech_orchestrate.
type SpeechOrchestrateRequest struct {
	Transcript string `json:"transcript"`
	Words []string `json:"words"`
}

// SpeechOrchestrateResponse is the response type for speech_orchestrate.
type SpeechOrchestrateResponse struct {
	Result string `json:"result"`
	ActionsToExecute []json.RawMessage `json:"actions_to_execute,omitempty"`
}

// CalibrateRequest is the request type for calibrate.
type CalibrateRequest struct {
	Action string `json:"action"`
	Words []string `json:"words,omitempty"`
}

// CalibrateResponse is the response type for calibrate.
type CalibrateResponse struct {
	CalibrationActive bool `json:"calibration_active"`
}
