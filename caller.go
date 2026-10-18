package branchkit

import "time"

// Caller makes one platform RPC call. It is all a generated wrapper needs:
// *Plugin is one, and a test can drive the wrappers over its own fake with
// NewMethods, no transport needed.
type Caller interface {
	Call(method string, params any, result any) error
}

// WithTimeout returns the typed wrappers with d as each call's timeout, in
// place of Call's 10 s default:
//
//	res, err := p.WithTimeout(30 * time.Second).PipelinesRun(req)
//
// A timeout is not a failure (see CallTimeoutError): the platform may have
// carried the call out.
func (p *Plugin) WithTimeout(d time.Duration) Methods {
	return NewMethods(timeoutCaller{p: p, d: d})
}

// timeoutCaller is a Plugin whose Call waits d.
type timeoutCaller struct {
	p *Plugin
	d time.Duration
}

func (t timeoutCaller) Call(method string, params any, result any) error {
	return t.p.CallWithTimeout(method, params, result, t.d)
}
