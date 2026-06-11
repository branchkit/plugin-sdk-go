# BranchKit Plugin SDK (Go)

The Go SDK for building [BranchKit](https://branchkit.dev) plugins —
native processes that add voice commands, window management, browser
integration, or anything else to the BranchKit platform. MIT licensed.

## Start here

- **[Your First Plugin](https://branchkit.dev/guide/getting-started/your-first-plugin)** —
  working plugin in ~10 minutes (`branchkit-cli dev init --template go`)
- **[Plugin Anatomy](https://branchkit.dev/guide/getting-started/plugin-anatomy)** —
  manifest, lifecycle, methods
- **[Plugin API Reference](https://branchkit.dev/reference/specs/plugin-api)** —
  every wire method, generated from the OpenRPC spec

## Minimal plugin

```go
package main

import shared "github.com/branchkit/plugin-sdk-go"

func main() {
    plugin := shared.NewPlugin()

    plugin.HandleAction("myplugin.greet", func(req *shared.OnActionRequest) (any, error) {
        plugin.Call("input.type_text", map[string]any{"text": "Hello!"}, nil)
        return nil, nil
    })

    plugin.Run()
}
```

Pair with a `plugin.json` manifest declaring the action — see the
tutorial. `branchkit-gen` generates typed param structs from your
manifest's `action_types`.

## Key surfaces

| Need | API |
|---|---|
| Handle dispatched actions | `HandleAction`, generated `actions_gen.go` |
| State (collections, 8 verbs) | `Get` / `List` / `Count` / `Put` / `Patch` / `Delete` / `Append` / events |
| Log-shaped collections | `Append`, `ListLog`, `GetLogEntry`, `DeleteLogEntry` (sugar over the verbs) |
| Commands & vocabulary | `CommandBuilder`, `CommandsPush` |
| Events | manifest `consumes.events` + `plugin.On(event, fn)` |
| Settings UI tab | `settings_tab` manifest field + render method |
| Logging | `shared.Logf` (shared actuator log), `plugin.Debug` (per-plugin file) |

## Conformance

Both official SDKs (Go and TypeScript) implement the same required
surface and pass a shared conformance harness. If you extend the SDK,
run it: `cargo run -p branchkit-sdk-test -- ./sdk-test/testplugin/testplugin`
from the BranchKit workspace.
