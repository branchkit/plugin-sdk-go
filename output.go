package branchkit

import "encoding/json"

// Semantic output helpers — parity with plugin-sdk-ts/src/output.ts and
// plugin-sdk-py/branchkit/output.py.
//
// The generated OutputState(state OutputState) wrapper is the whole call; a
// plugin states what is true for the person (OutputState, OutputSection,
// OutputItem in types_gen.go, OutputKind* / OutputUrgency* in
// closed_vocab_gen.go) and never sees a renderer. An item's Action is one
// of exactly two shapes — say these words, or dispatch this action type —
// and these build them so a producer never writes `&words` by hand.
//
// Producers state what is true (a kind, human-language phrases, an urgency)
// and never choose how or whether it is shown or spoken; every renderer reads
// the same document.

// SayAction is the action that injects `words` as if the person had spoken
// them — routed through the same matcher their voice reaches, so confirming
// the item is indistinguishable from saying it. The common case for a
// command.
func SayAction(words string) *OutputAction {
	return &OutputAction{Say: &words}
}

// DispatchAction is the action that dispatches `actionType` directly with
// `params` (nil for none) — for items that are not commands. `params` is
// marshalled here so the call site passes an ordinary map or struct.
func DispatchAction(actionType string, params any) *OutputAction {
	a := &OutputAction{Dispatch: &actionType}
	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			panic("branchkit.DispatchAction: params: " + err.Error())
		}
		a.Params = p
	}
	return a
}
