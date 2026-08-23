package branchkit

import "sync"

// The actor label — who this plugin is acting *on behalf of*.
//
// A host-shaped plugin runs things the platform does not model: the scripts
// host runs script files, the browser plugin fronts an extension, an
// ambassador fronts an external app. Every platform call it makes is its
// own, over its own session, so audit rows, collection records, and events
// can name only the plugin. ActOnBehalfOf stamps the finer-grained label on
// the outbound envelope, and the platform carries it into what it writes.
//
// Observability only, by construction. The platform never consults the
// label for any decision — it cannot, because the host supplies it from
// inside its own process, so a lying host would only lie about a label it
// already had the grant to act under. Setting it widens nothing and
// narrows nothing; it makes the trail readable. Per-hosted-thing
// ENFORCEMENT would need real delegated identities, which is a different
// (unbuilt) feature.
//
// Scoped per goroutine, exactly like the ambient inbound correlation id
// above it — each actuator→plugin request and notification runs on its own
// goroutine, so a host that wraps its per-hosted-thing dispatch gets the
// label on every call that dispatch makes. A handler that spawns its own
// goroutines does not propagate it.
var ambientActor sync.Map // map[int64]string

func currentActor() string {
	if v, ok := ambientActor.Load(goroutineID()); ok {
		return v.(string)
	}
	return ""
}

// ActOnBehalfOf marks outbound RPC from this goroutine as performed on
// behalf of actor, until the returned function is called. The idiom is
// `defer ActOnBehalfOf(name)()` at the top of whatever runs hosted code:
//
//	func (s *Script) dispatch(action string, params map[string]any) error {
//		defer branchkit.ActOnBehalfOf(s.file)()
//		return s.host.plugin.Dispatch(...)
//	}
//
// An empty actor is a no-op returning a no-op — a host with nothing to
// declare needs no special case. Nested calls restore the previous label,
// so one hosted thing invoking another leaves the trail intact.
func ActOnBehalfOf(actor string) func() {
	if actor == "" {
		return func() {}
	}
	gid := goroutineID()
	prev, had := ambientActor.Load(gid)
	ambientActor.Store(gid, actor)
	return func() {
		if had {
			ambientActor.Store(gid, prev)
		} else {
			ambientActor.Delete(gid)
		}
	}
}

// CurrentActor returns the actor label outbound calls on this goroutine are
// currently stamped with, or "" if none. Hosts read it to tag their own
// logs with the same name the platform will record.
func (p *Plugin) CurrentActor() string {
	return currentActor()
}
