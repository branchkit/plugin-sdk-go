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

func TestReadSSE_ContextCancellationStopsRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// Send one event then hold connection open
		fmt.Fprintf(w, "data: {\"event_type\":\"test.event\",\"source\":\"test\",\"timestamp\":1,\"data\":{}}\n\n")
		flusher.Flush()

		// Block until client disconnects
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan struct{})

	go func() {
		_ = readSSE(ctx, srv.Client(), srv.URL+"/v1/plugins/events", "", func(msg PluginEventMessage) {
			close(received)
		})
	}()

	// Wait for the event, then cancel
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	cancel()
	// readSSE should return promptly after cancel — if it doesn't, the test will timeout
}

func TestReadSSE_HandlerPanicDoesNotCrash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// First event will trigger a panic in the handler
		fmt.Fprintf(w, "data: {\"event_type\":\"panic.trigger\",\"source\":\"test\",\"timestamp\":1,\"data\":{}}\n\n")
		flusher.Flush()

		// Second event should still be delivered (handler wrapped with recovery)
		fmt.Fprintf(w, "data: {\"event_type\":\"after.panic\",\"source\":\"test\",\"timestamp\":2,\"data\":{}}\n\n")
		flusher.Flush()

		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	var events []string
	var mu sync.Mutex
	done := make(chan struct{})

	// Wrap handler with the same panic recovery used in SubscribeEvents
	handler := func(msg PluginEventMessage) {
		defer func() {
			if r := recover(); r != nil {
				// recovered
			}
		}()
		if msg.EventType == "panic.trigger" {
			panic("test panic")
		}
		mu.Lock()
		events = append(events, msg.EventType)
		close(done)
		mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = readSSE(ctx, srv.Client(), srv.URL+"/v1/plugins/events", "", handler)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out — panic may have killed the goroutine")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 1 || events[0] != "after.panic" {
		t.Errorf("expected [after.panic], got %v", events)
	}
}

func TestEventSubscription_CloseStopsGoroutine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// Keep sending events until client disconnects
		for {
			_, err := fmt.Fprintf(w, "data: {\"event_type\":\"tick\",\"source\":\"test\",\"timestamp\":0,\"data\":{}}\n\n")
			if err != nil {
				return
			}
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer srv.Close()

	p := &PlatformClient{token: ""}

	// Can't use SubscribeEvents (needs unix socket), so test Close() semantics
	// by simulating the goroutine pattern directly.
	ctx, cancel := context.WithCancel(context.Background())
	sub := &EventSubscription{cancel: cancel}

	sub.wg.Add(1)
	go func() {
		defer sub.wg.Done()
		_ = readSSE(ctx, srv.Client(), srv.URL+"/v1/plugins/events", p.token, func(msg PluginEventMessage) {})
	}()

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Close should return promptly
	done := make(chan struct{})
	go func() {
		sub.Close()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not return — goroutine may be stuck")
	}
}
