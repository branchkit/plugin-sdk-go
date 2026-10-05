# Changelog

Versions before this file predate it; their contents are in the repo's
git history.

## Unreleased

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
  listener spawns, so read it first and pass the value. An actuator that
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
