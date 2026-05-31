package shared

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
// A handler that spawns its own goroutines does not propagate the id — the
// same boundary the actuator has across `tokio::spawn`.
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
