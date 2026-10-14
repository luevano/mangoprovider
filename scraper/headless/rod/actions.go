package rod

import (
	"context"

	"github.com/go-rod/rod"
)

// Action: Something to run on the page once it's loaded, before its html is read.
type Action func(*rod.Page) error

type actionKey struct{}

// WithAction: The request made with ctx runs a on its page.
func WithAction(ctx context.Context, a Action) context.Context {
	if a == nil {
		return ctx
	}
	return context.WithValue(ctx, actionKey{}, a)
}

func action(ctx context.Context) Action {
	a, _ := ctx.Value(actionKey{}).(Action)
	return a
}
