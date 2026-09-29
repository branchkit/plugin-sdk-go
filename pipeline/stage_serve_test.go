package pipeline

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/branchkit/plugin-sdk-go/pipeline/audio"
)

// The stage runtime (ServeAudioConsumerOn, ServeSourceOn, CreditGranter) is
// what every Go stage's flow control rests on: a wrong credit number starves
// or floods the platform's window. These tests drive it over in-memory
// transports — a framed input buffer in, the framed output read back — and
// assert the exact event sequence a stage puts on the wire.

// frames renders events the way the platform's runner writes them.
func frames(t *testing.T, evs ...*Event) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for _, ev := range evs {
		if err := w.WriteEvent(ev); err != nil {
			t.Fatal(err)
		}
	}
	return &buf
}

func typed(t *testing.T, eventType string, data any, payload []byte) *Event {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return &Event{Type: eventType, Data: raw, Payload: payload}
}

func startEv(t *testing.T, session string) *Event {
	return typed(t, audio.EventAudioStart, audio.AudioStart{
		SessionId: session,
		Format:    audio.AudioFormat{Channels: 1, Rate: 16000, Width: 2},
	}, nil)
}

func chunkEv(t *testing.T, session string, ts uint64) *Event {
	return typed(t, audio.EventAudioChunk, audio.AudioChunk{SessionId: session, TimestampMs: ts}, []byte{1, 2, 3, 4})
}

func stopEv(t *testing.T, session string) *Event {
	return typed(t, audio.EventAudioStop, audio.AudioStop{SessionId: session}, nil)
}

// readAll decodes every framed event a stage wrote, until EOF.
func readAll(t *testing.T, out *bytes.Buffer) []*Event {
	t.Helper()
	r := NewReader(out)
	var evs []*Event
	for {
		ev, err := r.ReadEvent()
		if errors.Is(err, io.EOF) {
			return evs
		}
		if err != nil {
			t.Fatalf("read stage output: %v", err)
		}
		evs = append(evs, ev)
	}
}

type credit struct {
	session string
	frames  uint32
}

// splitOutput checks the first event is the capability handshake and returns
// the flow_credit events after it, failing on anything else.
func splitOutput(t *testing.T, evs []*Event) (Capability, []credit) {
	t.Helper()
	if len(evs) == 0 || evs[0].Type != EventCapability {
		t.Fatalf("first event must be the capability handshake, got %+v", evs)
	}
	var cap Capability
	if err := json.Unmarshal(evs[0].Data, &cap); err != nil {
		t.Fatalf("decode capability: %v", err)
	}
	var credits []credit
	for _, ev := range evs[1:] {
		if ev.Type != EventFlowCredit {
			t.Fatalf("unexpected output event %q (%s)", ev.Type, ev.Data)
		}
		var fc FlowCredit
		if err := json.Unmarshal(ev.Data, &fc); err != nil {
			t.Fatalf("decode flow_credit: %v", err)
		}
		credits = append(credits, credit{fc.SessionId, fc.Frames})
	}
	return cap, credits
}

func equalCredits(t *testing.T, got, want []credit) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("credits = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("credits = %+v, want %+v", got, want)
		}
	}
}

func consumerCap() Capability {
	return Capability{StageType: "gate", StageName: "test-consumer", LifecycleModes: []string{"per_run"}}
}

// scripted is a consumer whose chunk outcomes and stop flow are set by the
// test, recording what the runtime handed it.
type scripted struct {
	BaseConsumer
	chunkOutcomes []Chunk
	stopFlow      Flow
	eofErr        error

	starts   []string
	chunks   []audio.AudioChunk
	payloads [][]byte
	stops    []string
	others   []string
	eofCalls int
}

func (s *scripted) OnAudioStart(ev audio.AudioStart, _ *AudioCtx) error {
	s.starts = append(s.starts, ev.SessionId)
	return nil
}

func (s *scripted) OnAudioChunk(ev audio.AudioChunk, payload []byte, _ *AudioCtx) (Chunk, error) {
	i := len(s.chunks)
	s.chunks = append(s.chunks, ev)
	s.payloads = append(s.payloads, payload)
	if i < len(s.chunkOutcomes) {
		return s.chunkOutcomes[i], nil
	}
	return ChunkCounted, nil
}

func (s *scripted) OnAudioStop(ev audio.AudioStop, _ *AudioCtx) (Flow, error) {
	s.stops = append(s.stops, ev.SessionId)
	return s.stopFlow, nil
}

func (s *scripted) OnOther(ev *Event, _ *AudioCtx) (Flow, error) {
	s.others = append(s.others, ev.Type)
	return FlowContinue, nil
}

func (s *scripted) OnEOF(*AudioCtx) error {
	s.eofCalls++
	return s.eofErr
}

// Per-session policy: the initial window is stamped with the session on
// audio_start, a Dropped chunk stays out of the cadence, and FlowStop from
// OnAudioStop ends the loop without reading further or calling OnEOF.
func TestServeAudioConsumer_PerSessionGrantCadenceAndStop(t *testing.T) {
	in := frames(t,
		startEv(t, "s1"),
		chunkEv(t, "s1", 10), // counted: 1 of 2
		chunkEv(t, "s1", 20), // dropped: not counted
		chunkEv(t, "s1", 30), // counted: 2 of 2 -> grant 4
		stopEv(t, "s1"),      // FlowStop
		chunkEv(t, "s1", 40), // after the stop: must never be read
	)
	var out bytes.Buffer
	h := &scripted{
		chunkOutcomes: []Chunk{ChunkCounted, ChunkDropped, ChunkCounted},
		stopFlow:      FlowStop,
	}
	policy := CreditPolicy{Initial: 8, Every: 2, Grant: 4, When: GrantOnSessionStart}

	if err := ServeAudioConsumerOn(in, &out, consumerCap(), policy, h); err != nil {
		t.Fatalf("ServeAudioConsumerOn: %v", err)
	}

	cap, credits := splitOutput(t, readAll(t, &out))
	if cap.StageName != "test-consumer" || cap.StageType != "gate" {
		t.Fatalf("capability did not round-trip: %+v", cap)
	}
	equalCredits(t, credits, []credit{{"s1", 8}, {"s1", 4}})

	if len(h.starts) != 1 || h.starts[0] != "s1" {
		t.Fatalf("OnAudioStart sessions = %v", h.starts)
	}
	if len(h.chunks) != 3 {
		t.Fatalf("OnAudioChunk saw %d chunks, want 3 (the one after FlowStop must not be read)", len(h.chunks))
	}
	if h.chunks[2].TimestampMs != 30 || !bytes.Equal(h.payloads[2], []byte{1, 2, 3, 4}) {
		t.Fatalf("chunk decode lost fields: %+v payload %v", h.chunks[2], h.payloads[2])
	}
	if len(h.stops) != 1 || h.stops[0] != "s1" {
		t.Fatalf("OnAudioStop sessions = %v", h.stops)
	}
	if h.eofCalls != 0 {
		t.Fatalf("OnEOF called %d times after FlowStop, want 0", h.eofCalls)
	}
}

// GrantOnStart opens the window right after the handshake with an empty
// session id — before any input exists — and a clean EOF hands off to OnEOF,
// whose error is the runtime's return value.
func TestServeAudioConsumer_GrantOnStartThenEOF(t *testing.T) {
	var out bytes.Buffer
	sentinel := errors.New("eof-sentinel")
	h := &scripted{eofErr: sentinel}
	policy := CreditPolicy{Initial: 5, When: GrantOnStart}

	err := ServeAudioConsumerOn(&bytes.Buffer{}, &out, consumerCap(), policy, h)
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want OnEOF's error", err)
	}
	if h.eofCalls != 1 {
		t.Fatalf("OnEOF called %d times, want 1", h.eofCalls)
	}
	_, credits := splitOutput(t, readAll(t, &out))
	equalCredits(t, credits, []credit{{"", 5}})
}

// The cadence counter survives session boundaries under GrantOnStart (only a
// GrantNow resets it), so a grant can fall in the second session and is
// stamped with THAT session. Every=0 in the policy means no cadence grants.
func TestServeAudioConsumer_CadenceSurvivesSessionsAndEveryZeroNeverGrants(t *testing.T) {
	t.Run("counter survives sessions", func(t *testing.T) {
		in := frames(t,
			startEv(t, "a"), chunkEv(t, "a", 1), stopEv(t, "a"),
			startEv(t, "b"), chunkEv(t, "b", 2), stopEv(t, "b"),
		)
		var out bytes.Buffer
		policy := CreditPolicy{Initial: 3, Every: 2, Grant: 6, When: GrantOnStart}
		if err := ServeAudioConsumerOn(in, &out, consumerCap(), policy, &scripted{}); err != nil {
			t.Fatal(err)
		}
		_, credits := splitOutput(t, readAll(t, &out))
		equalCredits(t, credits, []credit{{"", 3}, {"b", 6}})
	})

	t.Run("every zero", func(t *testing.T) {
		in := frames(t, startEv(t, "s"), chunkEv(t, "s", 1), chunkEv(t, "s", 2), chunkEv(t, "s", 3))
		var out bytes.Buffer
		policy := CreditPolicy{Initial: 2, Every: 0, Grant: 9, When: GrantOnSessionStart}
		if err := ServeAudioConsumerOn(in, &out, consumerCap(), policy, &scripted{}); err != nil {
			t.Fatal(err)
		}
		_, credits := splitOutput(t, readAll(t, &out))
		equalCredits(t, credits, []credit{{"s", 2}})
	})

	t.Run("manual grants nothing", func(t *testing.T) {
		in := frames(t, startEv(t, "s"), chunkEv(t, "s", 1))
		var out bytes.Buffer
		if err := ServeAudioConsumerOn(in, &out, consumerCap(), NoCredit, &scripted{}); err != nil {
			t.Fatal(err)
		}
		_, credits := splitOutput(t, readAll(t, &out))
		equalCredits(t, credits, nil)
	})
}

// Unknown event types go to OnOther and the loop continues — wire leniency is
// contract.
func TestServeAudioConsumer_UnknownEventsReachOnOther(t *testing.T) {
	in := frames(t,
		typed(t, "ext.test.thing", map[string]int{"n": 1}, nil),
		startEv(t, "s"),
	)
	h := &scripted{}
	if err := ServeAudioConsumerOn(in, io.Discard, consumerCap(), NoCredit, h); err != nil {
		t.Fatal(err)
	}
	if len(h.others) != 1 || h.others[0] != "ext.test.thing" {
		t.Fatalf("OnOther saw %v", h.others)
	}
	if len(h.starts) != 1 {
		t.Fatalf("the loop stopped after the unknown event: starts = %v", h.starts)
	}
}

// Undecodable known events are fatal, and the error names what failed.
func TestServeAudioConsumer_BadEventDataIsAnError(t *testing.T) {
	for _, tc := range []struct {
		eventType string
		want      string
	}{
		{audio.EventAudioChunk, "decode audio_chunk"},
		{audio.EventAudioStart, "decode audio_start"},
		{audio.EventAudioStop, "decode audio_stop"},
	} {
		t.Run(tc.eventType, func(t *testing.T) {
			in := frames(t, &Event{Type: tc.eventType, Data: json.RawMessage(`{"session_id":42}`)})
			h := &scripted{}
			err := ServeAudioConsumerOn(in, io.Discard, consumerCap(), NoCredit, h)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one containing %q", err, tc.want)
			}
			if h.eofCalls != 0 {
				t.Fatal("OnEOF must not run after a decode error")
			}
		})
	}
}

// A malformed frame surfaces as the reader's error, not as EOF.
func TestServeAudioConsumer_MalformedFrameIsAnError(t *testing.T) {
	h := &scripted{}
	err := ServeAudioConsumerOn(strings.NewReader("not json\n"), io.Discard, consumerCap(), NoCredit, h)
	if err == nil || !strings.Contains(err.Error(), "bad header") {
		t.Fatalf("err = %v, want a bad-header error", err)
	}
	if h.eofCalls != 0 {
		t.Fatal("a malformed frame must not be treated as a clean EOF")
	}
}

func TestCreditGranter_Cadence(t *testing.T) {
	var out bytes.Buffer
	w := NewWriter(&out)
	g := NewCreditGranter(3, 7)

	var grantedAt []int
	count := func(n int, base int) {
		for i := 1; i <= n; i++ {
			before := out.Len()
			if err := g.OnChunk(w, "s"); err != nil {
				t.Fatal(err)
			}
			if out.Len() != before {
				grantedAt = append(grantedAt, base+i)
			}
		}
	}

	count(7, 0) // grants after chunks 3 and 6
	if want := []int{3, 6}; !equalInts(grantedAt, want) {
		t.Fatalf("granted after chunks %v, want %v", grantedAt, want)
	}

	// Chunk 7 left the counter at 1; GrantNow resets it, so the next cadence
	// grant needs three MORE chunks, not two.
	grantedAt = nil
	if err := g.GrantNow(w, "s", 11); err != nil {
		t.Fatal(err)
	}
	count(3, 0)
	if want := []int{3}; !equalInts(grantedAt, want) {
		t.Fatalf("after GrantNow, granted after chunks %v, want %v", grantedAt, want)
	}

	var got []credit
	for _, ev := range readAll(t, &out) {
		var fc FlowCredit
		if ev.Type != EventFlowCredit {
			t.Fatalf("unexpected event %q", ev.Type)
		}
		if err := json.Unmarshal(ev.Data, &fc); err != nil {
			t.Fatal(err)
		}
		got = append(got, credit{fc.SessionId, fc.Frames})
	}
	equalCredits(t, got, []credit{{"s", 7}, {"s", 7}, {"s", 11}, {"s", 7}})
}

func TestCreditGranter_EveryZeroNeverGrants(t *testing.T) {
	var out bytes.Buffer
	w := NewWriter(&out)
	g := NewCreditGranter(0, 5)
	for i := 0; i < 50; i++ {
		if err := g.OnChunk(w, "s"); err != nil {
			t.Fatal(err)
		}
	}
	if out.Len() != 0 {
		t.Fatalf("every=0 granted: %q", out.String())
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ServeSourceOn writes the handshake before the body runs, the body's emits
// follow it in order, RequestStop flips Stopped/Done, and the body's error is
// the return value.
func TestServeSource_HandshakeEmitsAndStop(t *testing.T) {
	var out bytes.Buffer
	bodyErr := errors.New("body-done")
	err := ServeSourceOn(&bytes.Buffer{}, &out, srcCapability(), SourceOptions{},
		func(ctx *SourceCtx) error {
			// The handshake is already on the wire when the body starts.
			if !strings.Contains(out.String(), `"type":"capability"`) {
				t.Error("body ran before the capability handshake was written")
			}
			if ctx.Stopped() {
				t.Error("Stopped before any stop was requested")
			}
			if err := ctx.Emit("ext.test.reading", map[string]int{"v": 1}); err != nil {
				return err
			}
			if err := ctx.EmitRaw("ext.test.blob", map[string]int{"v": 2}, []byte("xyz")); err != nil {
				return err
			}
			ctx.RequestStop()
			select {
			case <-ctx.Done():
			default:
				t.Error("Done not closed after RequestStop")
			}
			if !ctx.Stopped() {
				t.Error("Stopped false after RequestStop")
			}
			if ctx.StopRequest() != nil {
				t.Error("an internal stop must carry no StopRequest")
			}
			return bodyErr
		})
	if !errors.Is(err, bodyErr) {
		t.Fatalf("err = %v, want the body's error", err)
	}

	evs := readAll(t, &out)
	if len(evs) != 3 {
		t.Fatalf("got %d events, want capability + 2 emits", len(evs))
	}
	if evs[0].Type != EventCapability || evs[1].Type != "ext.test.reading" || evs[2].Type != "ext.test.blob" {
		t.Fatalf("event order = %s, %s, %s", evs[0].Type, evs[1].Type, evs[2].Type)
	}
	if string(evs[1].Data) != `{"v":1}` || evs[1].Payload != nil {
		t.Fatalf("Emit wrote data %s payload %v", evs[1].Data, evs[1].Payload)
	}
	if string(evs[2].Payload) != "xyz" {
		t.Fatalf("EmitRaw payload = %q", evs[2].Payload)
	}
}
