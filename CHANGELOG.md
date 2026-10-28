# Changelog

Versions before this file predate it; their contents are in the repo's
git history.

## Unreleased

## 0.27.0 — 2026-10-28

- New log-event constants for changes a person (or a plugin acting for
  them) makes to their own commands and lists: `command.user_created`,
  `command.user_modified`, `command.user_deleted`,
  `state.collection_created` and `state.collection_deleted`. Each names who
  made the change; the content is redacted unless log recording is on.
- The harness tests find the helloworld plugin under its `.exe` name on
  Windows.

## 0.26.0 — 2026-10-25

- New log-event constants: `plugin.restarted` (cause, backoff and restart
  count), `plugin.restart_budget_exhausted`, and `stage.respawned` (a
  pipeline stage started afresh instead of reused, with the reason).
  `state.ttl_swept` is now emitted at Info, once per collection a TTL sweep
  removed records from.
- A window's frame source can now be `assumed`: the platform cannot see the
  window (the focused window on GNOME Wayland) and gives its display's
  visible bounds, so its geometry is not a fact.
- Generated docs: `commands.resolve`'s `collections` narrows completions to
  commands stored in the named command-set collections; Hyprland is named
  among the desktop sessions an operation cannot run on, and the platform
  profile's `session` field lists `hyprland`.

## 0.25.0 — 2026-10-21

Pages, part two: the old names are gone.

- Breaking: the deprecated `SettingsTab` and `SettingsTabFunc` are removed;
  register a page with `Plugin.Page(key, fn)` / `PageFunc`.
- The manifest declares a plugin's pages as `implements.pages` (was
  `implements.settings_tabs`, now refused with the rename named).

## 0.24.0 — 2026-10-21

Pages: a plugin's own screens are registered as pages.

- `Plugin.Page(key, fn)` and `PageFunc` register a renderer for one of the
  plugin's own pages. `SettingsTab` and `SettingsTabFunc` are deprecated
  aliases and will be removed in the next release, when the manifest's
  `implements.settings_tabs` becomes `implements.pages`.

## 0.23.0 — 2026-10-20

Re-decodes can target the dictation stages, and a run can record its audio.

- `RecognitionRedecodeRequest.Language`: the language a dictation stage's
  probe decodes in, as the live stage's `--language`. A command stage
  ignores it. The dictation stages (WhisperKit, whisper.cpp) can now be
  re-decoded through `RecognitionRedecode` too.
- `PipelinesRunRequest.RecordTo`: record a run's input audio into a folder
  of the caller's own data dir, one WAV per span plus a `manifest.jsonl`
  line each.

## 0.22.0 — 2026-10-20

Re-decodes can match the live recognizer, and report time, confidence and
grammar.

- `RedecodeItem.Structured`: decode the item through the live structured
  grammar (the pattern DAG and union a recognition pipeline is seeded with)
  instead of the flat word list. The grammar is the platform's; set the
  flag to opt in.
- `RedecodeLine.DecodeMs` and `RedecodeLine.WordMargins`: the recognizer's
  time on the item and each decoded word's confidence margin. Both are
  absent on a failed line.
- `RecognitionRedecodeResponse.GrammarVersion` and
  `PipelinesGrammarResponse.GrammarVersion`: a content version of the
  structured grammar. They are equal for the same grammar, so a plugin
  caching re-decode results can key them by the grammar they were decoded
  under.

## 0.21.0 — 2026-10-19

Breaking: plugins no longer send raw messages to the host.

- Removed `Methods.ControlSignal` and `ControlSignalRequest`. Use the
  operation for the effect: `HUDShow` / `HUDHide` for windows,
  `IndicatorSet` for the recording indicator, `ArtifactDownload` for the
  plugin's own artifacts.
- Removed `OnActionResponse.ControlMessage`. A handler that opens or closes
  a window calls `HUDShow` / `HUDHide` itself; return data in `Result`.
  The platform ignores (and logs) a `control_message` from an older SDK.
- Removed `ControlMessage` from `DispatchResult`, `SelectionPickResult`
  and `SessionEndCleanupResult`: each operation now delivers its own HUD
  change. Also removed from the harness's `ActionResponse`.

## 0.20.0 — 2026-10-19

New surface; nothing removed.

- `Methods.IndicatorSet(IndicatorSetRequest{State: ...})` shows what input is
  doing on the recording indicator (the menu-bar or tray icon, and the
  floating pill). `IndicatorState` names what the input becomes, not any one
  plugin's modes: `IndicatorStateIdle`, `IndicatorStateInputText`,
  `IndicatorStateInputAction`, `IndicatorStateListening`,
  `IndicatorStateStandby`, `IndicatorStateProcessing`,
  `IndicatorStateSpeaking`, `IndicatorStateLoading`. It needs the
  `microphone` privilege, and a host that draws no indicator refuses it.
- `Methods.ArtifactDownload(ArtifactDownloadRequest{Ref: "<plugin>/<artifact>"})`
  downloads one of the caller's own declared artifacts on the platform's
  downloader; progress is published to `_platform.model_download`. A ref
  naming another plugin's artifact, or a missing downloader, is an error.
- Prefer these to the raw `ControlSignal("shell:icon:...")` and
  `ControlSignal("download_model:...")` forms: the platform now refuses a
  raw icon state, and a later release removes `ControlSignal`.

## 0.19.1 — 2026-10-18

Documentation only; no behaviour or type changes.

- `ResolveResult.SetsTags` and `ClearsTags` are documented: they are the tags the
  resolve's commit applied, so a write the platform's policy refused, or
  one a hold ended before, is absent; a preview reports what committing
  would set and clear.
- `CancelsBridge` is documented as the platform now reads it: a command of two or
  more words that opens with a literal word interrupts a held part-way
  match without it; it matters for one-word cancel words like "over".

## 0.19.0 — 2026-10-18

### Transcript payload

- `PipelineTranscriptEventParams` gains `focused_app`, `focused_app_now` and
  `partial`. The platform always sent them; the declared type lacked them, so
  a plugin on the typed helper could not read them. The platform now builds
  the payload from the declared type, so the two cannot drift again.

### Handshake declared

- The handshake every SDK answers itself is now declared in the platform's
  protocol and so generated: `HookListMethods`, `HookListActionTypes`,
  `HookOnReady` constants, and the `ListMethodsResponse`,
  `ListActionTypesResponse` and `OnReadyRequest` types. The SDK registers
  its own handshake through those constants. `OnReadyRequest.platform` is
  the same `PlatformProfileResponse` type `platform.profile` returns.

## 0.18.0 — 2026-10-18

### Typed event subscription

- One generated helper per platform event, beside the call wrappers:
  `p.OnAppFocused(func(evt AppFocusedEventParams) { … })`. It ties the
  event's name to its generated payload type, so a renamed event or field
  is a build error, where a `p.On("_platform.app.focused", …)` listener
  compiled and heard nothing. An event with no payload, or a signal (its
  payload is advisory: re-read the state it announces), takes `func()`.
  `OnEffectDisplaced` stays the hand-written helper for its event.
- A payload that does not decode skips the listener and logs one line with
  the event's name and the payload's size, never the payload.
- `On` and `OnPattern` are unchanged, for other plugins' events and
  patterns.

## 0.17.0 — 2026-10-17

0.16.0 was tagged 2026-10-10 without a section of its own; the changes
it carried are among these.

### Typed wrappers

- The generated wrappers are methods of a generated `Methods` type over a
  `Caller` (any type with `Call(method, params, result) error`), and
  `Plugin` embeds it, so `p.CollectionGet(req)` and every other call site
  is unchanged. `NewMethods(c)` gives the same wrappers over any Caller,
  such as a test's fake, with no transport.
- `p.WithTimeout(d)` returns the wrappers with `d` as each call's timeout,
  in place of `Call`'s 10 s default. Before, only a raw `CallWithTimeout`
  could wait longer.
- A `Plugin` built by hand (`&Plugin{...}`, inside the package) must set
  `p.Methods = NewMethods(p)`; `NewPlugin` and `NewDetachedPlugin` do.

### Network

- Over a handed-off proxy channel (`BRANCHKIT_PROXY=fd://N`, Linux), a dial
  whose deadline passed while waiting for the channel's answer no longer
  leaves the next dial one answer behind. The late answer is read and
  discarded before the next ask, so every dial gets the connection that
  answers its own ask; before, it got the previous dial's, and so did every
  dial after it.

### World

- `DisplayInfo.IsPrimary`: whether a display is the system's main one (the
  menu bar's on macOS, the primary monitor on Windows, RandR's primary
  output on X11). False for every display where the system cannot say.

### Native

- `NativeCaretBounds`: where the caret is on screen in the focused text
  field (`text.caret`). Nothing for a password field or a field that
  cannot say; on Linux, X11 only.

### HUD

- `RelativeToCaret`: a window above or below the line being typed, lined
  up with the text caret where it was when the window opened. Needs the
  `text.caret` privilege; otherwise, or where the caret cannot be read, on
  the display.
- Anchors cover the whole three-by-three grid: `AnchorCenterLeft` and
  `AnchorCenterRight` join the seven there were.
- `HUDCreateChannelRequest.RelativeTo` places a window against the display
  (`RelativeToDisplay`, the default), inside the focused window
  (`RelativeToFocusedWindow`) or beside the pointer (`RelativeToPointer`);
  `HUDCreateChannelRequest.Offset` nudges it from its anchored place, in
  pixels. The manifest's `hud_windows` take the same `relative_to` and
  `offset`. Needs a platform that knows them.

### Listen

- A connection served through the platform's listener relay (Windows)
  now reports the client's address as its `RemoteAddr`, so
  `http.Request.RemoteAddr` is the client's, as it is on an inherited
  listener. The relay header is now `BKRELAY/2`, whose answer carries the
  peer (`OK <addr>`); a bare `OK` from an older platform is still accepted.
  Needs a platform that speaks version 2.

### Output

- `OutputStateRequest.DrawsOwnWindow`: publish a state for every other
  renderer (speech, the window's accessibility element) while the plugin
  keeps drawing its own window with `hud.push`; the platform draws nothing.

### Request stages (stage runtime)

- Added `pipeline.ServeRequests` / `ServeRequestsOn` with the `RequestHandler`
  interface: the fourth stage shape, one answer per request (a language model,
  a translator, a classifier). The capability declares `stage_type: "request"`;
  each `request {request_id, body}` gets exactly one `reply` with the same id,
  in arrival order, carrying the handler's answer as `body` or its error as
  `error`. A request with no id gets a non-fatal `bad_request` error event.

## 0.15.0 — 2026-10-05

### Speech engines (stage runtime)

- Added `pipeline.ServeSpeechEngine` / `ServeSpeechEngineOn` with the
  `SpeechEngine` interface and `SpeakCtx`: the third stage shape, a
  text-to-speech engine. Each `speak` request becomes an audio session the
  stage streams back (`Start`, then `Audio` per piece); the runtime keeps
  requests in order, cancels the one in progress the moment the platform
  sends its `audio_stop` (`Cancelled`, `Done`, `Context`), closes a queued one
  that is cancelled before it starts, and ends every utterance with exactly
  one `audio_stop`, after an `error` when `Speak` fails.
- Added `pipeline.SharedClockMs`, the clock the platform stamps microphone
  audio with (`CLOCK_UPTIME_RAW` on macOS, `CLOCK_MONOTONIC` on other unixes,
  wall time on Windows). An audio sink stamps `playback_started` /
  `playback_ended` with it. `golang.org/x/sys` is now a direct dependency.
- Generated: `audio.Speak`, `audio.PlaybackStarted`, `audio.PlaybackEnded`
  and their event tags; `Capability.Voices` with `VoiceInfo`.

### Device triggers

- Added `BindingsSetTriggers` (`TriggerDecl`, `TriggerKind`): a device plugin publishes
  the triggers it offers, each with a label, an optional heading (a layer)
  and whether it is a button or momentary. Settings lists them, bound or not,
  and binds any of them from a command picker as the user's own binding.
  Publish again when a device connects or goes away; the list is dropped when
  the plugin stops.
- Added `BindingsReport` and `BindingsSet` (`BindingsReportRequest`, `BindingsSetRequest`, `BindingEdit`, `BindingEdge`):
  a plugin that owns an input device makes its controls binding triggers,
  bound in the platform's table like hotkeys. It reports each press of one of
  its own triggers and the platform runs what it is bound to; it may set what
  its own triggers are bound to, and those edits run on its authority. The
  caller is always the source: neither call can name another plugin's
  triggers. Trigger names are the plugin's own (`"g2/button3"`), with the
  hotkey event words after them, plus the new `repeat`.

### Key bindings (breaking)

- Removed `KeybindsRegister` and its `KeybindsRegisterRequest` / `KeybindsRegisterResult` /
  `RegistrySnapshot` / `RegistryEntry` types. The `keybinds.register` operation is gone: it let any plugin
  replace every hotkey on the machine. The platform now builds the hotkey
  table itself from the `_platform.bindings` collection and the user's edits
  in `_platform.binding_overrides`, and publishes the result as
  `_platform.bindings.active`. A plugin contributes a hotkey under
  `collection_data["_platform.bindings"]` (previously
  `collection_data.keybinds`, same shape).

### Pipeline

- `pipeline.Reader.ReadEvent` no longer reports `io.EOF` for a header that
  promised a payload followed by end of stream with no payload bytes: that is
  a truncated frame, and it now fails with `io.ErrUnexpectedEOF` (wrapped).
  Callers that treat `io.EOF` as an orderly close would otherwise finish on
  half the input.
- `monitors.DisplayInfo` gains `IsAsleep *bool` (`is_asleep`): the display is
  connected but in display sleep. Absent means awake.

### D-Bus calls (Linux)

- `NativeDbusCall(NativeDbusCallRequest)` calls one D-Bus method the plugin
  declared in `requires.dbus.methods` and the user switched on. Linux only;
  elsewhere the call fails with "D-Bus exists only on Linux".

## 0.13.0 — 2026-09-28

### Breaking: generated methods take a request struct

- Every generated method that takes parameters now takes its request struct
  instead of positional arguments:
  `p.CollectionFetch(branchkit.CollectionFetchRequest{ID: id, Name: name})`,
  not `p.CollectionFetch(id, name)`. Positional order came from the schema,
  which is alphabetical, so it said nothing a caller could guess, and 62
  methods had two or more parameters of the same type that compiled just as
  well swapped. Named fields make every argument say what it is, and a new
  optional field can be added later without breaking any caller. Methods
  with no parameters are unchanged.
- `branchkit.Ptr(v)` returns `&v`, for the optional (pointer) fields:
  `AcceptsInput: branchkit.Ptr(true)`.
- Migrating: each call's arguments become the request's fields, named as
  in the old signature, and a `nil` argument is simply left out.

### `**` in pattern listeners

- `OnPattern` now takes `**`, zero or more whole segments, matching the
  platform's subscription grammar: `ext.acme.**` hears every depth under the
  vendor (the bare `ext.acme` included), and `_platform.**` every platform
  event. `*` is unchanged — exactly one segment. Wildcards are whole segments
  only, so `a**` stays literal text. Before this, a `**` pattern matched
  nothing but its own literal text. The matcher runs the platform's topic
  conformance table (a copy ships beside the tests), so it answers exactly
  what delivery does.

## 0.12.0 — 2026-09-25

### Who sent this event

- `Plugin.CurrentEventOrigin()` returns an `EventOrigin{Source, OnBehalfOf}`
  for the event notification an `On` or `OnPattern` listener is handling.
  The platform delivers every event a subscription matches, whoever emitted
  it, and until now a listener could not tell two senders of one event type
  apart. `Source` is the emitter the platform authenticated (a plugin id,
  `_platform`, or a stage's name) and can be trusted; `OnBehalfOf` is that
  emitter's own actor label, a claim by `Source`. Same ambient shape as
  `CurrentCorrelation()`; empty in a request handler and in a goroutine the
  listener spawns, so read it first and pass the value. A platform that
  does not send the sender leaves it empty.

## 0.11.0 — 2026-09-24

### Spend a stored credential without holding it

- `HttpRequest` (`http.request`): hand the platform a whole HTTPS request.
  A header may be an `HttpHeader{Name, Secret, Prefix}` naming one of your
  plugin's stored secrets; the platform substitutes it only if that secret
  is bound to the request's host, performs the request in a sandboxed
  helper through your plugin's proxy, returns redirects unfollowed, and
  redacts any echo of the secret. Your plugin never receives the value.
- `NetworkRequestHost` (`network.request_host`): for a manifest declaring
  `requires.network: {hosts, requestable: true}`, ask for one more exact
  host at runtime. It appears on your plugin's page, off until the user
  allows it.
- `SecretsRequestSlot` (`secrets.request_slot`): ask for a credential row on
  your plugin's page, bound to a host you can reach. The user pastes the
  value there; it goes straight into the store.
- Secret-field manifest entries may declare `for_host`.

### Also new

- `BlobState` (`blob.state`): a blob's current generation, length and
  version, for a restarted provider or a starting consumer.
- `ErrUnsupported` and `UnsupportedReasonOf`: a platform refusal is its own
  error kind (`unsupported`, -32007) with a closed reason —
  `platform_no_analogue`, `platform_unported`, `session_unsupported` — no
  longer `not_permitted` with the reason in the detail text.
- `native.format_date` and richer, per-platform date/time format answers.

### Behaviour changes

- An explicitly EMPTY list in a request is now sent as `[]`, not dropped.
  Nullable collection fields on request structs lost `omitempty`: `nil`
  still means "absent", `[]string{}` now means "empty". This matters for
  `CommandsResolve`'s `ActiveTags` (absent = the live session's tags,
  empty = no tags) and `NativeFileTags`'s `Tags` (empty = clear the tags,
  which previously read the tags instead).
- The listener relay dials a Windows named-pipe rendezvous (`npipe://`);
  it had kept dialling TCP, so a Go plugin's listener was unreachable on
  Windows.

0.9.0 and 0.10.0 shipped without entries here; their changes (wrapper
parameters reordered to required-first, 61 wrappers gaining return values,
computed `ok` results becoming `(bool, error)`) are in the repo history
around those tags.

## 0.8.0 — 2026-09-19

### The platform's `Action` is a type

`dispatch` — the call a plugin makes most — took raw JSON. It takes the
generated `Action` now, as do `commands.resolve`'s winner and tied
candidates, and the actions a delegated pipeline returns from
`on_transcript`. `Action` is an internally tagged union with a
self-recursive `sequence` variant, and it is rendered in full: no field
that carries an action is raw JSON any more.

Two things stay deliberately open, and say so in their own doc comments:
an action's `params` (per-plugin — `branchkit-gen` types those from the
receiving plugin's `action_types`) and `commands.push`'s `action` (the
authored dialect is a superset of the wire shape, so one type would lie
about it).

### Closed shapes stopped hiding as JSON

Every remaining untyped member of the generated surface was audited
against the code that produces it, and either declared or defended in
writing. Newly typed here:

- `hud.push` takes `HudFragment[]`
- `hud.create_channel` takes the `Anchor` enum (an unrecognised value is
  now an error instead of a silent fall back to the default)
- `selection.set` takes `HUDItem[]`
- `keybinds.register` takes `RegistrySnapshot`
- `commands.enumerate` returns a typed `binding`
- `commands.push` takes `CommandSpec[]`
- `on_commands_changed` carries typed command snapshots — and
  `disabled_plugins`, which the schema had been omitting entirely

Untyped members across the three SDKs: 193 → 147. What is left is listed
field by field, each with a written reason, in the generator's fidelity
ledger (every remaining member is opaque by declaration in the Rust source).

### Known gap

A command `pattern` token is `string | string[]`, an untagged union. The
generator refuses to emit one rather than degrade it to raw JSON, so
`CommandSpec.pattern` and `.variants` stay open and the hand-written
command builder remains the way to construct a pattern.

### Also

`Dispatch` takes an `Action`; `PushCommandSpecs` goes straight
through the generated wrapper; the HUD sugar builds typed fragments.
