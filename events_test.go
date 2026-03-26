package shared

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestReadSSE_ParsesEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("missing Accept header")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// SSE keepalive comment (should be skipped)
		fmt.Fprintf(w, ": keepalive\n\n")
		flusher.Flush()

		// Two events
		fmt.Fprintf(w, "data: {\"event_type\":\"_platform.store.updated\",\"source\":\"_platform\",\"timestamp\":1,\"data\":{\"store\":\"key_names\"}}\n\n")
		flusher.Flush()

		fmt.Fprintf(w, "data: {\"event_type\":\"_platform.keyboard.layout_changed\",\"source\":\"_platform\",\"timestamp\":2,\"data\":{\"layout_id\":\"US\"}}\n\n")
		flusher.Flush()

		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	var events []PluginEventMessage
	var mu sync.Mutex
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = readSSE(ctx, srv.Client(), srv.URL+"/v1/plugins/events", "test-token", func(msg PluginEventMessage) {
			mu.Lock()
			events = append(events, msg)
			if len(events) == 2 {
				close(done)
			}
			mu.Unlock()
		})
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for events")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventType != "_platform.store.updated" {
		t.Errorf("event[0].EventType = %q, want _platform.store.updated", events[0].EventType)
	}
	if events[0].Timestamp != 1 {
		t.Errorf("event[0].Timestamp = %d, want 1", events[0].Timestamp)
	}
	if events[1].EventType != "_platform.keyboard.layout_changed" {
		t.Errorf("event[1].EventType = %q, want _platform.keyboard.layout_changed", events[1].EventType)
	}
}

func TestReadSSE_SkipsMalformedLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// Malformed JSON (should be skipped)
		fmt.Fprintf(w, "data: not-json\n\n")
		flusher.Flush()

		// Empty data line (should be skipped)
		fmt.Fprintf(w, "data:\n\n")
		flusher.Flush()

		// Valid event
		fmt.Fprintf(w, "data: {\"event_type\":\"test.ok\",\"source\":\"test\",\"timestamp\":1,\"data\":{}}\n\n")
		flusher.Flush()

		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	var events []PluginEventMessage
	var mu sync.Mutex
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = readSSE(ctx, srv.Client(), srv.URL+"/v1/plugins/events", "", func(msg PluginEventMessage) {
			mu.Lock()
			events = append(events, msg)
			close(done)
			mu.Unlock()
		})
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for events")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 1 {
		t.Fatalf("expected 1 event (skipping malformed), got %d", len(events))
	}
	if events[0].EventType != "test.ok" {
		t.Errorf("event.EventType = %q, want test.ok", events[0].EventType)
	}
}

func TestReadSSE_ReturnsErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	err := readSSE(context.Background(), srv.Client(), srv.URL+"/v1/plugins/events", "", func(msg PluginEventMessage) {
		t.Error("handler should not be called")
	})

	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if err.Error() != "HTTP 401" {
		t.Errorf("error = %q, want 'HTTP 401'", err.Error())
	}
}
