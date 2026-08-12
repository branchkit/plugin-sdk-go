package shared

import (
	"fmt"
	"os"
	"strings"
)

// The one route the actuator serves for settings-UI → plugin method calls.
// Registered in actuator/src/api/router.rs as
// `/v1/plugins/{plugin_id}/methods/{*method_path}`. Hand-writing this shape in
// a templ literal is how four plugins ended up with dead settings tabs: two
// invented a `/hooks/` segment that no route matches, one omitted `/methods/`
// entirely, and one still named a plugin id that was renamed years ago. The
// segment lives here so there is one spelling of it.
const methodRoutePrefix = "/v1/plugins/"

// MethodURL returns the settings-UI route that invokes `method` on this
// plugin. The plugin id comes from BRANCHKIT_PLUGIN_ID (set by the actuator
// when it spawns the process), so a renamed plugin cannot desync its own URLs.
//
// The actuator normalizes `-` and `/` to `_` before dispatch, so
// MethodURL("set-gap") and MethodURL("set_gap") both reach the handler
// registered as `set_gap`.
func MethodURL(method string) string {
	id := os.Getenv("BRANCHKIT_PLUGIN_ID")
	if id == "" {
		id = "unknown"
	}
	return methodRoutePrefix + id + "/methods/" + strings.TrimPrefix(method, "/")
}

// MethodPost returns the Datastar `@post(...)` expression that invokes
// `method` on this plugin, for use in a `data-on:click` attribute.
//
// payloadJS is a JavaScript object literal for the request params, or "" for a
// method that takes none. It is embedded verbatim — callers building it from
// user-controlled strings must run them through toolkit.JSEscape first, the
// same as any other inline expression.
//
//	<button data-on:click={ shared.MethodPost("set_auto_tile", "{enabled: true}") }>
//	<button data-on:click={ shared.MethodPost("reset", "") }>
func MethodPost(method, payloadJS string) string {
	if payloadJS == "" {
		return fmt.Sprintf("@post('%s')", MethodURL(method))
	}
	return fmt.Sprintf("@post('%s', {payload: %s})", MethodURL(method), payloadJS)
}
