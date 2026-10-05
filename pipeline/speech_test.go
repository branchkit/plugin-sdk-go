package pipeline

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branchkit/plugin-sdk-go/pipeline/audio"
)

// The speech engine runtime: one speak request in, one audio session out,
// closed by exactly one audio_stop whatever happened to it.

// wordEngine "speaks" one 4-byte chunk per word, pausing between chunks the
// way synthesis would. A word "fail" fails the utterance after its chunk.
type wordEngine struct {
	mu     sync.Mutex
	spoken []string
}

func (e *wordEngine) Speak(req audio.Speak, ctx *SpeakCtx) error {
	e.mu.Lock()
	e.spoken = append(e.spoken, req.SessionId)
	e.mu.Unlock()
	if err := ctx.Start(audio.AudioFormat{Channels: 1, Rate: 16000, Width: 2}); err != nil {
		return err
	}
	for _, word := range strings.Fields(req.Text) {
		flow, err := ctx.Audio([]byte{0, 0, 0, 0})
		if err != nil {
			return err
		}
		if flow == FlowStop {
			return nil
		}
		if word == "fail" {
			return errors.New("engine broke")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

func speakEv(t *testing.T, session, text string) *Event {
	return typed(t, audio.EventSpeak, audio.Speak{SessionId: session, Text: text}, nil)
}

func ttsCap() Capability {
	return Capability{StageType: "tts", StageName: "test", LifecycleModes: []string{"persistent"}}
}

// runEngine serves e over in-memory transports with inbound written up front;
// the input stays open for holdOpen (so queued utterances get spoken) and then
// reaches EOF.
func runEngine(t *testing.T, e SpeechEngine, holdOpen time.Duration, inbound ...*Event) []*Event {
	t.Helper()
	in := frames(t, inbound...)
	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write(in.Bytes())
		time.Sleep(holdOpen)
		_ = pw.Close()
	}()
	// Only the serving goroutine writes, so a plain buffer is safe.
	var out bytes.Buffer
	if err := ServeSpeechEngineOn(pr, &out, ttsCap(), e); err != nil {
		t.Fatal(err)
	}
	return readAll(t, &out)
}

func forSession(evs []*Event, session string) []string {
	var tags []string
	for _, ev := range evs {
		if strings.Contains(string(ev.Data), `"session_id":"`+session+`"`) {
			tags = append(tags, ev.Type)
		}
	}
	return tags
}

func TestSpeechEngineUtteranceIsStartChunksStopAndNeverCredit(t *testing.T) {
	out := runEngine(t, &wordEngine{}, 200*time.Millisecond, speakEv(t, "u1", "snap left now"))
	if out[0].Type != EventCapability {
		t.Fatalf("first event %q, want capability", out[0].Type)
	}
	got := strings.Join(forSession(out, "u1"), ",")
	want := "audio_start,audio_chunk,audio_chunk,audio_chunk,audio_stop"
	if got != want {
		t.Fatalf("u1 events = %s, want %s", got, want)
	}
	for _, ev := range out {
		if ev.Type == EventFlowCredit {
			t.Fatal("a producer implements no credit")
		}
	}
}

func TestSpeechEngineSpeaksInArrivalOrder(t *testing.T) {
	e := &wordEngine{}
	runEngine(t, e, 200*time.Millisecond, speakEv(t, "a", "one"), speakEv(t, "b", "two"))
	if strings.Join(e.spoken, ",") != "a,b" {
		t.Fatalf("spoken %v, want [a b]", e.spoken)
	}
}

func TestSpeechEngineCancelStopsTheUtteranceInProgress(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- ServeSpeechEngineOn(inR, outW, ttsCap(), &wordEngine{})
		_ = outW.Close()
	}()
	w := NewWriter(inW)
	r := NewReader(outR)
	// io.Pipe has no buffer: take the handshake before writing, or both
	// sides block on a write the other is not reading.
	if ev, err := r.ReadEvent(); err != nil || ev.Type != EventCapability {
		t.Fatalf("handshake: %v %v", ev, err)
	}
	if err := w.WriteEvent(speakEv(t, "long", strings.Repeat("word ", 200))); err != nil {
		t.Fatal(err)
	}
	for {
		ev, err := r.ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		if ev.Type == audio.EventAudioChunk {
			break
		}
	}
	if err := w.WriteEvent(stopEv(t, "long")); err != nil {
		t.Fatal(err)
	}
	var after []string
	for {
		ev, err := r.ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		after = append(after, ev.Type)
		if ev.Type == audio.EventAudioStop {
			break
		}
	}
	if len(after) >= 10 {
		t.Fatalf("engine kept speaking after the cancel: %v", after)
	}
	_ = inW.Close()
	rest, _ := io.ReadAll(outR)
	if strings.Contains(string(rest), `"long"`) {
		t.Fatal("an event for the session after its close")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSpeechEngineQueuedUtteranceCancelledBeforeItBeginsIsClosedUnspoken(t *testing.T) {
	e := &wordEngine{}
	out := runEngine(t, e, 300*time.Millisecond,
		speakEv(t, "a", "one two three four"),
		speakEv(t, "b", "never"),
		stopEv(t, "b"),
	)
	if strings.Join(e.spoken, ",") != "a" {
		t.Fatalf("spoken %v, want only a", e.spoken)
	}
	if got := strings.Join(forSession(out, "b"), ","); got != "audio_stop" {
		t.Fatalf("b events = %s, want audio_stop alone", got)
	}
}

func TestSpeechEngineFailedUtteranceReportsClosesAndTheNextPlays(t *testing.T) {
	out := runEngine(t, &wordEngine{}, 200*time.Millisecond,
		speakEv(t, "bad", "fail here"),
		speakEv(t, "good", "fine"),
	)
	if got := strings.Join(forSession(out, "bad"), ","); got != "audio_start,audio_chunk,error,audio_stop" {
		t.Fatalf("bad events = %s", got)
	}
	if got := strings.Join(forSession(out, "good"), ","); got != "audio_start,audio_chunk,audio_stop" {
		t.Fatalf("good events = %s", got)
	}
}

func TestSpeechEngineIgnoresUnknownAndMalformedInbound(t *testing.T) {
	e := &wordEngine{}
	out := runEngine(t, e, 200*time.Millisecond,
		typed(t, "ext.acme.thing", map[string]any{}, nil),
		typed(t, audio.EventSpeak, map[string]any{"no": "text"}, nil),
		speakEv(t, "ok", "hello"),
	)
	if strings.Join(e.spoken, ",") != "ok" {
		t.Fatalf("spoken %v, want [ok]", e.spoken)
	}
	if got := strings.Join(forSession(out, "ok"), ","); got != "audio_start,audio_chunk,audio_stop" {
		t.Fatalf("ok events = %s", got)
	}
}

func TestSharedClockMovesForward(t *testing.T) {
	a := SharedClockMs()
	time.Sleep(5 * time.Millisecond)
	if b := SharedClockMs(); b < a+4 {
		t.Fatalf("clock moved %d ms over a 5 ms sleep", b-a)
	}
}
