package shared

import "encoding/json"

// Trace writes a per-record diagnostic line at trace level. Dropped by
// default — surface by setting the plugin's threshold to `trace` via the
// Settings UI Debug tab or the `BRANCHKIT_LOG_PLUGIN` env var. Use for
// high-volume, per-record telemetry (every hint resolved, every audio
// frame, every cache hit) that you only want when actively debugging.
func (p *Plugin) Trace(tag string, data any) { p.logAtLevel("trace", tag, data) }

// Debug writes a tagged structured payload to this plugin's dedicated
// log at <app_support>/plugin-logs/<pluginID>.log at debug level.
//
// v1 surface — kept verbatim for backwards-compat. With v2's default
// threshold of `info`, plain Debug calls are dropped unless the plugin's
// threshold is lowered to `debug` or `trace`. Use Info for per-operation
// diagnostics you want visible by default; use Debug for verbose-only
// chatter. See notes/DESIGN_PLUGIN_LOG_LEVELS.md.
//
// Use shared.Logf instead for lines that describe coordination with the
// actuator or other plugins — those benefit from interleaving with the
// actuator's own match/dispatch/route lines in actuator.log.
//
// Best-effort. Returns no error: if the call fails the line is dropped,
// matching how stdio stderr writes from a plugin are handled. tag may
// be empty; data may be any JSON-serializable value (nil renders as
// `null`).
func (p *Plugin) Debug(tag string, data any) { p.logAtLevel("debug", tag, data) }

// Info writes a per-operation diagnostic line at info level. Visible by
// default (threshold defaults to `info`). Use for notable plugin-internal
// events: "STT model loaded", "browser extension connected".
func (p *Plugin) Info(tag string, data any) { p.logAtLevel("info", tag, data) }

// Warn writes a warning line. Goes to BOTH the per-plugin log and
// actuator.log (cross-posted via the `plugin.diagnostic` event) so
// plugin-level warnings interleave with the actuator's view of dispatch
// and coordination. Use for recoverable problems: "fingerprint stale,
// falling back to selector", "element-id mismatch, re-scanning".
func (p *Plugin) Warn(tag string, data any) { p.logAtLevel("warn", tag, data) }

// Error writes an error line. Goes to BOTH the per-plugin log and
// actuator.log. Use for unrecoverable problems within plugin scope:
// "STT model failed to load", "DOM scan threw".
func (p *Plugin) Error(tag string, data any) { p.logAtLevel("error", tag, data) }

// LogAt is a level-by-string helper for HTTP bridges that forward
// plugin-debug-log requests (e.g. the browser plugin's POST endpoint
// receives a `level` field from the extension). First-party plugin
// code should call Trace/Debug/Info/Warn/Error directly. Unknown
// level strings fall through to Debug rather than dropping silently.
func (p *Plugin) LogAt(level, tag string, data any) {
	switch level {
	case "trace", "debug", "info", "warn", "error":
		p.logAtLevel(level, tag, data)
	default:
		p.logAtLevel("debug", tag, data)
	}
}

func (p *Plugin) logAtLevel(level, tag string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	tagBytes, err := json.Marshal(tag)
	if err != nil {
		return
	}
	levelBytes, err := json.Marshal(level)
	if err != nil {
		return
	}
	_ = p.PluginDebug(json.RawMessage(payload), json.RawMessage(levelBytes), json.RawMessage(tagBytes))
}
