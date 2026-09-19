package branchkit

import "encoding/json"

// Semantic output helpers — parity with plugin-sdk-ts/src/output.ts and
// plugin-sdk-py/branchkit/output.py.
//
// The generated OutputState(state OutputState) wrapper is the whole call; a
// plugin states what is true for the person (OutputState, OutputSection,
// OutputItem in types_gen.go, OutputKind* / OutputUrgency* in
// closed_vocab_gen.go) and never sees a renderer. The one awkward corner is
// an item's Action, which the generator types as json.RawMessage because the
// wire allows it to be absent: these two build the only two shapes it can
// take, so a producer never hand-writes the envelope.
//
// Design: docs/design/DESIGN_SEMANTIC_OUTPUT_CHANNEL.md.

// SayAction is the action that injects `words` as if the person had spoken
// them — routed through the same matcher their voice reaches, so confirming
// the item is indistinguishable from saying it. The common case for a
// command.
func SayAction(words string) json.RawMessage {
	b, err := json.Marshal(OutputAction{Say: &words})
	if err != nil {
		// A string field cannot fail to marshal; keep the signature honest
		// without forcing every producer to handle an impossible error.
		panic("branchkit.SayAction: " + err.Error())
	}
	return b
}

// DispatchAction is the action that dispatches `actionType` directly with
// `params` (nil for none) — for items that are not commands.
func DispatchAction(actionType string, params any) json.RawMessage {
	a := OutputAction{Dispatch: &actionType}
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			panic("branchkit.DispatchAction: params: " + err.Error())
		}
		a.Params = p
	}
	b, err := json.Marshal(a)
	if err != nil {
		panic("branchkit.DispatchAction: " + err.Error())
	}
	return b
}
