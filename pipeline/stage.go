package pipeline

// The stage runtime — the layer between wire framing and a working stage.
//
// [Reader]/[Writer] get bytes on and off the pipe. This owns the obligations
// above them that every stage otherwise hand-rolls, and that fail SILENTLY
// when hand-rolled wrong:
//
//   - the capability handshake goes out first and unprompted,
//   - receiver-side flow credit follows a declared policy rather than a
//     counter each stage maintains itself,
//   - unknown events are tolerated (wire leniency is contract, not courtesy),
//   - a fatal error becomes one BKLOG1 line and exit 1, the same way in every
//     stage.
//
// # Which entry point
//
// Two, because there are two loop shapes:
//
//   - [ServeAudioConsumer] — read-driven. The stage's work is a reaction to an
//     inbound audio session. VAD gates, STT engines, command recognizers.
//   - [ServeSource] — notifier-driven. The stage produces spontaneously from a
//     device, OS notification, or timer, and may never read stdin at all.
//     Power/display/location monitors, microphones.
//
// An audio source is the second shape plus [SourceOptions.ListenForStop].
//
// # Why one is media-neutral and the other is not
//
// [ServeSource] assumes nothing about what you emit. [ServeAudioConsumer] is
// audio-bound because audio is the only STREAM the wire has — audio_start /
// audio_chunk / audio_stop are the streaming events, everything else is
// discrete, and flow credit counts audio frames. A runtime cannot be more
// general than the protocol it speaks.
//
// # Flow credit: which side are you on
//
//   - Consuming audio → you must grant credit. [CreditPolicy] drives it.
//   - Producing audio → you must not implement credit at all. The platform
//     holds the sender-side window; a producer that outruns it blocks on the
//     pipe. There is deliberately no sender-side helper here.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Run executes a stage body as main, mapping a fatal error to one structured
// log line and exit code 1:
//
//	func main() { pipeline.Run(run) }
func Run(body func() error) {
	if err := body(); err != nil {
		LogError("fatal: %v", err)
		os.Exit(1)
	}
}

// InitialGrant says when the runtime emits the initial credit window.
//
// The NUMBERS are receiver-chosen buffering policy and stay at your call site;
// only the mechanism is shared. This captures the one structural difference
// between stages: whether the window opens before any session exists.
type InitialGrant int

const (
	// GrantOnStart emits the initial window right after the handshake with an
	// empty session id — the stage buffers freely and wants upstream moving
	// before any session exists.
	GrantOnStart InitialGrant = iota
	// GrantOnSessionStart emits it on every audio_start, stamped with that
	// session (which also resets the cadence counter). Shallow-queue stages.
	GrantOnSessionStart
	// GrantManual emits nothing automatically — for a window that depends on
	// runtime state, or a stage that consumes no audio and must never grant.
	GrantManual
)

// CreditPolicy is the receiver-side credit configuration.
type CreditPolicy struct {
	Initial uint32       // frames in the unconditional initial window
	Every   uint32       // grant again after every N processed chunks
	Grant   uint32       // frames per cadence grant
	When    InitialGrant // when the initial window is emitted
}

// NoCredit is the policy for a stage that consumes no audio and must never
// grant.
var NoCredit = CreditPolicy{When: GrantManual}

// Chunk says whether a delivered audio_chunk counts toward the credit cadence.
type Chunk int

const (
	// ChunkCounted means processed — count it. The normal answer.
	ChunkCounted Chunk = iota
	// ChunkDropped means discarded without processing (a stale session id, an
	// unsupported format), so it does not count.
	//
	// Note the open question this preserves rather than settles: a dropped
	// chunk still spent a frame of the sender's window, so never counting it
	// shrinks that window for the rest of the run.
	ChunkDropped
)

// Flow says whether the consumer loop continues after a callback.
type Flow int

const (
	// FlowContinue keeps reading. The default.
	FlowContinue Flow = iota
	// FlowStop ends the loop and returns — a per_run stage finishing its
	// session.
	FlowStop
)

// writeTyped marshals data and writes it as one framed event.
func writeTyped(w *Writer, eventType string, data any, payload []byte) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("stage: marshal %s: %w", eventType, err)
	}
	return w.WriteEvent(&Event{Type: eventType, Data: raw, Payload: payload})
}

// AudioCtx is what a consumer callback is handed: the outbound writer, plus
// the credit granter wired to this stage's policy.
type AudioCtx struct {
	mu     sync.Mutex
	w      *Writer
	credit *CreditGranter
	policy CreditPolicy
}

// Emit marshals data and writes it as one framed event.
func (c *AudioCtx) Emit(eventType string, data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeTyped(c.w, eventType, data, nil)
}

// EmitRaw marshals data and writes it with a binary payload.
func (c *AudioCtx) EmitRaw(eventType string, data any, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeTyped(c.w, eventType, data, payload)
}

// GrantNow emits credit unconditionally and resets the cadence counter, for
// windows the policy cannot express — a re-grant at an utterance boundary.
func (c *AudioCtx) GrantNow(sessionID string, frames uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.credit.GrantNow(c.w, sessionID, frames)
}

// AudioConsumer is a read-driven stage.
//
// Embed [BaseConsumer] and override only the events you care about; it
// supplies conformant no-ops for the rest. Implementing the interface by hand
// is fine too, but then a method you forget is a compile error rather than a
// silent default — which is the trade this SDK generally prefers.
type AudioConsumer interface {
	// OnAudioStart is called when a session begins upstream.
	OnAudioStart(ev AudioStart, ctx *AudioCtx) error
	// OnAudioChunk receives one frame of audio. Return ChunkDropped to keep it
	// out of the credit cadence.
	OnAudioChunk(ev AudioChunk, payload []byte, ctx *AudioCtx) (Chunk, error)
	// OnAudioStop is called when the session ends. A per_run stage emits its
	// final result here and returns FlowStop.
	OnAudioStop(ev AudioStop, ctx *AudioCtx) (Flow, error)
	// OnOther receives any event the runtime did not decode, including unknown
	// types. Ignoring them is the default because wire leniency is contract.
	OnOther(ev *Event, ctx *AudioCtx) (Flow, error)
	// OnEOF is called on clean EOF — upstream closed.
	OnEOF(ctx *AudioCtx) error
}

// BaseConsumer supplies conformant defaults for every AudioConsumer method.
// Embed it so a stage implements only what it cares about.
type BaseConsumer struct{}

func (BaseConsumer) OnAudioStart(AudioStart, *AudioCtx) error { return nil }
func (BaseConsumer) OnAudioChunk(AudioChunk, []byte, *AudioCtx) (Chunk, error) {
	return ChunkCounted, nil
}
func (BaseConsumer) OnAudioStop(AudioStop, *AudioCtx) (Flow, error) { return FlowContinue, nil }
func (BaseConsumer) OnOther(*Event, *AudioCtx) (Flow, error)        { return FlowContinue, nil }
func (BaseConsumer) OnEOF(*AudioCtx) error                          { return nil }

// ServeAudioConsumer serves a read-driven stage on stdin/stdout.
func ServeAudioConsumer(cap Capability, policy CreditPolicy, handler AudioConsumer) error {
	return ServeAudioConsumerOn(os.Stdin, os.Stdout, cap, policy, handler)
}

// ServeAudioConsumerOn is [ServeAudioConsumer] over explicit transports. The
// stdio wrapper is what stages use; this exists so the runtime itself is
// testable over a pipe.
func ServeAudioConsumerOn(
	r io.Reader,
	w io.Writer,
	cap Capability,
	policy CreditPolicy,
	handler AudioConsumer,
) error {
	every := policy.Every
	if every == 0 {
		every = 1
	}
	reader := NewReader(r)
	ctx := &AudioCtx{
		w:      NewWriter(w),
		credit: NewCreditGranter(every, policy.Grant),
		policy: policy,
	}

	if err := ctx.Emit(EventCapability, cap); err != nil {
		return err
	}
	if policy.When == GrantOnStart && policy.Initial > 0 {
		if err := ctx.GrantNow("", policy.Initial); err != nil {
			return err
		}
	}

	for {
		ev, err := reader.ReadEvent()
		if errors.Is(err, io.EOF) {
			return handler.OnEOF(ctx)
		}
		if err != nil {
			return err
		}

		switch ev.Type {
		case EventAudioStart:
			var start AudioStart
			if err := json.Unmarshal(ev.Data, &start); err != nil {
				return fmt.Errorf("stage: decode audio_start: %w", err)
			}
			if policy.When == GrantOnSessionStart && policy.Initial > 0 {
				if err := ctx.GrantNow(start.SessionId, policy.Initial); err != nil {
					return err
				}
			}
			if err := handler.OnAudioStart(start, ctx); err != nil {
				return err
			}

		case EventAudioChunk:
			var chunk AudioChunk
			if err := json.Unmarshal(ev.Data, &chunk); err != nil {
				return fmt.Errorf("stage: decode audio_chunk: %w", err)
			}
			outcome, err := handler.OnAudioChunk(chunk, ev.Payload, ctx)
			if err != nil {
				return err
			}
			if outcome == ChunkCounted && policy.Every > 0 {
				ctx.mu.Lock()
				err = ctx.credit.OnChunk(ctx.w, chunk.SessionId)
				ctx.mu.Unlock()
				if err != nil {
					return err
				}
			}

		case EventAudioStop:
			var stop AudioStop
			if err := json.Unmarshal(ev.Data, &stop); err != nil {
				return fmt.Errorf("stage: decode audio_stop: %w", err)
			}
			flow, err := handler.OnAudioStop(stop, ctx)
			if err != nil {
				return err
			}
			if flow == FlowStop {
				return nil
			}

		default:
			flow, err := handler.OnOther(ev, ctx)
			if err != nil {
				return err
			}
			if flow == FlowStop {
				return nil
			}
		}
	}
}

// SourceOptions configures [ServeSource].
type SourceOptions struct {
	// ListenForStop watches stdin for the platform's audio_stop stop request
	// and stops when it arrives (or on EOF, which means the platform is gone).
	//
	// This is what makes an audio source out of an event source: the runner
	// ends a session by writing audio_stop to the source's stdin, and that
	// stop may carry a CutoffMs the source must forward verbatim on its own
	// downstream audio_stop. Read it back with [SourceCtx.StopRequest].
	//
	// Off by default: an event source that never opens stdin is the common
	// case, and turning this on for one would be a behavior change.
	ListenForStop bool
}

// SourceCtx is a running source stage's handle: the outbound writer plus the
// stop signal.
type SourceCtx struct {
	mu          sync.Mutex
	w           *Writer
	ctx         context.Context
	stop        context.CancelFunc
	stopRequest *AudioStop
}

// Emit marshals data and writes it as one framed event.
func (c *SourceCtx) Emit(eventType string, data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeTyped(c.w, eventType, data, nil)
}

// EmitRaw marshals data and writes it with a binary payload.
func (c *SourceCtx) EmitRaw(eventType string, data any, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return writeTyped(c.w, eventType, data, payload)
}

// Done is closed when a stop is requested. Select on it against your own
// event source.
func (c *SourceCtx) Done() <-chan struct{} { return c.ctx.Done() }

// Context is the stop signal as a context, for handing to anything that takes
// one.
func (c *SourceCtx) Context() context.Context { return c.ctx }

// Stopped reports whether a stop has been requested — for a producer loop
// that polls rather than selects.
func (c *SourceCtx) Stopped() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// RequestStop requests a stop from inside the stage.
func (c *SourceCtx) RequestStop() { c.stop() }

// StopRequest returns the audio_stop that requested this stop, when the
// platform sent one and [SourceOptions.ListenForStop] is on. Nil if the stop
// came from a signal, from stdin EOF, or from [SourceCtx.RequestStop].
//
// An audio source forwards this event's CutoffMs verbatim on its own
// downstream audio_stop.
func (c *SourceCtx) StopRequest() *AudioStop {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopRequest
}

// ServeSource serves a notifier-driven stage on stdout.
//
// It emits the capability handshake, installs a SIGTERM/SIGINT stop watcher
// (and the stop listener, per opts), then hands control to body. The stage
// owns its own loop — that is the point of this shape.
func ServeSource(cap Capability, opts SourceOptions, body func(*SourceCtx) error) error {
	return ServeSourceOn(os.Stdin, os.Stdout, cap, opts, body)
}

// ServeSourceOn is [ServeSource] over explicit transports, for tests.
func ServeSourceOn(
	r io.Reader,
	w io.Writer,
	cap Capability,
	opts SourceOptions,
	body func(*SourceCtx) error,
) error {
	base, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopSignals()
	ctx, cancel := context.WithCancel(base)
	defer cancel()

	sc := &SourceCtx{w: NewWriter(w), ctx: ctx, stop: cancel}
	if err := sc.Emit(EventCapability, cap); err != nil {
		return err
	}
	if opts.ListenForStop {
		go watchForStop(r, sc)
	}
	return body(sc)
}

// watchForStop reads stdin until the platform's audio_stop arrives, or until
// EOF/error — which mean the platform is gone.
//
// Every other inbound event is ignored rather than fatal: a source stage must
// tolerate inbound bytes without dying, which the conformance source suite
// tests directly.
func watchForStop(r io.Reader, sc *SourceCtx) {
	defer sc.stop()
	reader := NewReader(r)
	for {
		ev, err := reader.ReadEvent()
		if err != nil {
			return
		}
		if ev.Type != EventAudioStop {
			continue
		}
		var stop AudioStop
		if err := json.Unmarshal(ev.Data, &stop); err == nil {
			sc.mu.Lock()
			sc.stopRequest = &stop
			sc.mu.Unlock()
		}
		return
	}
}
