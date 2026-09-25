package branchkit

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
)

// A listener can read who sent the event it is handling — both listener
// shapes, and only for the delivery it is inside.
func TestListenersReadTheEventOrigin(t *testing.T) {
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
	}
	var exact, patterned []EventOrigin
	p.On("scripts.headphones.charged", func(json.RawMessage) {
		exact = append(exact, p.CurrentEventOrigin())
	})
	p.OnPattern("scripts.*.*", func(string, json.RawMessage) {
		patterned = append(patterned, p.CurrentEventOrigin())
	})

	p.handleNotification(rpcMessage{
		Method:     "scripts.headphones.charged",
		Source:     "scripts",
		OnBehalfOf: "headphones.lua",
	})
	// A later event from another sender, and one carrying no origin at all
	// (an older actuator, or a notification that is not a bus event): each
	// listener sees ITS delivery's origin, never a previous one's.
	p.handleNotification(rpcMessage{Method: "scripts.headphones.charged", Source: "impostor"})
	p.handleNotification(rpcMessage{Method: "scripts.headphones.charged"})

	want := []EventOrigin{
		{Source: "scripts", OnBehalfOf: "headphones.lua"},
		{Source: "impostor"},
		{},
	}
	for name, got := range map[string][]EventOrigin{"On": exact, "OnPattern": patterned} {
		if len(got) != len(want) {
			t.Fatalf("%s saw %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s saw %v, want %v", name, got, want)
			}
		}
	}
	if o := p.CurrentEventOrigin(); o != (EventOrigin{}) {
		t.Fatalf("origin leaked past the delivery: %+v", o)
	}
}

// The origin is per goroutine, like the correlation id: work a listener
// spawns does not inherit it, and a request handler never sees one.
func TestEventOriginDoesNotCrossAGoroutine(t *testing.T) {
	setAmbientOrigin(EventOrigin{Source: "scripts"})
	defer clearAmbientOrigin()

	var got EventOrigin
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		got = currentOrigin()
	}()
	wg.Wait()

	if got != (EventOrigin{}) {
		t.Fatalf("a fresh goroutine should inherit nothing, got %+v", got)
	}
	if currentOrigin().Source != "scripts" {
		t.Error("the delivering goroutine should be unaffected")
	}
}

// End to end through the read loop: the envelope field the actuator writes is
// the one the listener reads.
func TestEventOriginArrivesOffTheWire(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()
	// Drain what the plugin writes (plugin.initialized): the pipe is
	// unbuffered, so an undrained write would stall Run.
	go func() {
		for actuatorR.Scan() {
		}
	}()

	seen := make(chan EventOrigin, 1)
	p.On("emitter.thing", func(json.RawMessage) {
		seen <- p.CurrentEventOrigin()
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); p.Run() }()

	actuatorW.Write([]byte(`{"jsonrpc":"2.0","method":"emitter.thing","params":{},` +
		`"source":"emitter","on_behalf_of":"lamp.py"}` + "\n"))

	if got := <-seen; got != (EventOrigin{Source: "emitter", OnBehalfOf: "lamp.py"}) {
		t.Fatalf("CurrentEventOrigin() = %+v", got)
	}

	actuatorW.(io.Closer).Close()
	wg.Wait()
}
