package branchkit

import "encoding/json"

// HUD push sugar. The generated HUDPush takes `fragments json.RawMessage`,
// which proved awkward enough that every caller in the fleet hand-rolled
// `p.Call("hud.push", map[string]any{...})` instead — a generated wrapper
// with zero users is a failed surface. These cover the two real shapes.

// HUDPushFragment morphs `html` into the element with id `targetID`
// inside the named HUD window. This is the shape that sizes the window
// from its content — pushing with an empty target and raw replacement
// leaves the window 1px tall (a real shipped bug).
func (p *Plugin) HUDPushFragment(channel, targetID, html string) error {
	fragments, err := json.Marshal([]map[string]any{
		{"target_id": targetID, "html": html},
	})
	if err != nil {
		return err
	}
	return p.HUDPush(channel, fragments)
}

// HUDPushRaw replaces the HUD window's entire content with `html`
// (`raw: true`) — for windows whose markup carries its own container.
func (p *Plugin) HUDPushRaw(channel, html string) error {
	fragments, err := json.Marshal([]map[string]any{
		{"target_id": "", "html": html, "raw": true},
	})
	if err != nil {
		return err
	}
	return p.HUDPush(channel, fragments)
}
