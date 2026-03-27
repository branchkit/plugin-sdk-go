// AUTO-GENERATED from contracts/*.json — do not edit.
// Run: python3 contracts/generate.py

package shared

// RPC method names: plugin → actuator (use with plugin.Call()).
const (
	MethodCommandsDiscover = "commands.discover"
	MethodCommandsHasPartial = "commands.has_partial"
	MethodCommandsMatch = "commands.match"
	MethodControlSignal = "control.signal"
	MethodDiscoveryClose = "discovery.close"
	MethodDiscoveryOpen = "discovery.open"
	MethodEventsEmit = "events.emit"
	MethodGrammarPush = "grammar.push"
	MethodKeyNamesSet = "key_names.set"
	MethodKeybindsRegister = "keybinds.register"
	MethodListsDelete = "lists.delete"
	MethodListsGet = "lists.get"
	MethodListsUpdate = "lists.update"
	MethodNativeAudioDevices = "native.audio_devices"
	MethodNativeBatchIsTileable = "native.batch_is_tileable"
	MethodNativeBatchSetFrames = "native.batch_set_frames"
	MethodNativeBorders = "native.borders"
	MethodNativeCursor = "native.cursor"
	MethodNativeIsAppHidden = "native.is_app_hidden"
	MethodNativeKeyboardLayout = "native.keyboard_layout"
	MethodNativePollBurst = "native.poll_burst"
	MethodNativeRaiseWindow = "native.raise_window"
	MethodNativeRunApplescript = "native.run_applescript"
	MethodNativeSetAudioDevice = "native.set_audio_device"
	MethodNativeToggleFullscreen = "native.toggle_fullscreen"
	MethodNativeUnhideApp = "native.unhide_app"
	MethodNativeWarpCursor = "native.warp_cursor"
	MethodNativeWorldModel = "native.world_model"
	MethodSelectionSet = "selection.set"
	MethodStoreGet = "store.get"
	MethodStorePush = "store.push"
	MethodTagsGet = "tags.get"
	MethodTagsModify = "tags.modify"
)

// RPC method names: actuator → plugin (use with plugin.Handle()).
const (
	HookBuildCommandRegistry = "build_command_registry"
	HookCalibrate = "calibrate"
	HookOnAction = "on_action"
	HookRenderHud = "render_hud"
	HookRenderSettings = "render_settings"
	HookSpeechOrchestrate = "speech_orchestrate"
	HookSpeechPipeline = "speech_pipeline"
)

// Platform event type constants (use with plugin.On()).
const (
	EventActionExecuted = "_platform.action.executed"
	EventAppFocused = "_platform.app.focused"
	EventCommandMatched = "_platform.command.matched"
	EventDisplayChanged = "_platform.display.changed"
	EventKeyboardLayoutChanged = "_platform.keyboard.layout_changed"
	EventModeChanged = "_platform.mode.changed"
	EventPluginDisabled = "_platform.plugin.disabled"
	EventPluginEnabled = "_platform.plugin.enabled"
	EventSelectionPicked = "_platform.selection.picked"
	EventSpeechRecognized = "_platform.speech.recognized"
	EventSpeechSessionEnded = "_platform.speech.session_ended"
	EventStoreUpdated = "_platform.store.updated"
	EventTagsChanged = "_platform.tags.changed"
	EventWindowClosed = "_platform.window.closed"
	EventWindowCreated = "_platform.window.created"
	EventWindowFocused = "_platform.window.focused"
	EventWindowFrameChanged = "_platform.window.frame_changed"
	EventWindowTitleChanged = "_platform.window.title_changed"
	EventWorldUpdated = "_platform.world.updated"
)

// Tag namespace prefix constants.
const (
	TagPrefixApp = "app."
	TagPrefixPlugin = "plugin."
)
