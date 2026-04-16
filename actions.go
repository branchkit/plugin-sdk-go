package shared

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ActionHandlerFunc handles a single dispatched action. It receives the full
// OnActionRequest (action name, params, and active app/window context). The
// returned value is marshaled into the JSON-RPC response.
//
// For typed params, use UnmarshalParams or the HandleActionTyped free function.
type ActionHandlerFunc func(req *OnActionRequest) (any, error)

// UnmarshalParams decodes the request's params field into a typed struct.
// Returns nil if the request has no params (the destination is left zero-valued).
//
//	type SnapParams struct {
//	    Position string `json:"position"`
//	}
//	plugin.HandleAction("wm.snap", func(req *shared.OnActionRequest) (any, error) {
//	    var p SnapParams
//	    if err := req.UnmarshalParams(&p); err != nil {
//	        return nil, err
//	    }
//	    // use p.Position
//	    return nil, nil
//	})
func (r *OnActionRequest) UnmarshalParams(dst any) error {
	if len(r.Params) == 0 {
		return nil
	}
	return json.Unmarshal(r.Params, dst)
}

// actionRegistry holds per-action handlers. It is installed as the on_action
// HandlerFunc the first time HandleAction is called.
type actionRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ActionHandlerFunc
}

func (r *actionRegistry) set(action string, fn ActionHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[action] = fn
}

func (r *actionRegistry) get(action string) (ActionHandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.handlers[action]
	return fn, ok
}

// dispatch is the on_action HandlerFunc installed by the registry. It demuxes
// by req.Action: per-action handler if registered, otherwise OnActionResponse{
// Status: NotHandled}.
func (r *actionRegistry) dispatch(params json.RawMessage) (any, error) {
	var req OnActionRequest
	if len(params) > 0 {
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("on_action: bad params: %w", err)
		}
	}
	if fn, ok := r.get(req.Action); ok {
		return fn(&req)
	}
	return OnActionResponse{Status: OnActionStatusNotHandled}, nil
}

// HandleAction registers a handler for a single dispatched action type
// (e.g. "wm.snap", "voice.dictation_start"). The SDK installs an internal
// on_action handler that demuxes by req.Action.
//
// HandleAction is the only supported way to register action handlers.
// Calling Handle("on_action", ...) directly is reserved for plugins with
// dynamic dispatch needs (e.g. browser, which forwards every browser.*
// action to SSE clients) — but mixing the two will panic, since each is
// installing the same handler key.
//
//	plugin.HandleAction("wm.snap", func(req *shared.OnActionRequest) (any, error) {
//	    var p struct{ Position string `json:"position"` }
//	    req.UnmarshalParams(&p)
//	    return nil, nil // implicit "ok" — SDK fills in the response
//	})
//
// Return value semantics:
//   - return nil → OnActionResponse{Status: Ok} with no control_message
//   - return OnActionResponse → returned verbatim
//   - return any other value → marshaled as the JSON-RPC result (caller decides shape)
//   - return error → translated to a JSON-RPC error response
func (p *Plugin) HandleAction(action string, fn ActionHandlerFunc) {
	if p.actionRegistry == nil {
		if _, exists := p.handlers[HookOnAction]; exists {
			panic("plugin-sdk-go: cannot mix Handle(\"on_action\", ...) and HandleAction(...) — pick one")
		}
		p.actionRegistry = &actionRegistry{handlers: make(map[string]ActionHandlerFunc)}
		p.handlers[HookOnAction] = p.actionRegistry.dispatch
	}
	wrapped := func(req *OnActionRequest) (any, error) {
		result, err := fn(req)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return OnActionResponse{Status: OnActionStatusOk}, nil
		}
		return result, nil
	}
	p.actionRegistry.set(action, wrapped)
}

// HandleActionTyped is a generic helper that unmarshals the request params into
// a typed struct before invoking the handler. This is a free function (not a
// method) because Go does not allow type parameters on methods.
//
//	type SnapParams struct {
//	    Position string `json:"position"`
//	}
//	shared.HandleActionTyped(plugin, "wm.snap", func(p SnapParams, req *shared.OnActionRequest) (any, error) {
//	    // p is fully typed
//	    return nil, nil
//	})
func HandleActionTyped[T any](p *Plugin, action string, fn func(params T, req *OnActionRequest) (any, error)) {
	p.HandleAction(action, func(req *OnActionRequest) (any, error) {
		var params T
		if err := req.UnmarshalParams(&params); err != nil {
			return nil, fmt.Errorf("%s: bad params: %w", action, err)
		}
		return fn(params, req)
	})
}

// RegisteredActionTypes returns the list of action types registered via
// HandleAction. Useful for the (future) list_action_types RPC and for tests.
// Returns nil if no per-action handlers have been registered.
func (p *Plugin) RegisteredActionTypes() []string {
	if p.actionRegistry == nil {
		return nil
	}
	p.actionRegistry.mu.RLock()
	defer p.actionRegistry.mu.RUnlock()
	out := make([]string, 0, len(p.actionRegistry.handlers))
	for k := range p.actionRegistry.handlers {
		out = append(out, k)
	}
	return out
}
