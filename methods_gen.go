// AUTO-GENERATED from contracts/actuator-rpc.json — do not edit.
// Run: python3 contracts/generate_methods.py

package shared

import "encoding/json"

// Ensure json import is used.
var _ json.RawMessage

// StorePush write data to a shared store.
func (p *Plugin) StorePush(name string, data json.RawMessage) error {
	req := &StorePushRequest{
		Name: name,
		Data: data,
	}
	return p.Call(MethodStorePush, req, nil)
}

// StoreGet read data from a shared store.
func (p *Plugin) StoreGet(name string) (*StoreGetResponse, error) {
	req := &StoreGetRequest{
		Name: name,
	}
	var result StoreGetResponse
	err := p.Call(MethodStoreGet, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TagsGet get all active tags.
func (p *Plugin) TagsGet() ([]string, error) {
	var result struct {
		Tags []string `json:"tags"`
	}
	err := p.Call(MethodTagsGet, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Tags, nil
}

// TagsModify set and/or clear active tags.
func (p *Plugin) TagsModify(set []string, clear []string, clearScoped *bool) ([]string, error) {
	req := &TagsModifyRequest{
		Set: set,
		Clear: clear,
		ClearScoped: clearScoped,
	}
	var result struct {
		Tags []string `json:"tags"`
	}
	err := p.Call(MethodTagsModify, req, &result)
	if err != nil {
		return nil, err
	}
	return result.Tags, nil
}

// CommandsMatch match words against the command registry.
func (p *Plugin) CommandsMatch(words []string, activeTags []string) (*MatchResult, error) {
	req := &CommandsMatchRequest{
		Words: words,
		ActiveTags: activeTags,
	}
	var result MatchResult
	err := p.Call(MethodCommandsMatch, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CommandsHasPartial check if words partially match any command (prefix matching).
func (p *Plugin) CommandsHasPartial(words []string, activeTags []string) (*CommandsHasPartialResponse, error) {
	req := &CommandsHasPartialRequest{
		Words: words,
		ActiveTags: activeTags,
	}
	var result CommandsHasPartialResponse
	err := p.Call(MethodCommandsHasPartial, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CommandsDiscover discover available commands or next tokens for a partial match.
func (p *Plugin) CommandsDiscover(words []string, requireTag *string, activeTags []string) (*CommandsDiscoverResponse, error) {
	req := &CommandsDiscoverRequest{
		Words: words,
		RequireTag: requireTag,
		ActiveTags: activeTags,
	}
	var result CommandsDiscoverResponse
	err := p.Call(MethodCommandsDiscover, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CommandsList return all commands grouped by category for HUD display.
func (p *Plugin) CommandsList() (*CommandsListResponse, error) {
	var result CommandsListResponse
	err := p.Call(MethodCommandsList, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Execute execute an action (shortcut, shell command, type text, etc.).
func (p *Plugin) Execute(action json.RawMessage) (*ExecuteResponse, error) {
	req := &ExecuteRequest{
		Action: action,
	}
	var result ExecuteResponse
	err := p.Call(MethodExecute, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SettingsRulesCreate create a new user voice command rule.
func (p *Plugin) SettingsRulesCreate(newrulephrase string, newruleactiontype *string) error {
	req := &SettingsRulesCreateRequest{
		Newrulephrase: newrulephrase,
		Newruleactiontype: newruleactiontype,
	}
	return p.Call(MethodSettingsRulesCreate, req, nil)
}

// SettingsRulesUpdate update an existing user voice command rule.
func (p *Plugin) SettingsRulesUpdate(canonical string, newrulephrase string) error {
	req := &SettingsRulesUpdateRequest{
		Canonical: canonical,
		Newrulephrase: newrulephrase,
	}
	return p.Call(MethodSettingsRulesUpdate, req, nil)
}

// ListsGet retrieve a named list by name.
func (p *Plugin) ListsGet(name string) (*ListsGetResponse, error) {
	req := &ListsGetRequest{
		Name: name,
	}
	var result ListsGetResponse
	err := p.Call(MethodListsGet, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListsUpdate create or update a named list used in command matching.
func (p *Plugin) ListsUpdate(name string, entries json.RawMessage, merge *bool, label *string) (*ListsUpdateResponse, error) {
	req := &ListsUpdateRequest{
		Name: name,
		Entries: entries,
		Merge: merge,
		Label: label,
	}
	var result ListsUpdateResponse
	err := p.Call(MethodListsUpdate, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListsDelete delete a named list.
func (p *Plugin) ListsDelete(name string) error {
	req := &ListsDeleteRequest{
		Name: name,
	}
	return p.Call(MethodListsDelete, req, nil)
}

// HUDHide hide a HUD channel's window.
func (p *Plugin) HUDHide(channel string) error {
	req := &HUDHideRequest{
		Channel: channel,
	}
	return p.Call(MethodHudHide, req, nil)
}

// HUDPush push HTML fragments to a named HUD channel.
func (p *Plugin) HUDPush(channel string, fragments []json.RawMessage) error {
	req := &HUDPushRequest{
		Channel: channel,
		Fragments: fragments,
	}
	return p.Call(MethodHudPush, req, nil)
}

// HUDCreateChannel create a new HUD broadcast channel at runtime.
func (p *Plugin) HUDCreateChannel(channel string, anchor string, width *int, minHeight *int, acceptsInput *bool, description *string) (*HUDCreateChannelResponse, error) {
	req := &HUDCreateChannelRequest{
		Channel: channel,
		Anchor: anchor,
		Width: width,
		MinHeight: minHeight,
		AcceptsInput: acceptsInput,
		Description: description,
	}
	var result HUDCreateChannelResponse
	err := p.Call(MethodHudCreateChannel, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HUDRemoveChannel remove a HUD broadcast channel.
func (p *Plugin) HUDRemoveChannel(channel string) (*HUDRemoveChannelResponse, error) {
	req := &HUDRemoveChannelRequest{
		Channel: channel,
	}
	var result HUDRemoveChannelResponse
	err := p.Call(MethodHudRemoveChannel, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HUDSetSize report actual rendered size for a HUD channel window.
func (p *Plugin) HUDSetSize(channel string, height int) error {
	req := &HUDSetSizeRequest{
		Channel: channel,
		Height: height,
	}
	return p.Call(MethodHudSetSize, req, nil)
}

// HUDShow show a HUD channel's window.
func (p *Plugin) HUDShow(channel string) error {
	req := &HUDShowRequest{
		Channel: channel,
	}
	return p.Call(MethodHudShow, req, nil)
}

// SessionEndCleanup perform session-end cleanup: reset plugin tags, close discovery, emit session ended event.
func (p *Plugin) SessionEndCleanup() (*SessionEndCleanupResponse, error) {
	var result SessionEndCleanupResponse
	err := p.Call(MethodSessionEndCleanup, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// EventsAppend append an event to the structured event log.
func (p *Plugin) EventsAppend(sessionID *string, eventType string, data json.RawMessage) error {
	req := &EventsAppendRequest{
		SessionID: sessionID,
		EventType: eventType,
		Data: data,
	}
	return p.Call(MethodEventsAppend, req, nil)
}

// GrammarPush push voice commands to the matching engine.
func (p *Plugin) GrammarPush(commands []Command) (*GrammarPushResponse, error) {
	req := &GrammarPushRequest{
		Commands: commands,
	}
	var result GrammarPushResponse
	err := p.Call(MethodGrammarPush, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SelectionSet show the selection HUD with items for the user to pick from.
func (p *Plugin) SelectionSet(title *string, items []HudItem) error {
	req := &SelectionSetRequest{
		Title: title,
		Items: items,
	}
	return p.Call(MethodSelectionSet, req, nil)
}

// SelectionPick resolve a selection pick by index — clears selection state, emits event, closes HUD.
func (p *Plugin) SelectionPick(index int) (*SelectionPickResponse, error) {
	req := &SelectionPickRequest{
		Index: index,
	}
	var result SelectionPickResponse
	err := p.Call(MethodSelectionPick, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// MatchAliasesSet register word aliases for command matching (e.g., 'three' → '3'). Aliases are merged with existing entries..
func (p *Plugin) MatchAliasesSet(aliases map[string]string) (*MatchAliasesSetResponse, error) {
	req := &MatchAliasesSetRequest{
		Aliases: aliases,
	}
	var result MatchAliasesSetResponse
	err := p.Call(MethodMatchAliasesSet, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// MatchAliasesGet get all registered match aliases.
func (p *Plugin) MatchAliasesGet() (*MatchAliasesGetResponse, error) {
	var result MatchAliasesGetResponse
	err := p.Call(MethodMatchAliasesGet, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// KeybindsRegister register keybind snapshot with the platform (caches and sends to Swift shell).
func (p *Plugin) KeybindsRegister(snapshot json.RawMessage) (*KeybindsRegisterResponse, error) {
	req := &KeybindsRegisterRequest{
		Snapshot: snapshot,
	}
	var result KeybindsRegisterResponse
	err := p.Call(MethodKeybindsRegister, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// KeyNamesSet set the key name → keycode mapping for keyboard input simulation.
func (p *Plugin) KeyNamesSet(names map[string]int) (*KeyNamesSetResponse, error) {
	req := &KeyNamesSetRequest{
		Names: names,
	}
	var result KeyNamesSetResponse
	err := p.Call(MethodKeyNamesSet, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeWorldModel get a snapshot of all windows and displays.
func (p *Plugin) NativeWorldModel(onScreen *bool) (*WorldModel, error) {
	req := &NativeWorldModelRequest{
		OnScreen: onScreen,
	}
	var result WorldModel
	err := p.Call(MethodNativeWorldModel, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeBatchSetFrames set positions/sizes for multiple windows.
func (p *Plugin) NativeBatchSetFrames(frames []WindowFrame, readback *bool) ([]WindowFrame, error) {
	req := &NativeBatchSetFramesRequest{
		Frames: frames,
		Readback: readback,
	}
	var result struct {
		Results []WindowFrame `json:"results"`
	}
	err := p.Call(MethodNativeBatchSetFrames, req, &result)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// NativeRaiseWindow raise a window to the front.
func (p *Plugin) NativeRaiseWindow(windowID string) error {
	req := &NativeRaiseWindowRequest{
		WindowID: windowID,
	}
	return p.Call(MethodNativeRaiseWindow, req, nil)
}

// NativeBatchIsTileable check which windows can be tiled.
func (p *Plugin) NativeBatchIsTileable(windowIds []string) ([]json.RawMessage, error) {
	req := &NativeBatchIsTileableRequest{
		WindowIds: windowIds,
	}
	var result struct {
		Results []json.RawMessage `json:"results"`
	}
	err := p.Call(MethodNativeBatchIsTileable, req, &result)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// NativeToggleFullscreen toggle native fullscreen for a window.
func (p *Plugin) NativeToggleFullscreen(windowID string) error {
	req := &NativeToggleFullscreenRequest{
		WindowID: windowID,
	}
	return p.Call(MethodNativeToggleFullscreen, req, nil)
}

// NativeCursor get the current cursor position.
func (p *Plugin) NativeCursor() (*Point, error) {
	var result Point
	err := p.Call(MethodNativeCursor, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeWarpCursor move the cursor to a position.
func (p *Plugin) NativeWarpCursor(x int, y int) error {
	req := &NativeWarpCursorRequest{
		X: x,
		Y: y,
	}
	return p.Call(MethodNativeWarpCursor, req, nil)
}

// NativeIsAppHidden check if an application is hidden.
func (p *Plugin) NativeIsAppHidden(bundleID string) (bool, error) {
	req := &NativeIsAppHiddenRequest{
		BundleID: bundleID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeIsAppHidden, req, &result)
	return result.Result, err
}

// NativeUnhideApp unhide a hidden application.
func (p *Plugin) NativeUnhideApp(bundleID string) error {
	req := &NativeUnhideAppRequest{
		BundleID: bundleID,
	}
	return p.Call(MethodNativeUnhideApp, req, nil)
}

// NativeBorders draw window border overlays (forwarded to Swift shell).
func (p *Plugin) NativeBorders(frames json.RawMessage) error {
	req := &NativeBordersRequest{
		Frames: frames,
	}
	return p.Call(MethodNativeBorders, req, nil)
}

// NativeRunApplescript execute an AppleScript and return output.
func (p *Plugin) NativeRunApplescript(script string) (*ApplescriptResult, error) {
	req := &NativeRunApplescriptRequest{
		Script: script,
	}
	var result ApplescriptResult
	err := p.Call(MethodNativeRunApplescript, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativePollBurst request burst-mode world model polling (200ms intervals).
func (p *Plugin) NativePollBurst() error {
	return p.Call(MethodNativePollBurst, nil, nil)
}

// NativeAudioDevices list all audio input and output devices.
func (p *Plugin) NativeAudioDevices() ([]AudioDevice, error) {
	var result struct {
		Devices []AudioDevice `json:"devices"`
	}
	err := p.Call(MethodNativeAudioDevices, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Devices, nil
}

// NativeSetAudioDevice set the default audio input or output device.
func (p *Plugin) NativeSetAudioDevice(uid string, deviceType string) error {
	req := &NativeSetAudioDeviceRequest{
		UID: uid,
		DeviceType: deviceType,
	}
	return p.Call(MethodNativeSetAudioDevice, req, nil)
}

// NativeKeyboardLayout get the current keyboard layout and key mappings.
func (p *Plugin) NativeKeyboardLayout() (*NativeKeyboardLayoutResponse, error) {
	var result NativeKeyboardLayoutResponse
	err := p.Call(MethodNativeKeyboardLayout, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeRunningApps list all running applications.
func (p *Plugin) NativeRunningApps() ([]RunningApp, error) {
	var result struct {
		Apps []RunningApp `json:"apps"`
	}
	err := p.Call(MethodNativeRunningApps, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Apps, nil
}

// NativeFrontmostApp get the currently active (frontmost) application.
func (p *Plugin) NativeFrontmostApp() (*RunningApp, error) {
	var result struct {
		App *RunningApp `json:"app,omitempty"`
	}
	err := p.Call(MethodNativeFrontmostApp, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.App, nil
}

// NativeQuitApp gracefully quit an app by bundle ID.
func (p *Plugin) NativeQuitApp(bundleID string) (bool, error) {
	req := &NativeQuitAppRequest{
		BundleID: bundleID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeQuitApp, req, &result)
	return result.Result, err
}

// NativeForceQuitApp force-quit an app by bundle ID.
func (p *Plugin) NativeForceQuitApp(bundleID string) (bool, error) {
	req := &NativeForceQuitAppRequest{
		BundleID: bundleID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeForceQuitApp, req, &result)
	return result.Result, err
}

// NativeHideApp hide an app by bundle ID.
func (p *Plugin) NativeHideApp(bundleID string) error {
	req := &NativeHideAppRequest{
		BundleID: bundleID,
	}
	return p.Call(MethodNativeHideApp, req, nil)
}

// NativeActivateApp bring an app to front by bundle ID.
func (p *Plugin) NativeActivateApp(bundleID string, allWindows *bool) error {
	req := &NativeActivateAppRequest{
		BundleID: bundleID,
		AllWindows: allWindows,
	}
	return p.Call(MethodNativeActivateApp, req, nil)
}

// NativeMinimizeWindow minimize a window by ID.
func (p *Plugin) NativeMinimizeWindow(windowID string) error {
	req := &NativeMinimizeWindowRequest{
		WindowID: windowID,
	}
	return p.Call(MethodNativeMinimizeWindow, req, nil)
}

// NativeUnminimizeWindow restore a minimized window by ID.
func (p *Plugin) NativeUnminimizeWindow(windowID string) error {
	req := &NativeUnminimizeWindowRequest{
		WindowID: windowID,
	}
	return p.Call(MethodNativeUnminimizeWindow, req, nil)
}

// NativeCloseWindow close a window by ID.
func (p *Plugin) NativeCloseWindow(windowID string) error {
	req := &NativeCloseWindowRequest{
		WindowID: windowID,
	}
	return p.Call(MethodNativeCloseWindow, req, nil)
}

// NativeGetWindowInfo get detailed info for a single window.
func (p *Plugin) NativeGetWindowInfo(windowID string) (*WindowDetail, error) {
	req := &NativeGetWindowInfoRequest{
		WindowID: windowID,
	}
	var result WindowDetail
	err := p.Call(MethodNativeGetWindowInfo, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeVolume get system volume and mute state.
func (p *Plugin) NativeVolume() (*NativeVolumeResponse, error) {
	var result NativeVolumeResponse
	err := p.Call(MethodNativeVolume, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSetVolume set system volume (0.0–1.0).
func (p *Plugin) NativeSetVolume(volume float64) error {
	req := &NativeSetVolumeRequest{
		Volume: volume,
	}
	return p.Call(MethodNativeSetVolume, req, nil)
}

// NativeMute set mute state on default output device.
func (p *Plugin) NativeMute(muted bool) error {
	req := &NativeMuteRequest{
		Muted: muted,
	}
	return p.Call(MethodNativeMute, req, nil)
}

// NativeDarkMode check if dark mode is active.
func (p *Plugin) NativeDarkMode() (*NativeDarkModeResponse, error) {
	var result NativeDarkModeResponse
	err := p.Call(MethodNativeDarkMode, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSetDarkMode set dark or light mode.
func (p *Plugin) NativeSetDarkMode(dark bool) error {
	req := &NativeSetDarkModeRequest{
		Dark: dark,
	}
	return p.Call(MethodNativeSetDarkMode, req, nil)
}

// InputDrag atomic drag: mouse down, move, mouse up.
func (p *Plugin) InputDrag(fromX int, fromY int, toX int, toY int, durationMs *int) error {
	req := &InputDragRequest{
		FromX: fromX,
		FromY: fromY,
		ToX: toX,
		ToY: toY,
		DurationMs: durationMs,
	}
	return p.Call(MethodInputDrag, req, nil)
}

// InputClipboardRead read clipboard contents by type.
func (p *Plugin) InputClipboardRead(contentType string) (*ClipboardContents, error) {
	req := &InputClipboardReadRequest{
		ContentType: contentType,
	}
	var result ClipboardContents
	err := p.Call(MethodInputClipboardRead, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InputClipboardWrite write typed content to clipboard.
func (p *Plugin) InputClipboardWrite(contentType string, data string) error {
	req := &InputClipboardWriteRequest{
		ContentType: contentType,
		Data: data,
	}
	return p.Call(MethodInputClipboardWrite, req, nil)
}

// NativeBattery get battery status information.
func (p *Plugin) NativeBattery() (*NativeBatteryResponse, error) {
	var result NativeBatteryResponse
	err := p.Call(MethodNativeBattery, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeWifi get WiFi interface information.
func (p *Plugin) NativeWifi() (*NativeWifiResponse, error) {
	var result NativeWifiResponse
	err := p.Call(MethodNativeWifi, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativePlaySound play a named system sound.
func (p *Plugin) NativePlaySound(name string) error {
	req := &NativePlaySoundRequest{
		Name: name,
	}
	return p.Call(MethodNativePlaySound, req, nil)
}

// NativeSpeak speak text using the system text-to-speech engine.
func (p *Plugin) NativeSpeak(text string, voice *string, rate *float64) error {
	req := &NativeSpeakRequest{
		Text: text,
		Voice: voice,
		Rate: rate,
	}
	return p.Call(MethodNativeSpeak, req, nil)
}

// NativeDisplays get metadata for all connected displays.
func (p *Plugin) NativeDisplays() ([]json.RawMessage, error) {
	var result struct {
		Displays []json.RawMessage `json:"displays"`
	}
	err := p.Call(MethodNativeDisplays, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Displays, nil
}

// NativeBrightness get display brightness (0.0-1.0).
func (p *Plugin) NativeBrightness(displayID *int) (*NativeBrightnessResponse, error) {
	req := &NativeBrightnessRequest{
		DisplayID: displayID,
	}
	var result NativeBrightnessResponse
	err := p.Call(MethodNativeBrightness, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSetBrightness set display brightness (0.0-1.0).
func (p *Plugin) NativeSetBrightness(brightness float64, displayID *int) error {
	req := &NativeSetBrightnessRequest{
		Brightness: brightness,
		DisplayID: displayID,
	}
	return p.Call(MethodNativeSetBrightness, req, nil)
}

// NativeScreenshot capture a screenshot as base64-encoded PNG.
func (p *Plugin) NativeScreenshot(windowID *string, displayID *int, region json.RawMessage) (*NativeScreenshotResponse, error) {
	req := &NativeScreenshotRequest{
		WindowID: windowID,
		DisplayID: displayID,
		Region: region,
	}
	var result NativeScreenshotResponse
	err := p.Call(MethodNativeScreenshot, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeMenuBar read the menu bar structure of an application by PID.
func (p *Plugin) NativeMenuBar(pid int) ([]json.RawMessage, error) {
	req := &NativeMenuBarRequest{
		Pid: pid,
	}
	var result struct {
		Items []json.RawMessage `json:"items"`
	}
	err := p.Call(MethodNativeMenuBar, req, &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// NativeClickMenuItem click a menu item by navigating the menu bar path.
func (p *Plugin) NativeClickMenuItem(pid int, path []string) (bool, error) {
	req := &NativeClickMenuItemRequest{
		Pid: pid,
		Path: path,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeClickMenuItem, req, &result)
	return result.Result, err
}

// NativeListSpaces list all spaces across all displays.
func (p *Plugin) NativeListSpaces() ([]json.RawMessage, error) {
	var result struct {
		Spaces []json.RawMessage `json:"spaces"`
	}
	err := p.Call(MethodNativeListSpaces, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Spaces, nil
}

// NativeActiveSpace get the active space per display.
func (p *Plugin) NativeActiveSpace() ([]json.RawMessage, error) {
	var result struct {
		Active []json.RawMessage `json:"active"`
	}
	err := p.Call(MethodNativeActiveSpace, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Active, nil
}

// NativeMoveWindowToSpace move a window to a specific space.
func (p *Plugin) NativeMoveWindowToSpace(windowID string, spaceID int) (bool, error) {
	req := &NativeMoveWindowToSpaceRequest{
		WindowID: windowID,
		SpaceID: spaceID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeMoveWindowToSpace, req, &result)
	return result.Result, err
}

// NativeMoveWindowToDisplay move a window to a different display.
func (p *Plugin) NativeMoveWindowToDisplay(windowID string, displayID int) error {
	req := &NativeMoveWindowToDisplayRequest{
		WindowID: windowID,
		DisplayID: displayID,
	}
	return p.Call(MethodNativeMoveWindowToDisplay, req, nil)
}

// NativeCaptureWindow capture a single window as PNG (base64).
func (p *Plugin) NativeCaptureWindow(windowID string) (*NativeCaptureWindowResponse, error) {
	req := &NativeCaptureWindowRequest{
		WindowID: windowID,
	}
	var result NativeCaptureWindowResponse
	err := p.Call(MethodNativeCaptureWindow, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSetWindowAlpha set window transparency.
func (p *Plugin) NativeSetWindowAlpha(windowID string, alpha float64) error {
	req := &NativeSetWindowAlphaRequest{
		WindowID: windowID,
		Alpha: alpha,
	}
	return p.Call(MethodNativeSetWindowAlpha, req, nil)
}

// NativeAppIcon get app icon as PNG (base64).
func (p *Plugin) NativeAppIcon(bundleID string, size *int) (*NativeAppIconResponse, error) {
	req := &NativeAppIconRequest{
		BundleID: bundleID,
		Size: size,
	}
	var result NativeAppIconResponse
	err := p.Call(MethodNativeAppIcon, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InputDoubleClick double-click at position.
func (p *Plugin) InputDoubleClick(x *int, y *int) error {
	req := &InputDoubleClickRequest{
		X: x,
		Y: y,
	}
	return p.Call(MethodInputDoubleClick, req, nil)
}

// InputRightClick right-click at position.
func (p *Plugin) InputRightClick(x *int, y *int) error {
	req := &InputRightClickRequest{
		X: x,
		Y: y,
	}
	return p.Call(MethodInputRightClick, req, nil)
}

// InputSwitchInputSource switch keyboard input source.
func (p *Plugin) InputSwitchInputSource(sourceID string) (bool, error) {
	req := &InputSwitchInputSourceRequest{
		SourceID: sourceID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodInputSwitchInputSource, req, &result)
	return result.Result, err
}

// InputListInputSources list available keyboard input sources.
func (p *Plugin) InputListInputSources() ([]json.RawMessage, error) {
	var result struct {
		Sources []json.RawMessage `json:"sources"`
	}
	err := p.Call(MethodInputListInputSources, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Sources, nil
}

// NativeDnd get Do Not Disturb / Focus state.
func (p *Plugin) NativeDnd() (*NativeDndResponse, error) {
	var result NativeDndResponse
	err := p.Call(MethodNativeDnd, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSetDnd toggle Do Not Disturb.
func (p *Plugin) NativeSetDnd(enabled bool) error {
	req := &NativeSetDndRequest{
		Enabled: enabled,
	}
	return p.Call(MethodNativeSetDnd, req, nil)
}

// NativeBluetoothDevices list paired/connected Bluetooth devices.
func (p *Plugin) NativeBluetoothDevices() ([]json.RawMessage, error) {
	var result struct {
		Devices []json.RawMessage `json:"devices"`
	}
	err := p.Call(MethodNativeBluetoothDevices, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Devices, nil
}

// NativePreventSleep assert or release sleep prevention.
func (p *Plugin) NativePreventSleep(reason *string, assertionID *string) (*NativePreventSleepResponse, error) {
	req := &NativePreventSleepRequest{
		Reason: reason,
		AssertionID: assertionID,
	}
	var result NativePreventSleepResponse
	err := p.Call(MethodNativePreventSleep, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSystemUptime get system uptime.
func (p *Plugin) NativeSystemUptime() (*NativeSystemUptimeResponse, error) {
	var result NativeSystemUptimeResponse
	err := p.Call(MethodNativeSystemUptime, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeScreenLock lock the screen.
func (p *Plugin) NativeScreenLock() error {
	return p.Call(MethodNativeScreenLock, nil, nil)
}

// NativeColorAtPoint sample pixel color at screen coordinate.
func (p *Plugin) NativeColorAtPoint(x int, y int) (*NativeColorAtPointResponse, error) {
	req := &NativeColorAtPointRequest{
		X: x,
		Y: y,
	}
	var result NativeColorAtPointResponse
	err := p.Call(MethodNativeColorAtPoint, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeCursorInfo get current cursor type and position.
func (p *Plugin) NativeCursorInfo() (*NativeCursorInfoResponse, error) {
	var result NativeCursorInfoResponse
	err := p.Call(MethodNativeCursorInfo, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSpotlight search files via Spotlight.
func (p *Plugin) NativeSpotlight(query string, scope []string, limit *int) ([]json.RawMessage, error) {
	req := &NativeSpotlightRequest{
		Query: query,
		Scope: scope,
		Limit: limit,
	}
	var result struct {
		Results []json.RawMessage `json:"results"`
	}
	err := p.Call(MethodNativeSpotlight, req, &result)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// NativeTrash move file to Trash.
func (p *Plugin) NativeTrash(path string) (bool, error) {
	req := &NativeTrashRequest{
		Path: path,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeTrash, req, &result)
	return result.Result, err
}

// NativeFileTags read or write Finder tags on a file.
func (p *Plugin) NativeFileTags(path string, tags []string) (*NativeFileTagsResponse, error) {
	req := &NativeFileTagsRequest{
		Path: path,
		Tags: tags,
	}
	var result NativeFileTagsResponse
	err := p.Call(MethodNativeFileTags, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeRevealInFinder reveal file in Finder.
func (p *Plugin) NativeRevealInFinder(path string) error {
	req := &NativeRevealInFinderRequest{
		Path: path,
	}
	return p.Call(MethodNativeRevealInFinder, req, nil)
}

// NativeQuickLook generate Quick Look thumbnail as PNG (base64).
func (p *Plugin) NativeQuickLook(path string, size *int) (*NativeQuickLookResponse, error) {
	req := &NativeQuickLookRequest{
		Path: path,
		Size: size,
	}
	var result NativeQuickLookResponse
	err := p.Call(MethodNativeQuickLook, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeSelectedFinderItems get Finder selection.
func (p *Plugin) NativeSelectedFinderItems() ([]string, error) {
	var result struct {
		Paths []string `json:"paths"`
	}
	err := p.Call(MethodNativeSelectedFinderItems, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Paths, nil
}

// NativeNotify post a rich notification (osascript fallback).
func (p *Plugin) NativeNotify(title string, body *string, subtitle *string, sound *string) (*NativeNotifyResponse, error) {
	req := &NativeNotifyRequest{
		Title: title,
		Body: body,
		Subtitle: subtitle,
		Sound: sound,
	}
	var result NativeNotifyResponse
	err := p.Call(MethodNativeNotify, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeListNotifications list delivered notifications (partial — returns empty).
func (p *Plugin) NativeListNotifications() ([]json.RawMessage, error) {
	var result struct {
		Notifications []json.RawMessage `json:"notifications"`
	}
	err := p.Call(MethodNativeListNotifications, nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Notifications, nil
}

// NativeDismissNotification dismiss a delivered notification (partial — no-op).
func (p *Plugin) NativeDismissNotification(id string) error {
	req := &NativeDismissNotificationRequest{
		ID: id,
	}
	return p.Call(MethodNativeDismissNotification, req, nil)
}

// NativeCurrentUser get current user session info.
func (p *Plugin) NativeCurrentUser() (*UserSessionInfo, error) {
	var result UserSessionInfo
	err := p.Call(MethodNativeCurrentUser, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeDefaultBrowser get the default browser bundle ID.
func (p *Plugin) NativeDefaultBrowser() (*NativeDefaultBrowserResponse, error) {
	var result NativeDefaultBrowserResponse
	err := p.Call(MethodNativeDefaultBrowser, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeLoginItems list login items (launch-at-login entries).
func (p *Plugin) NativeLoginItems() (*NativeLoginItemsResponse, error) {
	var result NativeLoginItemsResponse
	err := p.Call(MethodNativeLoginItems, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeClipboardChangeCount get the clipboard change count.
func (p *Plugin) NativeClipboardChangeCount() (*NativeClipboardChangeCountResponse, error) {
	var result NativeClipboardChangeCountResponse
	err := p.Call(MethodNativeClipboardChangeCount, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InputClipboardReadAll read all items from the clipboard.
func (p *Plugin) InputClipboardReadAll() (*InputClipboardReadAllResponse, error) {
	var result InputClipboardReadAllResponse
	err := p.Call(MethodInputClipboardReadAll, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// InputClipboardWriteItems write multiple typed items to the clipboard.
func (p *Plugin) InputClipboardWriteItems(items []json.RawMessage) error {
	req := &InputClipboardWriteItemsRequest{
		Items: items,
	}
	return p.Call(MethodInputClipboardWriteItems, req, nil)
}

// NativeAxElementAtPoint get the accessibility element at a screen point.
func (p *Plugin) NativeAxElementAtPoint(pid int, x int, y int) (*AXElementInfo, error) {
	req := &NativeAxElementAtPointRequest{
		Pid: pid,
		X: x,
		Y: y,
	}
	var result AXElementInfo
	err := p.Call(MethodNativeAxElementAtPoint, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeAxElementTree get the accessibility element tree rooted at an element.
func (p *Plugin) NativeAxElementTree(element AXElementRef, depth *int) (*AXElementNode, error) {
	req := &NativeAxElementTreeRequest{
		Element: element,
		Depth: depth,
	}
	var result AXElementNode
	err := p.Call(MethodNativeAxElementTree, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeAxReadAttributes read specific attributes from an accessibility element.
func (p *Plugin) NativeAxReadAttributes(element AXElementRef, attributes []string) error {
	req := &NativeAxReadAttributesRequest{
		Element: element,
		Attributes: attributes,
	}
	return p.Call(MethodNativeAxReadAttributes, req, nil)
}

// NativeAxSetAttribute set an attribute on an accessibility element.
func (p *Plugin) NativeAxSetAttribute(element AXElementRef, attribute string, value json.RawMessage) (bool, error) {
	req := &NativeAxSetAttributeRequest{
		Element: element,
		Attribute: attribute,
		Value: value,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeAxSetAttribute, req, &result)
	return result.Result, err
}

// NativeAxPerformAction perform an action on an accessibility element.
func (p *Plugin) NativeAxPerformAction(element AXElementRef, action string) (bool, error) {
	req := &NativeAxPerformActionRequest{
		Element: element,
		Action: action,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeAxPerformAction, req, &result)
	return result.Result, err
}

// NativeAxObserve start observing AX notifications (STUB — not yet implemented).
func (p *Plugin) NativeAxObserve(pid int, notifications []string) (*NativeAxObserveResponse, error) {
	req := &NativeAxObserveRequest{
		Pid: pid,
		Notifications: notifications,
	}
	var result NativeAxObserveResponse
	err := p.Call(MethodNativeAxObserve, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeAxUnobserve stop observing AX notifications (STUB).
func (p *Plugin) NativeAxUnobserve(subscriptionID string) (bool, error) {
	req := &NativeAxUnobserveRequest{
		SubscriptionID: subscriptionID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeAxUnobserve, req, &result)
	return result.Result, err
}

// NativeObserveWindows start observing window events for a PID (STUB — not yet implemented).
func (p *Plugin) NativeObserveWindows(pid int) (*NativeObserveWindowsResponse, error) {
	req := &NativeObserveWindowsRequest{
		Pid: pid,
	}
	var result NativeObserveWindowsResponse
	err := p.Call(MethodNativeObserveWindows, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// NativeUnobserveWindows stop observing window events (STUB).
func (p *Plugin) NativeUnobserveWindows(subscriptionID string) (bool, error) {
	req := &NativeUnobserveWindowsRequest{
		SubscriptionID: subscriptionID,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeUnobserveWindows, req, &result)
	return result.Result, err
}

// NativeSetWindowLevel set a window's level (floating, normal, below).
func (p *Plugin) NativeSetWindowLevel(windowID string, level string) (bool, error) {
	req := &NativeSetWindowLevelRequest{
		WindowID: windowID,
		Level: level,
	}
	var result struct {
		Result bool `json:"result"`
	}
	err := p.Call(MethodNativeSetWindowLevel, req, &result)
	return result.Result, err
}

// ControlSignal send a control signal to the Swift shell via the control stream.
func (p *Plugin) ControlSignal(signal string) error {
	req := &ControlSignalRequest{
		Signal: signal,
	}
	return p.Call(MethodControlSignal, req, nil)
}

// EventsEmit emit a plugin event on the event bus.
func (p *Plugin) EventsEmit(eventType string, data json.RawMessage, correlationID *string) error {
	req := &EventsEmitRequest{
		EventType: eventType,
		Data: data,
		CorrelationID: correlationID,
	}
	return p.Call(MethodEventsEmit, req, nil)
}

// InputTypeText type text into the active application via clipboard paste.
func (p *Plugin) InputTypeText(text string) error {
	req := &InputTypeTextRequest{
		Text: text,
	}
	return p.Call(MethodInputTypeText, req, nil)
}

// InputPressKey press a key by raw keycode or name, with optional modifiers.
func (p *Plugin) InputPressKey(code *int, name *string, modifiers []string) error {
	req := &InputPressKeyRequest{
		Code: code,
		Name: name,
		Modifiers: modifiers,
	}
	return p.Call(MethodInputPressKey, req, nil)
}

// InputRawKey send a raw key event (press, release, or click) without modifier lifting.
func (p *Plugin) InputRawKey(code int, direction string) error {
	req := &InputRawKeyRequest{
		Code: code,
		Direction: direction,
	}
	return p.Call(MethodInputRawKey, req, nil)
}

// InputClick click a mouse button.
func (p *Plugin) InputClick(button string) error {
	req := &InputClickRequest{
		Button: button,
	}
	return p.Call(MethodInputClick, req, nil)
}

// InputScroll scroll the mouse wheel.
func (p *Plugin) InputScroll(direction string, amount *int) error {
	req := &InputScrollRequest{
		Direction: direction,
		Amount: amount,
	}
	return p.Call(MethodInputScroll, req, nil)
}

// InputMouseButton press or release a mouse button (for drag operations, etc.).
func (p *Plugin) InputMouseButton(button string, direction string) error {
	req := &InputMouseButtonRequest{
		Button: button,
		Direction: direction,
	}
	return p.Call(MethodInputMouseButton, req, nil)
}

// InputClipboardAction perform a clipboard action (copy, paste, or set text).
func (p *Plugin) InputClipboardAction(action string, text *string) error {
	req := &InputClipboardActionRequest{
		Action: action,
		Text: text,
	}
	return p.Call(MethodInputClipboardAction, req, nil)
}

// NativeLaunchApp launch an application by bundle ID.
func (p *Plugin) NativeLaunchApp(bundleID string, newInstance *bool) error {
	req := &NativeLaunchAppRequest{
		BundleID: bundleID,
		NewInstance: newInstance,
	}
	return p.Call(MethodNativeLaunchApp, req, nil)
}

// NativeOpenTarget open a URL or file path with the default handler.
func (p *Plugin) NativeOpenTarget(target string) error {
	req := &NativeOpenTargetRequest{
		Target: target,
	}
	return p.Call(MethodNativeOpenTarget, req, nil)
}
