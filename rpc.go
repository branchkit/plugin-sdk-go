package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// --- JSON-RPC 2.0 message types ---

// rpcMessage is the wire format for all JSON-RPC 2.0 messages.
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- Handler types ---

// HandlerFunc handles an actuator→plugin request and returns a result or error.
type HandlerFunc func(params json.RawMessage) (any, error)

// ListenerFunc handles an actuator→plugin notification (fire-and-forget).
type ListenerFunc func(params json.RawMessage)

// --- Pending call tracking ---

type pendingCall struct {
	ch chan callResult
}

type callResult struct {
	Result json.RawMessage
	Error  *rpcError
}

// --- Plugin struct ---

// Plugin manages bidirectional JSON-RPC 2.0 communication over stdin/stdout.
//
// Handle() and On() must be called before Run(). Call() may be called from
// any goroutine concurrently with Run().
type Plugin struct {
	pluginID string

	// stdin/stdout are from the plugin's perspective:
	// - we READ from os.Stdin (actuator writes to our stdin)
	// - we WRITE to os.Stdout (actuator reads from our stdout)
	writer  *json.Encoder
	scanner *bufio.Scanner

	handlers  map[string]HandlerFunc
	listeners map[string][]ListenerFunc

	pending map[uint64]*pendingCall
	nextID  atomic.Uint64

	mu        sync.Mutex // protects writer, pending
	closed    chan struct{}
	closeOnce sync.Once
}

// NewPlugin creates a new Plugin that communicates via stdin/stdout.
// The read loop starts immediately — Call() works from this point.
func NewPlugin() *Plugin {
	pluginID := os.Getenv("BRANCHKIT_PLUGIN_ID")
	if pluginID == "" {
		pluginID = "unknown"
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Allow up to 1MB per line for large payloads (settings HTML, HUD content)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	p := &Plugin{
		pluginID:  pluginID,
		writer:    json.NewEncoder(os.Stdout),
		scanner:   scanner,
		handlers:  make(map[string]HandlerFunc),
		listeners: make(map[string][]ListenerFunc),
		pending:   make(map[uint64]*pendingCall),
		closed:    make(chan struct{}),
	}

	// Handle SIGTERM gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-sigCh:
			Log(pluginID, "shutting down (signal)")
			p.closeOnce.Do(func() { close(p.closed) })
		case <-p.closed:
		}
		signal.Stop(sigCh)
	}()

	Log(pluginID, "started (JSON-RPC over stdio)")
	go p.readLoop()

	return p
}

// Handle registers a handler for actuator→plugin requests.
// The handler receives the params and returns a result (serialized as JSON) or an error.
func (p *Plugin) Handle(method string, fn HandlerFunc) {
	p.handlers[method] = fn
}

// On registers a listener for actuator→plugin notifications (fire-and-forget).
// Multiple listeners can be registered for the same method.
func (p *Plugin) On(method string, fn ListenerFunc) {
	p.listeners[method] = append(p.listeners[method], fn)
}

// Call sends a request to the actuator and blocks until a response arrives or timeout.
// The result is unmarshaled into the provided result pointer (pass nil to discard).
func (p *Plugin) Call(method string, params any, result any) error {
	return p.CallWithTimeout(method, params, result, 10*time.Second)
}

// CallWithTimeout sends a request with a custom timeout.
func (p *Plugin) CallWithTimeout(method string, params any, result any, timeout time.Duration) error {
	id := p.nextID.Add(1)

	var paramsRaw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		paramsRaw = data
	}

	pc := &pendingCall{ch: make(chan callResult, 1)}

	p.mu.Lock()
	p.pending[id] = pc
	err := p.writer.Encode(rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  paramsRaw,
	})
	p.mu.Unlock()

	if err != nil {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return fmt.Errorf("write request: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-pc.ch:
		if res.Error != nil {
			return fmt.Errorf("rpc error %d: %s", res.Error.Code, res.Error.Message)
		}
		if result != nil && len(res.Result) > 0 {
			return json.Unmarshal(res.Result, result)
		}
		return nil
	case <-timer.C:
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return fmt.Errorf("rpc call %q timed out after %v", method, timeout)
	case <-p.closed:
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return fmt.Errorf("plugin shutting down")
	}
}

// Notify sends a fire-and-forget notification to the actuator (no response expected).
func (p *Plugin) Notify(method string, params any) error {
	var paramsRaw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		paramsRaw = data
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.writer.Encode(rpcMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	})
}

// Run blocks until the plugin shuts down (stdin closes or SIGTERM).
// The read loop is already running — started by NewPlugin(). Call() works
// before Run() is called.
func (p *Plugin) Run() {
	<-p.closed
}

func (p *Plugin) readLoop() {
	for p.scanner.Scan() {
		line := p.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg rpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			Log(p.pluginID, fmt.Sprintf("failed to parse message: %v", err))
			continue
		}

		p.dispatch(msg)
	}

	if err := p.scanner.Err(); err != nil {
		Log(p.pluginID, fmt.Sprintf("stdin read error: %v", err))
	}

	// Signal shutdown to any in-flight Call() waiters
	p.closeOnce.Do(func() { close(p.closed) })

	// Resolve all pending calls with error
	p.mu.Lock()
	for id, pc := range p.pending {
		pc.ch <- callResult{Error: &rpcError{Code: -1, Message: "stdin closed"}}
		delete(p.pending, id)
	}
	p.mu.Unlock()

	Log(p.pluginID, "stdin closed, exiting")
}

// dispatch routes an incoming message to the appropriate handler.
func (p *Plugin) dispatch(msg rpcMessage) {
	// Response to a pending Call() — has id + (result or error), no method
	if msg.ID != nil && msg.Method == "" {
		p.mu.Lock()
		pc, ok := p.pending[*msg.ID]
		if ok {
			delete(p.pending, *msg.ID)
		}
		p.mu.Unlock()

		if ok {
			pc.ch <- callResult{Result: msg.Result, Error: msg.Error}
		}
		return
	}

	// Request from actuator — has id + method
	// Run in goroutine so handlers can call plugin.Call() without blocking the read loop (C1).
	if msg.ID != nil && msg.Method != "" {
		go p.handleRequest(msg)
		return
	}

	// Notification from actuator — has method, no id
	if msg.ID == nil && msg.Method != "" {
		p.handleNotification(msg)
		return
	}
}

// handleRequest processes an actuator→plugin request and sends a response.
func (p *Plugin) handleRequest(msg rpcMessage) {
	handler, ok := p.handlers[msg.Method]
	if !ok {
		p.sendError(*msg.ID, -32601, fmt.Sprintf("method not found: %s", msg.Method))
		return
	}

	// Run handler with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				Log(p.pluginID, fmt.Sprintf("handler panic for %s: %v", msg.Method, r))
				p.sendError(*msg.ID, -1, fmt.Sprintf("handler panic: %v", r))
			}
		}()

		result, err := handler(msg.Params)
		if err != nil {
			p.sendError(*msg.ID, -1, err.Error())
			return
		}

		resultRaw, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			p.sendError(*msg.ID, -1, fmt.Sprintf("marshal result: %v", marshalErr))
			return
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		if err := p.writer.Encode(rpcMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  resultRaw,
		}); err != nil {
			Log(p.pluginID, fmt.Sprintf("write response for %s failed: %v", msg.Method, err))
		}
	}()
}

// handleNotification processes an actuator→plugin notification.
func (p *Plugin) handleNotification(msg rpcMessage) {
	listeners := p.listeners[msg.Method]
	for _, fn := range listeners {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Log(p.pluginID, fmt.Sprintf("listener panic for %s: %v", msg.Method, r))
				}
			}()
			fn(msg.Params)
		}()
	}
}

// sendError sends a JSON-RPC error response.
func (p *Plugin) sendError(id uint64, code int, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.writer.Encode(rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Error:   &rpcError{Code: code, Message: message},
	}); err != nil {
		Log(p.pluginID, fmt.Sprintf("write error response failed: %v", err))
	}
}
