package pipeline

// The speech engine runtime — the third loop shape, request-driven: each
// speak request becomes an audio session the stage produces, streaming, and
// stops the moment the platform cancels it. Text-to-speech engines: the
// pipeline run the other way, from words to the speakers.
//
// A speech engine produces audio, so it implements no flow credit (see the
// package docs on stage.go): the platform holds the window.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/branchkit/plugin-sdk-go/pipeline/audio"
)

// SpeakCtx is what [SpeechEngine.Speak] is handed: where the utterance's
// audio goes, and whether it has been cancelled.
type SpeakCtx struct {
	mu        *sync.Mutex
	w         *Writer
	sessionID string
	ctx       context.Context
	started   bool
}

// SessionID is the utterance this context belongs to.
func (c *SpeakCtx) SessionID() string { return c.sessionID }

// Context is cancelled when the platform cancels the utterance (or goes
// away). Hand it to anything that takes one, or select on Done.
func (c *SpeakCtx) Context() context.Context { return c.ctx }

// Done is closed when the utterance is cancelled.
func (c *SpeakCtx) Done() <-chan struct{} { return c.ctx.Done() }

// Cancelled reports whether the platform has cancelled the utterance. Stop
// synthesizing when it has: nothing more of it will be sent.
func (c *SpeakCtx) Cancelled() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// Start opens the utterance's audio in format. Call it once, before the first
// [SpeakCtx.Audio]. Engines usually know their format only once the model is
// loaded, which is why it is given here and not in the capability.
func (c *SpeakCtx) Start(format audio.AudioFormat) error {
	if c.started {
		return errors.New("speech engine: Start called twice for one utterance")
	}
	c.started = true
	if c.Cancelled() {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeTyped(c.w, audio.EventAudioStart, audio.AudioStart{SessionId: c.sessionID, Format: format}, nil)
}

// Audio sends one chunk of audio, in the format given to Start, as soon as it
// is synthesized. Send it in pieces as the engine makes them, never the whole
// utterance at the end, so the first words play while the rest are made.
//
// Returns [FlowStop] once the utterance is cancelled, without sending
// anything: stop synthesizing and return. The runtime closes the utterance
// either way.
func (c *SpeakCtx) Audio(pcm []byte) (Flow, error) {
	if !c.started {
		return FlowStop, errors.New("speech engine: Audio before Start")
	}
	if c.Cancelled() {
		return FlowStop, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	chunk := audio.AudioChunk{SessionId: c.sessionID, TimestampMs: SharedClockMs()}
	if err := writeTyped(c.w, audio.EventAudioChunk, chunk, pcm); err != nil {
		return FlowStop, err
	}
	return FlowContinue, nil
}

// SpeechEngine is a text-to-speech stage (stage_type "tts").
//
// The engine implements one thing, how to say one request; the runtime owns
// the rest of the contract — the handshake, the order requests are spoken in,
// cancellation, and closing every utterance with exactly one audio_stop,
// including one that failed or was cancelled before it began.
type SpeechEngine interface {
	// Speak says req: call ctx.Start once with the audio format, then
	// ctx.Audio for each piece as it is synthesized, and return when the
	// utterance is done or Audio says FlowStop.
	//
	// An error fails this utterance only: the runtime sends an error event
	// for it, closes it, and goes on to the next. A failure that makes the
	// engine unusable belongs before ServeSpeechEngine (a model that does not
	// load), where Run turns it into exit 1.
	Speak(req audio.Speak, ctx *SpeakCtx) error
}

// ServeSpeechEngine serves a speech engine on stdin/stdout.
func ServeSpeechEngine(cap Capability, engine SpeechEngine) error {
	return ServeSpeechEngineOn(os.Stdin, os.Stdout, cap, engine)
}

type speakInbound struct {
	speak  *audio.Speak
	cancel string
}

// ServeSpeechEngineOn is [ServeSpeechEngine] over explicit transports, for
// tests.
func ServeSpeechEngineOn(r io.Reader, w io.Writer, cap Capability, engine SpeechEngine) error {
	mu := &sync.Mutex{}
	writer := NewWriter(w)
	if err := writeTyped(writer, EventCapability, cap, nil); err != nil {
		return err
	}

	// The utterance being spoken, so a cancel for it reaches the engine while
	// Speak is still running rather than after it returns.
	var curMu sync.Mutex
	var curID string
	var curCancel context.CancelFunc

	// The inbox is unbounded on purpose: the reader must never block on a
	// busy engine, or a cancel for the utterance being spoken would wait
	// behind the requests queued after it.
	var inMu sync.Mutex
	var pending []speakInbound
	closed := false
	wake := make(chan struct{}, 1)
	post := func(m speakInbound) {
		inMu.Lock()
		pending = append(pending, m)
		inMu.Unlock()
		select {
		case wake <- struct{}{}:
		default:
		}
	}
	go func() {
		reader := NewReader(r)
		for {
			ev, err := reader.ReadEvent()
			if err != nil {
				// EOF or a broken pipe: the platform is gone, and whatever is
				// being said will not be heard.
				curMu.Lock()
				if curCancel != nil {
					curCancel()
				}
				curMu.Unlock()
				inMu.Lock()
				closed = true
				inMu.Unlock()
				select {
				case wake <- struct{}{}:
				default:
				}
				return
			}
			switch ev.Type {
			case audio.EventSpeak:
				var req audio.Speak
				if err := json.Unmarshal(ev.Data, &req); err != nil || req.SessionId == "" {
					LogWarn("speak: undecodable request")
					continue
				}
				post(speakInbound{speak: &req})
			case audio.EventAudioStop:
				var stop audio.AudioStop
				if err := json.Unmarshal(ev.Data, &stop); err != nil {
					continue
				}
				curMu.Lock()
				if curCancel != nil && curID == stop.SessionId {
					curCancel()
				}
				curMu.Unlock()
				post(speakInbound{cancel: stop.SessionId})
			}
		}
	}()

	// take hands over everything that has arrived, and whether the platform
	// is gone.
	take := func() ([]speakInbound, bool) {
		inMu.Lock()
		defer inMu.Unlock()
		got := pending
		pending = nil
		return got, closed
	}

	closeUtterance := func(id string) error {
		mu.Lock()
		defer mu.Unlock()
		return writeTyped(writer, audio.EventAudioStop, audio.AudioStop{SessionId: id}, nil)
	}

	var queue []audio.Speak
	handle := func(m speakInbound) error {
		if m.speak != nil {
			queue = append(queue, *m.speak)
			return nil
		}
		// Queued and not yet begun: close it now. A cancel for the utterance
		// being spoken already went through its context, and one for an id no
		// longer known is moot.
		for i, q := range queue {
			if q.SessionId == m.cancel {
				queue = append(queue[:i], queue[i+1:]...)
				return closeUtterance(m.cancel)
			}
		}
		return nil
	}

	for {
		// Take everything that has arrived before choosing what to say next,
		// so a cancel already sent for a queued utterance is honored before it
		// starts.
		msgs, gone := take()
		if gone {
			return nil
		}
		for _, m := range msgs {
			if err := handle(m); err != nil {
				return err
			}
		}
		if len(queue) == 0 {
			<-wake
			continue
		}
		req := queue[0]
		queue = queue[1:]

		uctx, cancel := context.WithCancel(context.Background())
		curMu.Lock()
		curID, curCancel = req.SessionId, cancel
		curMu.Unlock()

		sc := &SpeakCtx{mu: mu, w: writer, sessionID: req.SessionId, ctx: uctx}
		speakErr := engine.Speak(req, sc)

		curMu.Lock()
		curID, curCancel = "", nil
		curMu.Unlock()
		cancel()

		if speakErr != nil {
			mu.Lock()
			sid := req.SessionId
			err := writeTyped(writer, EventError, ErrorEvent{
				SessionId: &sid,
				Code:      "speak_failed",
				Message:   speakErr.Error(),
				Fatal:     false,
			}, nil)
			mu.Unlock()
			if err != nil {
				return fmt.Errorf("stage: write error: %w", err)
			}
		}
		if err := closeUtterance(req.SessionId); err != nil {
			return err
		}
	}
}
