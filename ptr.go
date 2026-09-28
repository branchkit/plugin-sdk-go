package branchkit

// Ptr returns a pointer to v. Optional fields on the generated request
// structs are pointers, so an absent field is nil and a present one is
// written inline:
//
//	p.HUDCreateChannel(branchkit.HUDCreateChannelRequest{
//		Channel:      "status",
//		AcceptsInput: branchkit.Ptr(true),
//	})
func Ptr[T any](v T) *T { return &v }
