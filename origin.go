package branchkit

import "sync"

// EventOrigin names who sent the event notification a listener is handling.
//
// The platform delivers an event to every plugin whose manifest subscription
// matches it, and the event type alone does not say who emitted it: a
// subscription to `*.focused` hears every plugin's `focused`, and a host that
// relays its hosted things' events wants to know the event really came from
// itself. The actuator puts the sender on the notification's envelope, and the
// SDK makes it readable from inside the listener through CurrentEventOrigin —
// the same ambient shape as CurrentCorrelation, so no listener signature
// changes.
type EventOrigin struct {
	// Source is the emitter the platform authenticated: a plugin id,
	// "_platform" for platform events, or a stage's name for `ext.*` events.
	// The platform force-sets it from the emitting connection, so a listener
	// can trust it. "" outside an event listener, or from an actuator that
	// predates the field.
	Source string
	// OnBehalfOf is the emitter's actor label — which hosted thing it said it
	// was acting for (see ActOnBehalfOf), or "" if none. A CLAIM by Source,
	// never checked by the platform: trust it exactly as far as you trust
	// Source.
	OnBehalfOf string
}

// Ambient inbound event origin, keyed per goroutine exactly like the ambient
// correlation (see correlation.go): notifications are delivered one at a time
// on the SDK's notification goroutine, so the entry is set for the duration of
// one delivery and removed after it.
var ambientOrigin sync.Map // map[int64]EventOrigin

func setAmbientOrigin(o EventOrigin) {
	ambientOrigin.Store(goroutineID(), o)
}

func clearAmbientOrigin() {
	ambientOrigin.Delete(goroutineID())
}

func currentOrigin() EventOrigin {
	if v, ok := ambientOrigin.Load(goroutineID()); ok {
		return v.(EventOrigin)
	}
	return EventOrigin{}
}

// CurrentEventOrigin returns the sender of the event notification being
// handled on this goroutine — for On and OnPattern listeners — or the zero
// EventOrigin when none is in flight (request handlers, and any goroutine a
// listener spawns).
//
// A listener that hands work to another goroutine should read it first and
// pass the value along; unlike the correlation id, nothing outbound depends on
// it, so there is no Run helper to carry it.
//
//	plugin.OnPattern("scripts.*.*", func(eventType string, payload json.RawMessage) {
//	    if plugin.CurrentEventOrigin().Source != "scripts" {
//	        return // not the host this listener relays for
//	    }
//	    ...
//	})
func (p *Plugin) CurrentEventOrigin() EventOrigin {
	return currentOrigin()
}
