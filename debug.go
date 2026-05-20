package shared

import "encoding/json"

// Debug writes a tagged structured payload to this plugin's dedicated
// log at <app_support>/plugin-logs/<pluginID>.log.
//
// Use this for diagnostic chatter that aids debugging this plugin
// specifically but doesn't belong interleaved with the actuator's
// cross-cutting log (see CLAUDE.md and notes/DESIGN_PLUGIN_LOGGING.md).
//
// Use shared.Logf instead for lines that describe coordination with the
// actuator or other plugins — those benefit from interleaving with the
// actuator's own match/dispatch/route lines in actuator.log.
//
// Best-effort. Returns no error: if the call fails the line is dropped,
// matching how stdio stderr writes from a plugin are handled. tag may
// be empty; data may be any JSON-serializable value (nil renders as
// `null`).
func (p *Plugin) Debug(tag string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	tagBytes, err := json.Marshal(tag)
	if err != nil {
		return
	}
	_ = p.PluginDebug(json.RawMessage(payload), json.RawMessage(tagBytes))
}
