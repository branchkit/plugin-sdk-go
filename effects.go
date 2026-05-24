package shared

import (
	"encoding/json"
	"fmt"
)

// derefOr returns *p or the zero value when p is nil.
func derefOr[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// EffectDisplacedEvent is the payload delivered to OnEffectDisplaced
// callbacks. Mirrors the actuator-side broadcast shape — see
// `actuator/src/operations/registered/effects.rs`.
type EffectDisplacedEvent struct {
	// Effect that was displaced (e.g. "suppress_notifications").
	Effect string `json:"effect"`
	// Plugin that displaced this one and now holds top-of-stack.
	NewOwner string `json:"new_owner"`
	// Plugin that lost top-of-stack ownership. The SDK filters on this
	// so `OnEffectDisplaced` only fires when *this* plugin was displaced;
	// the field is exposed for plugins that subscribe to the underlying
	// event directly via `On(EventEffectDisplaced, ...)`.
	DisplacedOwner string `json:"displaced_owner"`
}

// AssertEffect declares this plugin is asserting `name`. The plugin must
// have declared this effect in its manifest's `provides.effects[*].asserts`
// — undeclared effects return an error.
//
// Returns:
//   - granted: true when the assertion is now top-of-stack OR the plugin
//     already held it (idempotent re-assert). False once user-consent
//     revocation lands and the user has revoked this effect on this
//     plugin (consent surface is post-Step-2 work).
//   - alreadyHeld: true when the plugin already had an active assertion
//     on this effect. Implies granted=true.
//   - displaced: name of the previous top-of-stack owner if this assertion
//     overrode someone, "" otherwise.
//
// See notes/DESIGN_CAPABILITY_MECHANISM.md for the mechanism design.
func (p *Plugin) AssertEffect(name string) (granted bool, alreadyHeld bool, displaced string, err error) {
	res, err := p.EffectsAssert(name)
	if err != nil {
		return false, false, "", err
	}
	if res == nil {
		return false, false, "", fmt.Errorf("effects.assert returned nil response")
	}
	return res.Granted, res.AlreadyHeld, derefOr(res.Displaced), nil
}

// RetractEffect releases this plugin's assertion of `name`. Idempotent —
// retracting an effect this plugin doesn't hold returns retracted=false
// with no error.
//
// `newOwner` names the effective owner after the call, or "" when the
// stack is now empty.
func (p *Plugin) RetractEffect(name string) (retracted bool, newOwner string, err error) {
	res, err := p.EffectsRetract(name)
	if err != nil {
		return false, "", err
	}
	if res == nil {
		return false, "", fmt.Errorf("effects.retract returned nil response")
	}
	return res.Retracted, derefOr(res.NewOwner), nil
}

// IsEffectActive returns true when this plugin currently holds top-of-
// stack for `name` (i.e. is the effective owner). `currentOwner` names
// the actual top-of-stack regardless of whether it's this plugin —
// useful for surfacing "Meeting Mode is overriding Focus Mode" UI.
//
// Unknown effect names resolve to (false, "", nil) — same shape as an
// empty stack — so polling on a typo'd name doesn't require error
// handling.
func (p *Plugin) IsEffectActive(name string) (active bool, currentOwner string, err error) {
	res, err := p.EffectsIsActive(name)
	if err != nil {
		return false, "", err
	}
	if res == nil {
		return false, "", fmt.Errorf("effects.is_active returned nil response")
	}
	return res.Active, derefOr(res.CurrentOwner), nil
}

// OnEffectDisplaced registers a callback fired when this plugin's
// assertion is overridden by a later asserter.
//
// Delivery is broadcast — every plugin subscribed to
// `_platform.effect.displaced` receives every displacement event. This
// helper filters on `displaced_owner == this plugin's id` so the
// callback only fires for *this* plugin's displacements. Plugins that
// want to observe all displacements (e.g. a UI showing system effect
// state) should subscribe directly via
// `On(EventEffectDisplaced, ...)`.
//
// See `notes/DESIGN_CAPABILITY_MECHANISM.md` section 10.2.
//
// Multiple callbacks can be registered; each fires for every event.
func (p *Plugin) OnEffectDisplaced(handler func(evt EffectDisplacedEvent)) {
	p.On(EventEffectDisplaced, func(params json.RawMessage) {
		if len(params) == 0 {
			return
		}
		var evt EffectDisplacedEvent
		if err := json.Unmarshal(params, &evt); err != nil {
			// Malformed payload — drop silently rather than crashing the
			// listener thread. The actuator's emit shape is fixed at
			// the source and any change goes through codegen, so a
			// shape mismatch here means the SDK is older than the
			// actuator and the plugin should be rebuilt.
			return
		}
		if evt.DisplacedOwner != p.pluginID {
			return
		}
		handler(evt)
	})
}

