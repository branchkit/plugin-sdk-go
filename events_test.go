package branchkit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func eventPlugin() *Plugin {
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
		pluginID:  "test-plugin",
	}
	p.Methods = NewMethods(p)
	return p
}

func deliver(p *Plugin, event, raw string) {
	p.handleNotification(rpcMessage{Method: event, Params: json.RawMessage(raw)})
}

func TestTypedEventHelperDecodesThePayload(t *testing.T) {
	p := eventPlugin()
	var got []CollectionUpdatedEventParams
	p.OnCollectionUpdated(func(evt CollectionUpdatedEventParams) { got = append(got, evt) })
	deliver(p, EventCollectionUpdated, `{"collection":"keyboard.keys"}`)
	if len(got) != 1 || got[0].Collection != "keyboard.keys" {
		t.Fatalf("got %+v", got)
	}
}

func TestSignalHelperTakesNoPayload(t *testing.T) {
	p := eventPlugin()
	calls := 0
	p.OnPowerChanged(func() { calls++ })
	p.OnSystemWillSleep(func() { calls++ })
	deliver(p, EventPowerChanged, `{"on_battery":true}`)
	deliver(p, EventSystemWillSleep, ``)
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// A payload that does not decode skips the listener and logs its size,
// never its text: a json error can quote the input.
func TestUndecodablePayloadIsSkippedAndNotQuoted(t *testing.T) {
	var log bytes.Buffer
	prev := eventLog
	eventLog = &log
	defer func() { eventLog = prev }()

	p := eventPlugin()
	calls := 0
	p.OnPipelineTranscript(func(PipelineTranscriptEventParams) { calls++ })
	raw := `{"text": ["SECRETWORD"]}`
	deliver(p, EventPipelineTranscript, raw)
	if calls != 0 {
		t.Fatalf("listener ran on a payload that did not decode")
	}
	if !strings.Contains(log.String(), EventPipelineTranscript) {
		t.Fatalf("log does not name the event: %q", log.String())
	}
	if strings.Contains(log.String(), "SECRETWORD") {
		t.Fatalf("log quotes the payload: %q", log.String())
	}
}
