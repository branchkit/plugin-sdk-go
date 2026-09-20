// Package pipeline provides wire protocol reader/writer helpers for
// pipeline stages. Each wire event is a JSON header line terminated by
// '\n', optionally followed by exactly payload_length bytes of binary
// payload.
package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// MaxPayload is the largest binary payload the reader will accept (16 MB).
const MaxPayload = 16 * 1024 * 1024

// Event is a single wire-format message: a type tag, an opaque JSON data
// blob, and an optional binary payload.
type Event struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data,omitempty"`
	Payload []byte          `json:"-"`
}

// wireHeader is the JSON line written/read on the wire.
type wireHeader struct {
	Type          string          `json:"type"`
	Data          json.RawMessage `json:"data,omitempty"`
	PayloadLength int             `json:"payload_length,omitempty"`
}

// Reader reads framed events from an io.Reader.
type Reader struct {
	r *bufio.Reader
}

// NewReader wraps r in a pipeline Reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(r)}
}

// ReadEvent reads the next event from the stream. Returns nil, io.EOF on
// a clean stream close.
func (r *Reader) ReadEvent() (*Event, error) {
	line, err := r.r.ReadBytes('\n')
	if err != nil {
		if err == io.EOF && len(line) == 0 {
			return nil, io.EOF
		}
		if err == io.EOF {
			// Partial line with no newline — treat as malformed.
			return nil, fmt.Errorf("wire: incomplete header (no trailing newline)")
		}
		return nil, err
	}

	var h wireHeader
	if err := json.Unmarshal(line, &h); err != nil {
		return nil, fmt.Errorf("wire: bad header %q: %w", string(line), err)
	}

	if h.PayloadLength > MaxPayload {
		return nil, fmt.Errorf("wire: payload_length %d exceeds 16 MB cap", h.PayloadLength)
	}

	var payload []byte
	if h.PayloadLength > 0 {
		payload = make([]byte, h.PayloadLength)
		if _, err := io.ReadFull(r.r, payload); err != nil {
			return nil, err
		}
	}

	return &Event{
		Type:    h.Type,
		Data:    h.Data,
		Payload: payload,
	}, nil
}

// Writer writes framed events to an io.Writer.
type Writer struct {
	w *bufio.Writer
}

// NewWriter wraps w in a pipeline Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// WriteEvent writes a single event and flushes immediately.
//
// Empty data — nil, `{}`, or `null` — is omitted from the header, matching
// the Rust and TS writers (the wire contract's omitted-when-empty rule).
// Without this, an event decoded from a lenient peer's `data:{}` would
// re-serialize non-canonically.
func (wr *Writer) WriteEvent(ev *Event) error {
	data := ev.Data
	if s := string(bytes.TrimSpace(data)); s == "" || s == "{}" || s == "null" {
		data = nil
	}
	h := wireHeader{
		Type:          ev.Type,
		Data:          data,
		PayloadLength: len(ev.Payload),
	}
	// json.Marshal escapes <, > and & as \u003c / \u003e / \u0026. The Rust
	// writer is the canonical encoding this port is held byte-identical to
	// (stage-sdk-test's framing suite builds its expected bytes with it), and
	// serde_json does not escape them — nor do the TS and Python ports. Go was
	// the only one that did, and the suite was green only because no fixture
	// carried those characters. An Encoder with SetEscapeHTML(false) is the
	// documented way off that default; it also appends the '\n' we want.
	// Found 2026-09-19 while porting the Python framing.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(h); err != nil {
		return fmt.Errorf("wire: marshal header: %w", err)
	}
	if _, err := wr.w.Write(buf.Bytes()); err != nil {
		return err
	}
	if len(ev.Payload) > 0 {
		if _, err := wr.w.Write(ev.Payload); err != nil {
			return err
		}
	}
	return wr.w.Flush()
}

// Flush flushes the underlying buffer.
func (wr *Writer) Flush() error {
	return wr.w.Flush()
}
