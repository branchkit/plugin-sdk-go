package shared

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// PluginEventMessage is the envelope for events on the open event bus.
type PluginEventMessage struct {
	EventType     string          `json:"event_type"`
	Source        string          `json:"source"`
	Timestamp     uint64          `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
	CorrelationID *string         `json:"correlation_id,omitempty"`
}

// EventSubscription manages a persistent SSE connection to the actuator's event stream.
type EventSubscription struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Close terminates the SSE connection and waits for the reader goroutine to exit.
func (s *EventSubscription) Close() {
	s.cancel()
	s.wg.Wait()
}

// SubscribeEvents opens a persistent SSE connection to /v1/plugins/events and calls
// handler for each matching event. Topics are comma-joined and passed as a query parameter.
// Returns an EventSubscription that must be closed on shutdown.
//
// The connection automatically reconnects with exponential backoff if it drops.
func (p *PlatformClient) SubscribeEvents(topics []string, handler func(PluginEventMessage)) *EventSubscription {
	ctx, cancel := context.WithCancel(context.Background())
	sub := &EventSubscription{cancel: cancel}

	socketPath := os.Getenv("BRANCHKIT_SOCKET")
	if socketPath == "" {
		fmt.Fprintln(os.Stderr, "[events] BRANCHKIT_SOCKET not set — SSE subscription disabled")
		cancel()
		return sub
	}

	// Dedicated HTTP client for SSE — no timeout, separate from the request/response client.
	sseClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(dialCtx, "unix", socketPath)
			},
		},
	}

	url := "http://localhost/v1/plugins/events"
	if len(topics) > 0 {
		url += "?topic=" + strings.Join(topics, ",")
	}

	// Wrap handler with panic recovery so a bad handler doesn't kill the SSE goroutine.
	safeHandler := func(msg PluginEventMessage) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[events] handler panic for %s: %v\n", msg.EventType, r)
			}
		}()
		handler(msg)
	}

	sub.wg.Add(1)
	go func() {
		defer sub.wg.Done()
		backoff := time.Second

		for {
			if ctx.Err() != nil {
				return
			}

			err := readSSE(ctx, sseClient, url, p.token, safeHandler)
			if ctx.Err() != nil {
				return
			}

			fmt.Fprintf(os.Stderr, "[events] SSE disconnected: %v — reconnecting in %v\n", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 30*time.Second)
		}
	}()

	return sub
}

// readSSE connects to an SSE endpoint and reads events until the connection drops or ctx is cancelled.
func readSSE(ctx context.Context, client *http.Client, url string, token string, handler func(PluginEventMessage)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	fmt.Fprintf(os.Stderr, "[events] SSE connected to %s\n", url)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: "data: {json}" lines, blank lines delimit events
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}

		var msg PluginEventMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			fmt.Fprintf(os.Stderr, "[events] failed to parse event: %v\n", err)
			continue
		}
		handler(msg)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read: %w", err)
	}
	return fmt.Errorf("stream ended")
}
