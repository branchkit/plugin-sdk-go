package branchkit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// The typed event helpers in events_gen.go (OnAppFocused, OnPowerChanged,
// …) register through these two. Each ties a platform event's name to its
// generated payload type, so a renamed event or field is a build error
// rather than a listener that silently hears nothing.

// eventLog is where a payload that does not decode is reported; a test
// swaps it.
var eventLog io.Writer = os.Stderr

// onEvent registers fn for event, with the payload decoded into T. A
// payload that does not decode skips fn and logs one line naming the event
// and the payload's size, never the decode error: a json error can quote
// the input, and a payload may carry what the user said or typed.
func onEvent[T any](p *Plugin, event string, fn func(T)) {
	p.On(event, func(params json.RawMessage) {
		var evt T
		if len(params) > 0 {
			if err := json.Unmarshal(params, &evt); err != nil {
				fmt.Fprintf(eventLog, "[branchkit-sdk] %s: a %d-byte payload did not decode; listener skipped\n", event, len(params))
				return
			}
		}
		fn(evt)
	})
}

// onSignal registers fn for an event whose payload is not handed over: one
// with none, or a signal, whose payload is advisory and whose listener
// re-reads the state it announces.
func onSignal(p *Plugin, event string, fn func()) {
	p.On(event, func(json.RawMessage) { fn() })
}
