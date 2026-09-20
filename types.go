package branchkit

// Hand-written type extensions and convenience types.
// Generated types are in types_gen.go.

// Rect is an integer rectangle (SDK convenience, not part of the RPC
// protocol).
//
// It carries no json tags on purpose. It never crosses the wire — a caller
// converts to the generated WindowFrame / WindowBounds at the boundary, as
// plugins/tiling does — and tags on a type that is not a wire type make it
// read as one. It was counted as unreviewed wire surface by
// check-sdk-parity's coverage ratchet for exactly that reason (2026-09-19).
type Rect struct {
	X int
	Y int
	W int
	H int
}
