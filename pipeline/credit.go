package pipeline

// Receiver-side flow-credit granting.
//
// The pipeline contract mandates window-based backpressure: the receiver
// advertises credit, the sender decrements per audio_chunk and blocks at
// zero. HOW MUCH to advertise — the initial window, the grant cadence, the
// grant size — is receiver-chosen buffering policy and deliberately NOT part
// of the contract (a gate buffering freely wants a wide window; an STT engine
// wants shallow queues). This owns only the mechanism: the chunks-since-last-
// grant counter and the flow_credit emission.
//
// There is deliberately no sender-side counterpart. The platform sits between
// every pair of stages and holds that window itself; a stage that PRODUCES
// audio implements no credit at all and simply blocks on the pipe.

// CreditGranter counts processed chunks and grants credit on a cadence.
type CreditGranter struct {
	every uint32
	grant uint32
	since uint32
}

// NewCreditGranter grants `grant` frames after every `every` chunks.
func NewCreditGranter(every, grant uint32) *CreditGranter {
	return &CreditGranter{every: every, grant: grant}
}

// GrantNow emits `frames` of credit unconditionally and resets the cadence
// counter. Use it for the initial window, or a re-grant at an utterance
// boundary.
func (c *CreditGranter) GrantNow(w *Writer, sessionID string, frames uint32) error {
	c.since = 0
	return emitCredit(w, sessionID, frames)
}

// OnChunk counts one processed chunk and grants after every `every` of them.
//
// The counter deliberately survives session boundaries: a stage wanting a
// per-session reset gets it from GrantNow's initial grant, and one that does
// not simply never resets.
func (c *CreditGranter) OnChunk(w *Writer, sessionID string) error {
	if c.every == 0 {
		return nil
	}
	c.since++
	if c.since >= c.every {
		c.since = 0
		return emitCredit(w, sessionID, c.grant)
	}
	return nil
}

func emitCredit(w *Writer, sessionID string, frames uint32) error {
	return writeTyped(w, EventFlowCredit, FlowCredit{
		SessionId: sessionID,
		Frames:    frames,
	}, nil)
}
