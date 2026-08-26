package branchkit

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// Ambient inbound-correlation tracking.
//
// The actuator carries correlation in a thread-local read via
// `correlation::current()` and stamps every event it emits from the live
// scope. The Go SDK mirrors that so a handler can thread the inbound id
// forward without any signature change: each actuator→plugin request and
// notification runs in its own goroutine (see `dispatch`), so the inbound
// envelope `correlation_id` is stashed keyed by goroutine id for the duration
// of the call and read back by `CurrentCorrelation()` / outbound stamping.
//
// A handler that spawns its own goroutines does not propagate the id: the new
// goroutine has a different id and so a different entry. Use
// [RunWithCorrelation] at the top of the spawned function to carry it across.
//
// This is a Go-specific wart, not a platform one. The TS SDK gets propagation
// free from `AsyncLocalStorage` and the Python SDK from `contextvars`, both of
// which follow the async context into work spawned inside a handler. Go has no
// equivalent ambient, so it has to be asked for.
var ambientCorrelation sync.Map // map[int64]string

func setAmbientCorrelation(id string) {
	if id == "" {
		return
	}
	ambientCorrelation.Store(goroutineID(), id)
}

func clearAmbientCorrelation() {
	ambientCorrelation.Delete(goroutineID())
}

func currentCorrelation() string {
	if v, ok := ambientCorrelation.Load(goroutineID()); ok {
		return v.(string)
	}
	return ""
}

// CurrentCorrelation returns the inbound correlation id for the actuator→plugin
// request or notification currently being handled on this goroutine, or "" if
// none is in flight. Handlers use it to tie their own work (logs, plugin-side
// transports) back to the upstream causal chain; outbound calls inherit it
// automatically, so most handlers never need to read it explicitly.
func (p *Plugin) CurrentCorrelation() string {
	return currentCorrelation()
}

// goroutineID parses the calling goroutine's numeric id out of its stack
// header ("goroutine 123 [running]:"). Go exposes no public accessor; this is
// the established idiom and runs only on handler entry/exit + outbound stamping,
// not on a tight loop.
func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	line := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	idx := bytes.IndexByte(line, ' ')
	if idx < 0 {
		return 0
	}
	id, _ := strconv.ParseInt(string(line[:idx]), 10, 64)
	return id
}

// RunWithCorrelation runs fn with id as the ambient inbound correlation for
// the calling goroutine, restoring whatever was ambient before.
//
// The case this exists for is a handler that spawns work and dispatches from
// it — the reply must not wait on a nested round trip, so the call happens on
// a new goroutine, where the ambient id would otherwise be absent and the
// causal chain would break:
//
//	corr := plugin.CurrentCorrelation()
//	go branchkit.RunWithCorrelation(corr, func() {
//	    plugin.Call("dispatch", args, &resp) // stamps corr on the envelope
//	})
//
// An empty id runs fn with no ambient rather than installing a blank one, so
// the emits carry nothing instead of carrying something meaningless — the
// same posture as the actuator's `correlation::current()`.
//
// Named to match the TS SDK's runWithCorrelation, which is the same idea
// expressed through AsyncLocalStorage.
func RunWithCorrelation(id string, fn func()) {
	if id == "" {
		fn()
		return
	}
	prev := currentCorrelation()
	setAmbientCorrelation(id)
	defer func() {
		// Restore rather than clear: a nested RunWithCorrelation must not
		// strip the outer scope on its way out.
		if prev == "" {
			clearAmbientCorrelation()
			return
		}
		setAmbientCorrelation(prev)
	}()
	fn()
}
