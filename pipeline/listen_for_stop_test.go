package pipeline

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/branchkit/plugin-sdk-go/pipeline/audio"
)

// SourceOptions.ListenForStop is opt-in and no stage in this tree turns it on,
// so until these tests its ON path had never executed anywhere — the field
// carried a `//deadfield:ignore` saying exactly that. Unused platform surface
// is fine; unused AND never-run is a promise nobody has checked, which is the
// distinction that cost two first-party plugins real work when `render_hud`
// turned out to be unreachable (docs/design/DESIGN_HUD_PULL_MODEL.md in app).
//
// ServeSourceOn exists for this: explicit transports, no subprocess.

// stopFrame renders one framed audio_stop the way the platform's runner does.
func stopFrame(t *testing.T, sessionID string, cutoff *uint64) []byte {
	t.Helper()
	data, err := json.Marshal(audio.AudioStop{SessionId: sessionID, CutoffMs: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := NewWriter(&buf).WriteEvent(&Event{Type: audio.EventAudioStop, Data: data}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func srcCapability() Capability {
	return Capability{StageType: "monitor", StageName: "test-source", LifecycleModes: []string{"persistent"}}
}

// The whole point of the flag: an inbound audio_stop ends the body, and the
// stop it carries is readable — including CutoffMs, which an audio source must
// forward verbatim downstream.
func TestListenForStop_StopsBodyAndExposesTheRequest(t *testing.T) {
	cutoff := uint64(4321)
	in := bytes.NewReader(stopFrame(t, "sess-7", &cutoff))

	var got *audio.AudioStop
	err := ServeSourceOn(in, io.Discard, srcCapability(), SourceOptions{ListenForStop: true},
		func(ctx *SourceCtx) error {
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				t.Error("ListenForStop did not stop the body")
			}
			got = ctx.StopRequest()
			return nil
		})
	if err != nil {
		t.Fatalf("ServeSourceOn: %v", err)
	}
	if got == nil {
		t.Fatal("StopRequest() is nil — the stop arrived but was not recorded")
	}
	if got.SessionId != "sess-7" {
		t.Errorf("SessionId = %q, want sess-7", got.SessionId)
	}
	if got.CutoffMs == nil || *got.CutoffMs != cutoff {
		t.Errorf("CutoffMs = %v, want %d — a source forwards this verbatim", got.CutoffMs, cutoff)
	}
}

// Off by default is the documented behaviour, and it is what every stage in
// this tree relies on: the same bytes must not stop a source that did not ask.
func TestListenForStop_OffIgnoresTheSameBytes(t *testing.T) {
	in := bytes.NewReader(stopFrame(t, "sess-7", nil))

	stopped := make(chan struct{})
	err := ServeSourceOn(in, io.Discard, srcCapability(), SourceOptions{},
		func(ctx *SourceCtx) error {
			select {
			case <-ctx.Done():
				close(stopped)
			case <-time.After(150 * time.Millisecond):
			}
			return nil
		})
	if err != nil {
		t.Fatalf("ServeSourceOn: %v", err)
	}
	select {
	case <-stopped:
		t.Error("an audio_stop stopped a source with ListenForStop off")
	default:
	}
}

// EOF means the platform is gone, so the body must end even though no stop
// event ever arrived — and StopRequest stays nil, since nothing asked.
func TestListenForStop_EOFStopsWithNoRequest(t *testing.T) {
	var got *audio.AudioStop
	err := ServeSourceOn(bytes.NewReader(nil), io.Discard, srcCapability(),
		SourceOptions{ListenForStop: true},
		func(ctx *SourceCtx) error {
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				t.Error("EOF did not stop the body")
			}
			got = ctx.StopRequest()
			return nil
		})
	if err != nil {
		t.Fatalf("ServeSourceOn: %v", err)
	}
	if got != nil {
		t.Errorf("StopRequest() = %+v, want nil — the stop came from EOF, not the platform", got)
	}
}

// A source must tolerate inbound bytes it does not recognise rather than die
// on them; watchForStop skips to the stop it is waiting for.
func TestListenForStop_SkipsUnrelatedInboundEvents(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for _, ev := range []*Event{
		{Type: "ext.someone.else", Data: json.RawMessage(`{"n":1}`)},
		{Type: audio.EventAudioStart, Data: json.RawMessage(`{"session_id":"x"}`)},
	} {
		if err := w.WriteEvent(ev); err != nil {
			t.Fatal(err)
		}
	}
	buf.Write(stopFrame(t, "sess-9", nil))

	var got *audio.AudioStop
	err := ServeSourceOn(bytes.NewReader(buf.Bytes()), io.Discard, srcCapability(),
		SourceOptions{ListenForStop: true},
		func(ctx *SourceCtx) error {
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				t.Error("the stop behind two unrelated events never arrived")
			}
			got = ctx.StopRequest()
			return nil
		})
	if err != nil {
		t.Fatalf("ServeSourceOn: %v", err)
	}
	if got == nil || got.SessionId != "sess-9" {
		t.Errorf("StopRequest() = %+v, want the sess-9 stop", got)
	}
}
