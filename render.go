package branchkit

import (
	"context"
	"io"
	"strings"
)

// HTMLComponent is anything that writes itself as HTML — the shape a
// templ component has. Declared here rather than imported so the SDK
// carries no templ dependency: a `templ.Component` satisfies it, and so
// does any other renderer with the same signature.
type HTMLComponent interface {
	Render(ctx context.Context, w io.Writer) error
}

// RenderComponent renders c to a string. Its signature matches
// [SettingsTabFunc], so a tab renderer can be one line:
//
//	plugin.SettingsTab("apps", func(req *branchkit.RenderSettingsRequest) (string, error) {
//	    return branchkit.RenderComponent(views.Apps(rows))
//	})
//
// A render error propagates instead of becoming an empty tab: the platform
// draws it as the tab's error state.
func RenderComponent(c HTMLComponent) (string, error) {
	var b strings.Builder
	if err := c.Render(context.Background(), &b); err != nil {
		return "", err
	}
	return b.String(), nil
}
