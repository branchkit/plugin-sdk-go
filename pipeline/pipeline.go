// Package pipeline provides wire protocol reader/writer helpers for
// pipeline stages. Each wire event is a JSON header line terminated by
// '\n', optionally followed by exactly payload_length bytes of binary
// payload.
package pipeline

import (
	"bufio"
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
func (wr *Writer) WriteEvent(ev *Event) error {
	h := wireHeader{
		Type:          ev.Type,
		Data:          ev.Data,
		PayloadLength: len(ev.Payload),
	}
	hdr, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("wire: marshal header: %w", err)
	}
	hdr = append(hdr, '\n')
	if _, err := wr.w.Write(hdr); err != nil {
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
