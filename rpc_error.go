package branchkit

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrorKind is the closed-vocabulary classification the actuator attaches to
// an RPC error. The values are generated into `closed_vocab_gen.go` from
// `actuator/src/fault.rs::ErrorKind`.
//
// Deliberately a plain string, not an enum with validation: an actuator newer
// than this SDK may send a kind that has no constant here, and that must
// fall through a switch rather than fail to parse.
type ErrorKind string

// FaultData (the structured payload from a JSON-RPC error's `data` member)
// is generated into closed_vocab_gen.go alongside the ErrorKind values.

// RPCError is an error returned by the actuator in response to a plugin call.
//
// Recover it with errors.As:
//
//	var rpcErr *branchkit.RPCError
//	if errors.As(err, &rpcErr) && rpcErr.Kind == branchkit.ErrorKindNotPermitted {
//	    // the collection's shape forbids this op — different remedy from
//	    // ErrorKindForbidden, which means the caller lacks a privilege
//	}
//
// Branch on Kind, never on Message. Message is human-readable prose whose
// wording is not part of the contract.
//
// Version skew: an actuator that predates structured errors sends no `data`,
// in which case Kind is empty and only Code and Message are meaningful. Treat
// an unrecognized Kind the same as ErrorKindInternal.
type RPCError struct {
	// Code is the JSON-RPC error code. Derived from Kind actuator-side, so
	// the two never disagree; prefer Kind for branching.
	Code int
	// Message is human-readable and may change between releases.
	Message string
	// Kind is the machine-readable classification. Empty when the actuator
	// sent no structured data.
	Kind ErrorKind
	// Data is the full structured payload. Nil when absent from the wire.
	Data *FaultData
}

func (e *RPCError) Error() string {
	if e.Kind != "" {
		return fmt.Sprintf("rpc error %d (%s): %s", e.Code, e.Kind, e.Message)
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Is supports errors.Is against the per-kind sentinels below, so callers can
// write errors.Is(err, ErrRecordingDisabled) without unwrapping.
func (e *RPCError) Is(target error) bool {
	var s *sentinelError
	if !errors.As(target, &s) {
		return false
	}
	return e.Kind == s.kind
}

// sentinelError is the comparable target behind the exported per-kind
// sentinels. Kept unexported so the only way to make one is via this file.
type sentinelError struct {
	kind ErrorKind
}

func (s *sentinelError) Error() string { return string(s.kind) }

// Per-kind sentinels for errors.Is. Only the kinds a plugin realistically
// branches on get one; for anything else use errors.As and read Kind.
var (
	// ErrRecordingDisabled reports that a log collection has recording
	// turned off, so the append was refused.
	ErrRecordingDisabled error = &sentinelError{kind: ErrorKindRecordingDisabled}
	// ErrNotPermitted reports that the collection's shape forbids the op
	// (e.g. Put on an append-only log) regardless of caller.
	ErrNotPermitted error = &sentinelError{kind: ErrorKindNotPermitted}
	// ErrNotFound reports a missing collection, record, or target.
	ErrNotFound error = &sentinelError{kind: ErrorKindNotFound}
	// ErrForbidden reports that this caller is not authorized for the op.
	// Contrast ErrNotPermitted, which is about the collection's shape.
	ErrForbidden error = &sentinelError{kind: ErrorKindForbidden}
)

// ErrorKindOf returns the kind carried by err, if any. The bool is false when
// err is not an RPCError or carries no structured data.
func ErrorKindOf(err error) (ErrorKind, bool) {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) || rpcErr.Kind == "" {
		return "", false
	}
	return rpcErr.Kind, true
}

// toRPCError converts a wire error into the exported type.
func (e *rpcError) toRPCError() *RPCError {
	out := &RPCError{Code: e.Code, Message: e.Message}
	if len(e.Data) == 0 {
		return out
	}
	var d FaultData
	// A malformed or unexpected `data` must not turn a usable error into a
	// parse failure — the code and message still carry the story.
	if err := json.Unmarshal(e.Data, &d); err != nil {
		return out
	}
	out.Data = &d
	out.Kind = d.Kind
	return out
}
