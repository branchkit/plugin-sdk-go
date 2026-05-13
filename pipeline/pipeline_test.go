package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestRoundtripNoPayload(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	ev := &Event{
		Type: "audio_stop",
		Data: json.RawMessage(`{"session_id":"abc"}`),
	}
	if err := w.WriteEvent(ev); err != nil {
		t.Fatal(err)
	}

	r := NewReader(&buf)
	got, err := r.ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "audio_stop" {
		t.Fatalf("type = %q, want audio_stop", got.Type)
	}
	if string(got.Data) != `{"session_id":"abc"}` {
		t.Fatalf("data = %s, want {\"session_id\":\"abc\"}", got.Data)
	}
	if len(got.Payload) != 0 {
		t.Fatalf("payload len = %d, want 0", len(got.Payload))
	}
}

func TestRoundtripWithPayload(t *testing.T) {
	payload := make([]byte, 640)
	for i := range payload {
		payload[i] = byte(i & 0xff)
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)
	ev := &Event{
		Type:    "audio_chunk",
		Data:    json.RawMessage(`{"session_id":"abc","timestamp_ms":120}`),
		Payload: payload,
	}
	if err := w.WriteEvent(ev); err != nil {
		t.Fatal(err)
	}

	r := NewReader(&buf)
	got, err := r.ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "audio_chunk" {
		t.Fatalf("type = %q, want audio_chunk", got.Type)
	}
	if !bytes.Equal(got.Payload, payload) {
		t.Fatalf("payload mismatch")
	}
}

func TestMultipleEventsInOrder(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for i := range 5 {
		ev := &Event{
			Type:    "audio_chunk",
			Data:    json.RawMessage(fmt.Sprintf(`{"session_id":"s","timestamp_ms":%d}`, i*20)),
			Payload: bytes.Repeat([]byte{byte(i)}, 16),
		}
		if err := w.WriteEvent(ev); err != nil {
			t.Fatal(err)
		}
	}

	r := NewReader(&buf)
	for i := range 5 {
		got, err := r.ReadEvent()
		if err != nil {
			t.Fatalf("event %d: %v", i, err)
		}
		if got.Type != "audio_chunk" {
			t.Fatalf("event %d: type = %q", i, got.Type)
		}
		want := bytes.Repeat([]byte{byte(i)}, 16)
		if !bytes.Equal(got.Payload, want) {
			t.Fatalf("event %d: payload mismatch", i)
		}
	}
}

func TestMalformedHeaderReturnsError(t *testing.T) {
	r := NewReader(strings.NewReader("not-json\n"))
	_, err := r.ReadEvent()
	if err == nil {
		t.Fatal("expected error for malformed header")
	}
	if !strings.Contains(err.Error(), "wire: bad header") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanEOFReturnsNilEOF(t *testing.T) {
	r := NewReader(strings.NewReader(""))
	got, err := r.ReadEvent()
	if got != nil {
		t.Fatalf("expected nil event, got %+v", got)
	}
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestHeaderOmitsZeroPayloadLength(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	ev := &Event{
		Type: "audio_stop",
		Data: json.RawMessage(`{"session_id":"x"}`),
	}
	if err := w.WriteEvent(ev); err != nil {
		t.Fatal(err)
	}
	line := buf.String()
	if strings.Contains(line, "payload_length") {
		t.Fatalf("header should omit payload_length when zero, got: %s", line)
	}
}

func TestHeaderOmitsEmptyData(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	ev := &Event{
		Type: "audio_stop",
	}
	if err := w.WriteEvent(ev); err != nil {
		t.Fatal(err)
	}
	line := buf.String()
	if strings.Contains(line, `"data"`) {
		t.Fatalf("header should omit data when nil, got: %s", line)
	}
}

func TestOversizedPayloadLengthRejected(t *testing.T) {
	// Header claims 32 MB payload.
	header := `{"type":"audio_chunk","data":{},"payload_length":33554432}` + "\n"
	r := NewReader(strings.NewReader(header))
	_, err := r.ReadEvent()
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
	if !strings.Contains(err.Error(), "16 MB cap") {
		t.Fatalf("unexpected error: %v", err)
	}
}
