package branchkit

import "encoding/json"

// Platform is what this machine can do: the OS, the Linux desktop session,
// the host, every operation a call to which would be refused here (with the
// refusal's own reason), and the host events that never fire. The actuator
// sends it with on_ready, so it is set by the time an OnReady callback runs;
// nil before, unless Supports has already fetched it.
func (p *Plugin) Platform() *PlatformProfileResponse {
	p.platformMu.Lock()
	defer p.platformMu.Unlock()
	return p.platform
}

// Supports reports whether calling method can succeed on this machine, as
// far as the platform goes: false only when the profile lists it as
// unavailable. Asking first lets a plugin degrade deliberately — hide a
// command, pick another route — instead of handling a refusal after the
// fact. A method the profile does not list is supported, including one this
// SDK has never heard of.
//
// Before on_ready this fetches the profile once (platform.profile) and keeps
// it; if that fails it answers true, and the call itself will say.
func (p *Plugin) Supports(method string) bool {
	profile := p.Platform()
	if profile == nil {
		fetched, err := p.PlatformProfile()
		if err != nil {
			return true
		}
		p.platformMu.Lock()
		if p.platform == nil {
			p.platform = fetched
		}
		profile = p.platform
		p.platformMu.Unlock()
	}
	for _, u := range profile.Unavailable {
		if u.Op == method {
			return false
		}
	}
	return true
}

// takePlatform keeps on_ready's `platform` param. An actuator older than the
// profile sends none; Supports then fetches it on first use.
func (p *Plugin) takePlatform(params json.RawMessage) {
	var ready struct {
		Platform *PlatformProfileResponse `json:"platform"`
	}
	if json.Unmarshal(params, &ready) != nil || ready.Platform == nil {
		return
	}
	p.platformMu.Lock()
	p.platform = ready.Platform
	p.platformMu.Unlock()
}
