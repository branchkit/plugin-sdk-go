// AUTO-GENERATED from contracts/*.json — do not edit.
// Run: python3 contracts/generate_types.py

package shared

import "encoding/json"

// Ensure json import is used.
var _ json.RawMessage

// ===== Shared types (from components/schemas) =====

// AXElementInfo is auto-generated from the OpenRPC spec.
type AXElementInfo struct {
	Role string `json:"role"`
	Subrole *string `json:"subrole,omitempty"`
	Title *string `json:"title,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
	Description *string `json:"description,omitempty"`
	Position []int `json:"position,omitempty"`
	Size []int `json:"size,omitempty"`
	Enabled bool `json:"enabled"`
	Focused bool `json:"focused"`
	ChildrenCount int `json:"children_count"`
	Actions []string `json:"actions"`
	Attributes []string `json:"attributes"`
	Path []AXPathSegment `json:"path"`
}

// AXElementNode is auto-generated from the OpenRPC spec.
type AXElementNode struct {
	Element AXElementInfo `json:"element"`
	Children []AXElementNode `json:"children"`
}

// AXElementRef is auto-generated from the OpenRPC spec.
type AXElementRef struct {
	Pid int `json:"pid"`
	Path []AXPathSegment `json:"path,omitempty"`
}

// AXPathSegment is auto-generated from the OpenRPC spec.
type AXPathSegment struct {
	Role string `json:"role"`
	Index int `json:"index"`
}

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

// BoolResult is auto-generated from the OpenRPC spec.
type BoolResult struct {
	Result bool `json:"result"`
}

// ClipboardContents is auto-generated from the OpenRPC spec.
type ClipboardContents struct {
	ContentType string `json:"content_type"`
	Text *string `json:"text,omitempty"`
	FileUrls []string `json:"file_urls,omitempty"`
	ImageBase64 *string `json:"image_base64,omitempty"`
	AvailableTypes []string `json:"available_types"`
}

// Command is auto-generated from the OpenRPC spec.
type Command struct {
	Phrase json.RawMessage `json:"phrase"`
	Action json.RawMessage `json:"action"`
	RequiresTags []string `json:"requires_tags,omitempty"`
	SetsTags []string `json:"sets_tags,omitempty"`
	ClearsTags []string `json:"clears_tags,omitempty"`
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

// RunningApp is auto-generated from the OpenRPC spec.
type RunningApp struct {
	Pid int `json:"pid"`
	BundleID *string `json:"bundle_id,omitempty"`
	Name string `json:"name"`
	IsHidden bool `json:"is_hidden"`
	IsActive bool `json:"is_active"`
}

// UserSessionInfo is auto-generated from the OpenRPC spec.
type UserSessionInfo struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	HomeDirectory string `json:"home_directory"`
}

// WindowDetail is auto-generated from the OpenRPC spec.
type WindowDetail struct {
	WindowID string `json:"window_id"`
	Title *string `json:"title,omitempty"`
	Subrole *string `json:"subrole,omitempty"`
	IsMinimized bool `json:"is_minimized"`
	IsFullscreen bool `json:"is_fullscreen"`
	IsFocused bool `json:"is_focused"`
	Alpha *float64 `json:"alpha,omitempty"`
	Bounds json.RawMessage `json:"bounds"`
	DisplayID int `json:"display_id"`
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
	Source *string `json:"source,omitempty"`
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
	ClearScoped *bool `json:"clear_scoped,omitempty"`
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

// CommandsDiscoverRequest is the request type for commands.discover.
type CommandsDiscoverRequest struct {
	Words []string `json:"words,omitempty"`
	RequireTag *string `json:"require_tag,omitempty"`
	ActiveTags []string `json:"active_tags,omitempty"`
}

// CommandsDiscoverResponse is the response type for commands.discover.
type CommandsDiscoverResponse struct {
	Title string `json:"title"`
	Items []json.RawMessage `json:"items"`
}

// CommandsListResponse is the response type for commands.list.
type CommandsListResponse struct {
	Title string `json:"title"`
	Footer string `json:"footer"`
	Sections []json.RawMessage `json:"sections"`
}

// ExecuteRequest is the request type for execute.
type ExecuteRequest struct {
	Action json.RawMessage `json:"action"`
}

// ExecuteResponse is the response type for execute.
type ExecuteResponse struct {
	Status *string `json:"status,omitempty"`
	ShellAction json.RawMessage `json:"shell_action,omitempty"`
}

// SettingsRulesCreateRequest is the request type for settings.rules_create.
type SettingsRulesCreateRequest struct {
	Newrulephrase string `json:"newrulephrase"`
	Newruleactiontype *string `json:"newruleactiontype,omitempty"`
}

// SettingsRulesCreateResponse is the response type for settings.rules_create.
type SettingsRulesCreateResponse struct {
	Ok bool `json:"ok"`
}

// SettingsRulesUpdateRequest is the request type for settings.rules_update.
type SettingsRulesUpdateRequest struct {
	Canonical string `json:"canonical"`
	Newrulephrase string `json:"newrulephrase"`
}

// SettingsRulesUpdateResponse is the response type for settings.rules_update.
type SettingsRulesUpdateResponse struct {
	Ok bool `json:"ok"`
}

// ListsGetRequest is the request type for lists.get.
type ListsGetRequest struct {
	Name string `json:"name"`
}

// ListsGetResponse is the response type for lists.get.
type ListsGetResponse struct {
	Name *string `json:"name,omitempty"`
	Entries json.RawMessage `json:"entries,omitempty"`
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

// HUDHideRequest is the request type for hud.hide.
type HUDHideRequest struct {
	Channel string `json:"channel"`
}

// HUDHideResponse is the response type for hud.hide.
type HUDHideResponse struct {
	Ok bool `json:"ok"`
}

// HUDPushRequest is the request type for hud.push.
type HUDPushRequest struct {
	Channel string `json:"channel"`
	Fragments []json.RawMessage `json:"fragments"`
}

// HUDPushResponse is the response type for hud.push.
type HUDPushResponse struct {
	Ok bool `json:"ok"`
}

// HUDCreateChannelRequest is the request type for hud.create_channel.
type HUDCreateChannelRequest struct {
	Channel string `json:"channel"`
	Anchor string `json:"anchor,omitempty"`
	Width *int `json:"width,omitempty"`
	MinHeight *int `json:"min_height,omitempty"`
	AcceptsInput *bool `json:"accepts_input,omitempty"`
	Description *string `json:"description,omitempty"`
}

// HUDCreateChannelResponse is the response type for hud.create_channel.
type HUDCreateChannelResponse struct {
	Ok bool `json:"ok"`
	Error *string `json:"error,omitempty"`
}

// HUDRemoveChannelRequest is the request type for hud.remove_channel.
type HUDRemoveChannelRequest struct {
	Channel string `json:"channel"`
}

// HUDRemoveChannelResponse is the response type for hud.remove_channel.
type HUDRemoveChannelResponse struct {
	Ok bool `json:"ok"`
	Removed *bool `json:"removed,omitempty"`
}

// HUDSetSizeRequest is the request type for hud.set_size.
type HUDSetSizeRequest struct {
	Channel string `json:"channel"`
	Height int `json:"height"`
}

// HUDSetSizeResponse is the response type for hud.set_size.
type HUDSetSizeResponse struct {
	Ok bool `json:"ok"`
}

// HUDShowRequest is the request type for hud.show.
type HUDShowRequest struct {
	Channel string `json:"channel"`
}

// HUDShowResponse is the response type for hud.show.
type HUDShowResponse struct {
	Ok bool `json:"ok"`
}

// SessionEndCleanupResponse is the response type for session.end_cleanup.
type SessionEndCleanupResponse struct {
	Ok *bool `json:"ok,omitempty"`
	ShellAction *string `json:"shell_action,omitempty"`
	ResetEngine *bool `json:"reset_engine,omitempty"`
}

// EventsAppendRequest is the request type for events.append.
type EventsAppendRequest struct {
	SessionID *string `json:"session_id,omitempty"`
	EventType string `json:"event_type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// EventsAppendResponse is the response type for events.append.
type EventsAppendResponse struct {
	Ok bool `json:"ok"`
}

// GrammarPushRequest is the request type for grammar.push.
type GrammarPushRequest struct {
	Commands []Command `json:"commands"`
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

// SelectionPickRequest is the request type for selection.pick.
type SelectionPickRequest struct {
	Index int `json:"index"`
}

// SelectionPickResponse is the response type for selection.pick.
type SelectionPickResponse struct {
	Ok *bool `json:"ok,omitempty"`
	ItemID *string `json:"item_id,omitempty"`
	ResetEngine *bool `json:"reset_engine,omitempty"`
	ShellAction *string `json:"shell_action,omitempty"`
}

// MatchAliasesSetRequest is the request type for match_aliases.set.
type MatchAliasesSetRequest struct {
	Aliases map[string]string `json:"aliases"`
}

// MatchAliasesSetResponse is the response type for match_aliases.set.
type MatchAliasesSetResponse struct {
	Ok *bool `json:"ok,omitempty"`
	Count *int `json:"count,omitempty"`
}

// MatchAliasesGetResponse is the response type for match_aliases.get.
type MatchAliasesGetResponse struct {
	Aliases map[string]string `json:"aliases,omitempty"`
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

// NativeRunningAppsResponse is the response type for native.running_apps.
type NativeRunningAppsResponse struct {
	Apps []RunningApp `json:"apps,omitempty"`
}

// NativeFrontmostAppResponse is the response type for native.frontmost_app.
type NativeFrontmostAppResponse struct {
	App *RunningApp `json:"app,omitempty"`
}

// NativeQuitAppRequest is the request type for native.quit_app.
type NativeQuitAppRequest struct {
	BundleID string `json:"bundle_id"`
}

// NativeQuitAppResponse is the response type for native.quit_app.
type NativeQuitAppResponse struct {
	Result *bool `json:"result,omitempty"`
}

// NativeForceQuitAppRequest is the request type for native.force_quit_app.
type NativeForceQuitAppRequest struct {
	BundleID string `json:"bundle_id"`
}

// NativeForceQuitAppResponse is the response type for native.force_quit_app.
type NativeForceQuitAppResponse struct {
	Result *bool `json:"result,omitempty"`
}

// NativeHideAppRequest is the request type for native.hide_app.
type NativeHideAppRequest struct {
	BundleID string `json:"bundle_id"`
}

// NativeActivateAppRequest is the request type for native.activate_app.
type NativeActivateAppRequest struct {
	BundleID string `json:"bundle_id"`
	AllWindows *bool `json:"all_windows,omitempty"`
}

// NativeMinimizeWindowRequest is the request type for native.minimize_window.
type NativeMinimizeWindowRequest struct {
	WindowID string `json:"window_id"`
}

// NativeUnminimizeWindowRequest is the request type for native.unminimize_window.
type NativeUnminimizeWindowRequest struct {
	WindowID string `json:"window_id"`
}

// NativeCloseWindowRequest is the request type for native.close_window.
type NativeCloseWindowRequest struct {
	WindowID string `json:"window_id"`
}

// NativeGetWindowInfoRequest is the request type for native.get_window_info.
type NativeGetWindowInfoRequest struct {
	WindowID string `json:"window_id"`
}

// NativeVolumeResponse is the response type for native.volume.
type NativeVolumeResponse struct {
	Volume *float64 `json:"volume,omitempty"`
	IsMuted *bool `json:"is_muted,omitempty"`
}

// NativeSetVolumeRequest is the request type for native.set_volume.
type NativeSetVolumeRequest struct {
	Volume float64 `json:"volume"`
}

// NativeMuteRequest is the request type for native.mute.
type NativeMuteRequest struct {
	Muted bool `json:"muted"`
}

// NativeDarkModeResponse is the response type for native.dark_mode.
type NativeDarkModeResponse struct {
	IsDark *bool `json:"is_dark,omitempty"`
}

// NativeSetDarkModeRequest is the request type for native.set_dark_mode.
type NativeSetDarkModeRequest struct {
	Dark bool `json:"dark"`
}

// InputDragRequest is the request type for input.drag.
type InputDragRequest struct {
	FromX int `json:"from_x"`
	FromY int `json:"from_y"`
	ToX int `json:"to_x"`
	ToY int `json:"to_y"`
	DurationMs *int `json:"duration_ms,omitempty"`
}

// InputClipboardReadRequest is the request type for input.clipboard_read.
type InputClipboardReadRequest struct {
	ContentType string `json:"content_type"`
}

// InputClipboardWriteRequest is the request type for input.clipboard_write.
type InputClipboardWriteRequest struct {
	ContentType string `json:"content_type"`
	Data string `json:"data"`
}

// InputClipboardWriteResponse is the response type for input.clipboard_write.
type InputClipboardWriteResponse struct {
	Ok *bool `json:"ok,omitempty"`
}

// NativeBatteryResponse is the response type for native.battery.
type NativeBatteryResponse struct {
	Level *float64 `json:"level,omitempty"`
	IsCharging *bool `json:"is_charging,omitempty"`
	IsPluggedIn *bool `json:"is_plugged_in,omitempty"`
	TimeRemainingMinutes *int `json:"time_remaining_minutes,omitempty"`
	IsPresent *bool `json:"is_present,omitempty"`
}

// NativeWifiResponse is the response type for native.wifi.
type NativeWifiResponse struct {
	Ssid *string `json:"ssid,omitempty"`
	Bssid *string `json:"bssid,omitempty"`
	Rssi *int `json:"rssi,omitempty"`
	IsConnected *bool `json:"is_connected,omitempty"`
	IsEnabled *bool `json:"is_enabled,omitempty"`
}

// NativePlaySoundRequest is the request type for native.play_sound.
type NativePlaySoundRequest struct {
	Name string `json:"name"`
}

// NativeSpeakRequest is the request type for native.speak.
type NativeSpeakRequest struct {
	Text string `json:"text"`
	Voice *string `json:"voice,omitempty"`
	Rate *float64 `json:"rate,omitempty"`
}

// NativeDisplaysResponse is the response type for native.displays.
type NativeDisplaysResponse struct {
	Displays []json.RawMessage `json:"displays,omitempty"`
}

// NativeBrightnessRequest is the request type for native.brightness.
type NativeBrightnessRequest struct {
	DisplayID *int `json:"display_id,omitempty"`
}

// NativeBrightnessResponse is the response type for native.brightness.
type NativeBrightnessResponse struct {
	Brightness *float64 `json:"brightness,omitempty"`
}

// NativeSetBrightnessRequest is the request type for native.set_brightness.
type NativeSetBrightnessRequest struct {
	Brightness float64 `json:"brightness"`
	DisplayID *int `json:"display_id,omitempty"`
}

// NativeScreenshotRequest is the request type for native.screenshot.
type NativeScreenshotRequest struct {
	WindowID *string `json:"window_id,omitempty"`
	DisplayID *int `json:"display_id,omitempty"`
	Region json.RawMessage `json:"region,omitempty"`
}

// NativeScreenshotResponse is the response type for native.screenshot.
type NativeScreenshotResponse struct {
	ImageBase64 *string `json:"image_base64,omitempty"`
	Format *string `json:"format,omitempty"`
}

// NativeMenuBarRequest is the request type for native.menu_bar.
type NativeMenuBarRequest struct {
	Pid int `json:"pid"`
}

// NativeMenuBarResponse is the response type for native.menu_bar.
type NativeMenuBarResponse struct {
	Items []json.RawMessage `json:"items,omitempty"`
}

// NativeClickMenuItemRequest is the request type for native.click_menu_item.
type NativeClickMenuItemRequest struct {
	Pid int `json:"pid"`
	Path []string `json:"path"`
}

// NativeClickMenuItemResponse is the response type for native.click_menu_item.
type NativeClickMenuItemResponse struct {
	Result *bool `json:"result,omitempty"`
}

// NativeListSpacesResponse is the response type for native.list_spaces.
type NativeListSpacesResponse struct {
	Spaces []json.RawMessage `json:"spaces,omitempty"`
}

// NativeActiveSpaceResponse is the response type for native.active_space.
type NativeActiveSpaceResponse struct {
	Active []json.RawMessage `json:"active,omitempty"`
}

// NativeMoveWindowToSpaceRequest is the request type for native.move_window_to_space.
type NativeMoveWindowToSpaceRequest struct {
	WindowID string `json:"window_id"`
	SpaceID int `json:"space_id"`
}

// NativeMoveWindowToSpaceResponse is the response type for native.move_window_to_space.
type NativeMoveWindowToSpaceResponse struct {
	Result *bool `json:"result,omitempty"`
}

// NativeMoveWindowToDisplayRequest is the request type for native.move_window_to_display.
type NativeMoveWindowToDisplayRequest struct {
	WindowID string `json:"window_id"`
	DisplayID int `json:"display_id"`
}

// NativeCaptureWindowRequest is the request type for native.capture_window.
type NativeCaptureWindowRequest struct {
	WindowID string `json:"window_id"`
}

// NativeCaptureWindowResponse is the response type for native.capture_window.
type NativeCaptureWindowResponse struct {
	ImageBase64 *string `json:"image_base64,omitempty"`
	Format *string `json:"format,omitempty"`
}

// NativeSetWindowAlphaRequest is the request type for native.set_window_alpha.
type NativeSetWindowAlphaRequest struct {
	WindowID string `json:"window_id"`
	Alpha float64 `json:"alpha"`
}

// NativeAppIconRequest is the request type for native.app_icon.
type NativeAppIconRequest struct {
	BundleID string `json:"bundle_id"`
	Size *int `json:"size,omitempty"`
}

// NativeAppIconResponse is the response type for native.app_icon.
type NativeAppIconResponse struct {
	ImageBase64 *string `json:"image_base64,omitempty"`
	Format *string `json:"format,omitempty"`
}

// InputDoubleClickRequest is the request type for input.double_click.
type InputDoubleClickRequest struct {
	X *int `json:"x,omitempty"`
	Y *int `json:"y,omitempty"`
}

// InputRightClickRequest is the request type for input.right_click.
type InputRightClickRequest struct {
	X *int `json:"x,omitempty"`
	Y *int `json:"y,omitempty"`
}

// InputSwitchInputSourceRequest is the request type for input.switch_input_source.
type InputSwitchInputSourceRequest struct {
	SourceID string `json:"source_id"`
}

// InputSwitchInputSourceResponse is the response type for input.switch_input_source.
type InputSwitchInputSourceResponse struct {
	Result *bool `json:"result,omitempty"`
}

// InputListInputSourcesResponse is the response type for input.list_input_sources.
type InputListInputSourcesResponse struct {
	Sources []json.RawMessage `json:"sources,omitempty"`
}

// NativeDndResponse is the response type for native.dnd.
type NativeDndResponse struct {
	Enabled *bool `json:"enabled,omitempty"`
	FocusName *string `json:"focus_name,omitempty"`
}

// NativeSetDndRequest is the request type for native.set_dnd.
type NativeSetDndRequest struct {
	Enabled bool `json:"enabled"`
}

// NativeBluetoothDevicesResponse is the response type for native.bluetooth_devices.
type NativeBluetoothDevicesResponse struct {
	Devices []json.RawMessage `json:"devices,omitempty"`
}

// NativePreventSleepRequest is the request type for native.prevent_sleep.
type NativePreventSleepRequest struct {
	Reason *string `json:"reason,omitempty"`
	AssertionID *string `json:"assertion_id,omitempty"`
}

// NativePreventSleepResponse is the response type for native.prevent_sleep.
type NativePreventSleepResponse struct {
	AssertionID *string `json:"assertion_id,omitempty"`
}

// NativeSystemUptimeResponse is the response type for native.system_uptime.
type NativeSystemUptimeResponse struct {
	UptimeSeconds *float64 `json:"uptime_seconds,omitempty"`
	Formatted *string `json:"formatted,omitempty"`
}

// NativeColorAtPointRequest is the request type for native.color_at_point.
type NativeColorAtPointRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// NativeColorAtPointResponse is the response type for native.color_at_point.
type NativeColorAtPointResponse struct {
	R *int `json:"r,omitempty"`
	G *int `json:"g,omitempty"`
	B *int `json:"b,omitempty"`
	A *int `json:"a,omitempty"`
	Hex *string `json:"hex,omitempty"`
}

// NativeCursorInfoResponse is the response type for native.cursor_info.
type NativeCursorInfoResponse struct {
	CursorType *string `json:"cursor_type,omitempty"`
	X *int `json:"x,omitempty"`
	Y *int `json:"y,omitempty"`
}

// NativeSpotlightRequest is the request type for native.spotlight.
type NativeSpotlightRequest struct {
	Query string `json:"query"`
	Scope []string `json:"scope,omitempty"`
	Limit *int `json:"limit,omitempty"`
}

// NativeSpotlightResponse is the response type for native.spotlight.
type NativeSpotlightResponse struct {
	Results []json.RawMessage `json:"results,omitempty"`
}

// NativeTrashRequest is the request type for native.trash.
type NativeTrashRequest struct {
	Path string `json:"path"`
}

// NativeTrashResponse is the response type for native.trash.
type NativeTrashResponse struct {
	Result *bool `json:"result,omitempty"`
}

// NativeFileTagsRequest is the request type for native.file_tags.
type NativeFileTagsRequest struct {
	Path string `json:"path"`
	Tags []string `json:"tags,omitempty"`
}

// NativeFileTagsResponse is the response type for native.file_tags.
type NativeFileTagsResponse struct {
	Path *string `json:"path,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// NativeRevealInFinderRequest is the request type for native.reveal_in_finder.
type NativeRevealInFinderRequest struct {
	Path string `json:"path"`
}

// NativeQuickLookRequest is the request type for native.quick_look.
type NativeQuickLookRequest struct {
	Path string `json:"path"`
	Size *int `json:"size,omitempty"`
}

// NativeQuickLookResponse is the response type for native.quick_look.
type NativeQuickLookResponse struct {
	ImageBase64 *string `json:"image_base64,omitempty"`
	Format *string `json:"format,omitempty"`
}

// NativeSelectedFinderItemsResponse is the response type for native.selected_finder_items.
type NativeSelectedFinderItemsResponse struct {
	Paths []string `json:"paths,omitempty"`
}

// NativeNotifyRequest is the request type for native.notify.
type NativeNotifyRequest struct {
	Title string `json:"title"`
	Body *string `json:"body,omitempty"`
	Subtitle *string `json:"subtitle,omitempty"`
	Sound *string `json:"sound,omitempty"`
}

// NativeNotifyResponse is the response type for native.notify.
type NativeNotifyResponse struct {
	ID *string `json:"id,omitempty"`
}

// NativeListNotificationsResponse is the response type for native.list_notifications.
type NativeListNotificationsResponse struct {
	Notifications []json.RawMessage `json:"notifications,omitempty"`
}

// NativeDismissNotificationRequest is the request type for native.dismiss_notification.
type NativeDismissNotificationRequest struct {
	ID string `json:"id"`
}

// NativeDefaultBrowserResponse is the response type for native.default_browser.
type NativeDefaultBrowserResponse struct {
	BundleID *string `json:"bundle_id,omitempty"`
}

// NativeLoginItemsResponse is the response type for native.login_items.
type NativeLoginItemsResponse struct {
	Items []json.RawMessage `json:"items,omitempty"`
}

// NativeClipboardChangeCountResponse is the response type for native.clipboard_change_count.
type NativeClipboardChangeCountResponse struct {
	Count *int `json:"count,omitempty"`
}

// InputClipboardReadAllResponse is the response type for input.clipboard_read_all.
type InputClipboardReadAllResponse struct {
	Items []json.RawMessage `json:"items,omitempty"`
}

// InputClipboardWriteItemsRequest is the request type for input.clipboard_write_items.
type InputClipboardWriteItemsRequest struct {
	Items []json.RawMessage `json:"items"`
}

// NativeAxElementAtPointRequest is the request type for native.ax_element_at_point.
type NativeAxElementAtPointRequest struct {
	Pid int `json:"pid"`
	X int `json:"x"`
	Y int `json:"y"`
}

// NativeAxElementTreeRequest is the request type for native.ax_element_tree.
type NativeAxElementTreeRequest struct {
	Element AXElementRef `json:"element"`
	Depth *int `json:"depth,omitempty"`
}

// NativeAxReadAttributesRequest is the request type for native.ax_read_attributes.
type NativeAxReadAttributesRequest struct {
	Element AXElementRef `json:"element"`
	Attributes []string `json:"attributes"`
}

// NativeAxSetAttributeRequest is the request type for native.ax_set_attribute.
type NativeAxSetAttributeRequest struct {
	Element AXElementRef `json:"element"`
	Attribute string `json:"attribute"`
	Value json.RawMessage `json:"value"`
}

// NativeAxPerformActionRequest is the request type for native.ax_perform_action.
type NativeAxPerformActionRequest struct {
	Element AXElementRef `json:"element"`
	Action string `json:"action"`
}

// NativeAxObserveRequest is the request type for native.ax_observe.
type NativeAxObserveRequest struct {
	Pid int `json:"pid"`
	Notifications []string `json:"notifications"`
}

// NativeAxObserveResponse is the response type for native.ax_observe.
type NativeAxObserveResponse struct {
	SubscriptionID *string `json:"subscription_id,omitempty"`
}

// NativeAxUnobserveRequest is the request type for native.ax_unobserve.
type NativeAxUnobserveRequest struct {
	SubscriptionID string `json:"subscription_id"`
}

// NativeObserveWindowsRequest is the request type for native.observe_windows.
type NativeObserveWindowsRequest struct {
	Pid int `json:"pid"`
}

// NativeObserveWindowsResponse is the response type for native.observe_windows.
type NativeObserveWindowsResponse struct {
	SubscriptionID *string `json:"subscription_id,omitempty"`
}

// NativeUnobserveWindowsRequest is the request type for native.unobserve_windows.
type NativeUnobserveWindowsRequest struct {
	SubscriptionID string `json:"subscription_id"`
}

// NativeSetWindowLevelRequest is the request type for native.set_window_level.
type NativeSetWindowLevelRequest struct {
	WindowID string `json:"window_id"`
	Level string `json:"level"`
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

// InputTypeTextRequest is the request type for input.type_text.
type InputTypeTextRequest struct {
	Text string `json:"text"`
}

// InputPressKeyRequest is the request type for input.press_key.
type InputPressKeyRequest struct {
	Code *int `json:"code,omitempty"`
	Name *string `json:"name,omitempty"`
	Modifiers []string `json:"modifiers,omitempty"`
}

// InputRawKeyRequest is the request type for input.raw_key.
type InputRawKeyRequest struct {
	Code int `json:"code"`
	Direction string `json:"direction"`
}

// InputClickRequest is the request type for input.click.
type InputClickRequest struct {
	Button string `json:"button,omitempty"`
}

// InputScrollRequest is the request type for input.scroll.
type InputScrollRequest struct {
	Direction string `json:"direction"`
	Amount *int `json:"amount,omitempty"`
}

// InputMouseButtonRequest is the request type for input.mouse_button.
type InputMouseButtonRequest struct {
	Button string `json:"button,omitempty"`
	Direction string `json:"direction"`
}

// InputClipboardActionRequest is the request type for input.clipboard_action.
type InputClipboardActionRequest struct {
	Action string `json:"action"`
	Text *string `json:"text,omitempty"`
}

// NativeLaunchAppRequest is the request type for native.launch_app.
type NativeLaunchAppRequest struct {
	BundleID string `json:"bundle_id"`
	NewInstance *bool `json:"new_instance,omitempty"`
}

// NativeOpenTargetRequest is the request type for native.open_target.
type NativeOpenTargetRequest struct {
	Target string `json:"target"`
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
