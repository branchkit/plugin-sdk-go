# BranchKit Plugin SDK (Go)

BranchKit is an accessibility plugin platform for the desktop. The platform
loads plugins, confines each one to what it declared, tracks what is true right
now (which app has focus, which mode is active), and routes voice commands,
hotkeys and other input to whichever plugin claims them. This SDK is how a Go
program becomes one of those plugins. MIT licensed.

**Status:** BranchKit is pre-launch; the application is in private
development, and this SDK is published and usable today. Versions are 0.x, so a
minor release can break callers — [CHANGELOG.md](CHANGELOG.md) says what
changed and how to migrate.

## Install

Go 1.24 or later.

```sh
go get github.com/branchkit/plugin-sdk-go@latest
```

The fastest start is the scaffold, which writes a working plugin, generates its
typed action handlers and builds it:

```sh
branchkit-cli dev init --name my-plugin --template go
```

## A minimal plugin

A plugin is a directory with a manifest, the commands it contributes, and a
program. This is what `dev init` writes, trimmed.

`plugin.json` declares who the plugin is, what it may do, and what it offers:

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "version": "0.1.0",
  "min_api_version": "0.2.0",
  "requires": { "privileges": ["input"] },
  "dev": { "build": [["go", "build", "-o", "../my-plugin-plugin", "."]], "build_dir": "src" },
  "run": "./my-plugin-plugin",
  "action_prefix": "myplugin",
  "action_types": {
    "greet": { "label": "Greet", "fields": [{ "key": "name", "label": "Name", "field_type": "string" }] }
  },
  "collection_data": { "voice_commands": "commands.json" },
  "implements": { "on_action": true }
}
```

`commands.json` maps a spoken phrase to an action:

```json
[
  {
    "pattern": ["hello", "branchkit"],
    "action": { "type": "myplugin.greet", "params": { "name": "BranchKit" } },
    "description": "Say Hello BranchKit"
  }
]
```

`src/main.go` handles the action:

```go
package main

import "github.com/branchkit/plugin-sdk-go"

func main() {
	plugin := branchkit.NewPlugin()

	// HandleGreet and GreetParams are generated from plugin.json into
	// actions_gen.go by `branchkit-gen --plugin .`.
	HandleGreet(plugin, func(p GreetParams, _ *branchkit.OnActionRequest) (any, error) {
		name := "BranchKit"
		if p.Name != nil {
			name = *p.Name
		}
		return nil, plugin.InputTypeText(branchkit.InputTypeTextRequest{Text: "Hello, " + name + "!"})
	})

	plugin.Run() // blocks until BranchKit stops the plugin
}
```

Say "hello branchkit" and the plugin types `Hello, BranchKit!` at the cursor.
Typing needs the `input` privilege, which is why the manifest asks for it.

## Calling the platform

Every platform method has a generated Go method on `*Plugin` that takes the
method's request struct and returns its typed result:

```go
rec, err := plugin.CollectionFetch(branchkit.CollectionFetchRequest{ID: id, Name: "notes"})

err = plugin.HUDCreateChannel(branchkit.HUDCreateChannelRequest{
	Channel:      "status",
	AcceptsInput: branchkit.Ptr(true), // optional fields are pointers
})
```

Arguments are always named fields, never positional, so each one says what it
is and a wrong name fails the build. Leave an optional field out to mean
"absent"; set one with `branchkit.Ptr(v)`. The generated methods are in
[methods_gen.go](methods_gen.go) and their request and result types in
[types_gen.go](types_gen.go), with the platform's own description on every
field.

**The manifest is the permission.** A call that needs a privilege the manifest
did not declare is refused before it runs, with an error of kind `forbidden`
naming the operation. Some privileges also ask the user the first time. Network
access works the same way: a host not listed under `requires.network` is
refused with a `*HostRefusedError`.

**Errors** from the platform are `*branchkit.RPCError` (`Code`, `Message`,
`Kind`, `Data`). Test for the common kinds with `errors.Is`:

```go
if errors.Is(err, branchkit.ErrForbidden) { /* declare the privilege */ }
```

`ErrNotFound`, `ErrNotPermitted`, `ErrRecordingDisabled` and `ErrUnsupported`
(a method this OS does not provide) work the same way.

`plugin.Call(method, params, &out)` is the untyped escape hatch, for a method
too new to have a generated wrapper. Prefer the wrapper whenever one exists.

## What the SDK covers

| Need | API |
|---|---|
| Handle an action | generated `Handle<Action>` (from `action_types`), `HandleAction`, `HandleActionTyped[T]` |
| Serve your own method | `Handle`, `HandleTyped[Req]`, `HandleCommand[Req]` |
| React to events | `On(event, fn)`, `OnPattern("ext.acme.**", fn)`, `CurrentEventOrigin()`; emit with `EventsEmit` |
| Store state | `Get` / `List` / `ListPage` / `Count` / `Put` / `PutMany` / `Patch` / `Delete` / `Replace`, `Subscribe` |
| Append-only logs | `Append`, `AppendKeyed`, `ListLog`, `GetLogEntry`, `DeleteLogEntry` |
| Keep a live copy | `MirrorCollection(name)`, `Settings[T](plugin, name)` |
| Contribute commands | `Command(Word("open"), Capture("app", "apps"))…Build()`, `PushCommandSpecs`, `PushCommandGroup` |
| A settings tab | `SettingsTab(key, fn)` + `implements.settings_tabs` in the manifest; buttons in package `ui` |
| Show something | `OutputState(OutputStateRequest{…})` with `SayAction` / `DispatchAction`, `HUDPush` |
| Hold a system effect | `AssertEffect`, `RetractEffect`, `IsEffectActive`, `OnEffectDisplaced` |
| Trace a request | `CurrentCorrelation()`, `RunWithCorrelation(id, fn)` |
| Log | `Info` / `Warn` / `Error` / `Debug` / `Trace(tag, data)` to your plugin's log (Debug and Trace are off by default) |
| Outbound HTTP | plain `net/http` (the default transport goes through the platform's proxy), `NewUpstreamClient(baseURL)` |
| Raw TCP (MQTT, a local daemon) | `Dial(host, port)`, `DialContext` |
| Accept local connections | `ListenLocal(plugin)` with `requires.sockets.listen` |
| Test a plugin | package `harness` |

Package `pipeline` is for pipeline stages (audio and monitor processes on a
separate wire), not for ordinary plugins.

## Testing a plugin

Package `harness` loads your plugin against a simulated platform and matches
phrases the way the real matcher does, without audio:

```go
func TestGreetMatches(t *testing.T) {
	h := harness.Start(t, "..")
	result := h.MustSimulateCommand("hello branchkit")
	if result.ActionType() != "myplugin.greet" {
		t.Fatalf("got %q", result.ActionType())
	}
}
```

It runs the `branchkit-test-harness` binary that ships inside BranchKit.app;
set `BRANCHKIT_TEST_HARNESS` to its path anywhere else. `go test ./...` runs your
tests; `branchkit-cli dev test .` checks the manifest and runs the platform's
own conformance checks against the plugin.

Against the running app:

```sh
branchkit-cli plugin install . --build            # install it
branchkit-cli dev watch .                          # rebuild and reload on save
branchkit-cli dev say "hello branchkit" --simulate # match and report, execute nothing
branchkit-cli dev plog my-plugin --since 30s       # read its log
```

## Learn more

- **API reference:** [pkg.go.dev/github.com/branchkit/plugin-sdk-go](https://pkg.go.dev/github.com/branchkit/plugin-sdk-go).
- **Local docs:** `branchkit-cli docs path` prints the documentation bundled
  with your installed BranchKit, for reading or grepping offline.
- **Worked examples:** [helloworld-go](https://github.com/branchkit/branchkit-plugin-helloworld-go)
  (the scaffold), and real plugins built on this SDK:
  [keyboard](https://github.com/branchkit/branchkit-plugin-keyboard),
  [system](https://github.com/branchkit/branchkit-plugin-system),
  [windows](https://github.com/branchkit/branchkit-plugin-windows).

## Versioning

Tags follow semver, 0.x for now: a minor release may break callers, and
CHANGELOG.md names every break with its migration. The Go, TypeScript
([plugin-sdk-ts](https://github.com/branchkit/plugin-sdk-ts)) and Python
([plugin-sdk-py](https://github.com/branchkit/plugin-sdk-py)) SDKs implement the
same surface and are held to it by one cross-language conformance suite.

## Contributing

[Issues](https://github.com/branchkit/plugin-sdk-go/issues/new/choose) are welcome:
a bug in the SDK or its docs, or, most useful, something you tried to build
and couldn't, with what you needed from the platform. You don't need to know
how BranchKit is built to tell us that.

We don't take pull requests for code yet. Much of each SDK is generated from
BranchKit's platform contracts, which aren't public, and the three are kept in
step across Go, TypeScript and Python, so the maintainers make each change in
all three at once. Files ending in `_gen.go` are generated; don't edit them by hand.

Found a security problem, such as a way around the sandbox or a permission
check? Please [report it privately](https://github.com/branchkit/plugin-sdk-go/security/advisories/new),
not in a public issue.

To run this SDK's own tests:

```sh
CGO_ENABLED=1 go test -race ./...
```
