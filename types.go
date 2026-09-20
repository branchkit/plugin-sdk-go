package branchkit

// Hand-written type extensions and convenience types.
// Generated types are in types_gen.go.

// Rect is an integer rectangle (SDK convenience, not part of the RPC protocol).
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}
