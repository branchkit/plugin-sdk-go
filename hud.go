package branchkit

// HUD push sugar. The generated HUDPush used to take
// `fragments json.RawMessage`, which proved awkward enough that every
// caller in the fleet hand-rolled `p.Call("hud.push", map[string]any{...})`
// instead — a generated wrapper with zero users is a failed surface. It
// takes `[]HudFragment` now (2026-09-19), so the wrapper is usable
// directly; these two stay because they carry a lesson the type cannot.

// HUDPushFragment morphs `html` into the element with id `targetID`
// inside the named HUD window. This is the shape that sizes the window
// from its content — pushing with an empty target and raw replacement
// leaves the window 1px tall (a real shipped bug).
func (p *Plugin) HUDPushFragment(channel, targetID, html string) error {
	return p.HUDPush(channel, []HudFragment{{TargetID: targetID, HTML: html}})
}

// HUDPushRaw replaces the HUD window's entire content with `html`
// (`raw: true`) — for windows whose markup carries its own container.
func (p *Plugin) HUDPushRaw(channel, html string) error {
	raw := true
	return p.HUDPush(channel, []HudFragment{{TargetID: "", HTML: html, Raw: &raw}})
}
