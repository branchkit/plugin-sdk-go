# Changelog

Versions before this file predate it; their contents are in the repo's
git history.

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
field by field, with a reason, in the platform's
`DESIGN_SDK_GENERATION_FIDELITY.md` ledger.

### Known gap

A command `pattern` token is `string | string[]`, an untagged union. The
generator refuses to emit one rather than degrade it to raw JSON, so
`CommandSpec.pattern` and `.variants` stay open and the hand-written
command builder remains the way to construct a pattern.

### Also

`Dispatch` takes an `Action`; `PushCommandSpecs` goes straight
through the generated wrapper; the HUD sugar builds typed fragments.
