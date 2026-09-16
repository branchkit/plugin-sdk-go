package branchkit

import (
	"encoding/json"
	"fmt"
)

// SettingsTabFunc renders one settings tab. It returns the tab's HTML
// FRAGMENT — the platform's frame owns the container the fragment is
// morphed into, and the SDK attaches the stylesheet registered with
// [Plugin.SettingsCSS]. An error is rendered by the platform as the tab's
// error state, so a renderer that cannot draw should say why rather than
// return an empty string.
type SettingsTabFunc func(req *RenderSettingsRequest) (string, error)

// settingsRefresher is what the render hook needs from a settings mirror:
// a synchronous read-through. Kept as an interface so the hook does not
// care about T.
type settingsRefresher interface {
	Refresh() error
}

// SettingsTab registers the renderer for the manifest-declared settings
// tab `key`. The first call installs the SDK's own `render_settings`
// handler, which on every render:
//
//  1. refreshes every settings mirror created on this plugin with
//     [Settings], so the render reads state at least as fresh as whatever
//     woke it — the stream re-renders on collection writes that may
//     arrive before the mirror's collection.updated does;
//  2. dispatches on `tab_key`; a key with no renderer is an error, which
//     the platform shows as the tab's error state instead of a blank body;
//  3. returns the fragment with the registered stylesheet.
//
// The platform's method proxy discards a settings method's result and
// answers 204: a method that changed something returns nil and lets the
// re-render that follows draw it. Rendering inside a method is wasted.
//
// SettingsTab and Handle("render_settings", …) are mutually exclusive —
// both install a handler for the same RPC method. Calling either after
// the other panics, regardless of order.
func (p *Plugin) SettingsTab(key string, fn SettingsTabFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.settingsTabs == nil {
		if _, taken := p.handlers[HookRenderSettings]; taken {
			panic("plugin-sdk-go: cannot mix Handle(\"render_settings\", ...) and SettingsTab(...) — pick one")
		}
		p.settingsTabs = make(map[string]SettingsTabFunc)
		p.handlers[HookRenderSettings] = p.renderSettingsTab
	}
	p.settingsTabs[key] = fn
}

// SettingsCSS registers the stylesheet returned with every tab this
// plugin renders. One sheet per plugin: the platform places it in a
// `<style>` element it owns, outside the morph target, so it is sent
// once per change rather than inside every fragment.
func (p *Plugin) SettingsCSS(css string) {
	p.mu.Lock()
	p.settingsCSS = css
	p.mu.Unlock()
}

func (p *Plugin) renderSettingsTab(params json.RawMessage) (any, error) {
	var req RenderSettingsRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("%s: bad params: %w", HookRenderSettings, err)
		}
	}
	p.mu.Lock()
	fn := p.settingsTabs[req.TabKey]
	css := p.settingsCSS
	mirrors := append([]settingsRefresher(nil), p.settingsMirrors...)
	p.mu.Unlock()
	if fn == nil {
		return nil, fmt.Errorf("no renderer registered for settings tab %q", req.TabKey)
	}
	// Read through before drawing. A refresh failure is logged, not
	// fatal: the mirror keeps its last snapshot and the tab still draws.
	for _, m := range mirrors {
		if err := m.Refresh(); err != nil {
			Logf(p.pluginID, "settings read-through failed: %v", err)
		}
	}
	html, err := fn(&req)
	if err != nil {
		return nil, err
	}
	resp := RenderSettingsResponse{HTML: html}
	if css != "" {
		resp.CSS = &css
	}
	return resp, nil
}
