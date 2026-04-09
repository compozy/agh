package hooks

import (
	"context"
	"errors"
	"fmt"
)

const maxDispatchDepth = 3

type dispatchDepthContextKey struct{}

var ErrDispatchDepthExceeded = errors.New("hooks: dispatch depth exceeded")

func enterDispatch(ctx context.Context, event HookEvent) (context.Context, int, error) {
	depth := currentDispatchDepth(ctx) + 1
	if depth > maxDispatchDepth {
		return ctx, currentDispatchDepth(ctx), fmt.Errorf(
			"%w for event %q: depth %d exceeds max %d",
			ErrDispatchDepthExceeded,
			event,
			depth,
			maxDispatchDepth,
		)
	}

	return context.WithValue(ctx, dispatchDepthContextKey{}, depth), depth, nil
}

func currentDispatchDepth(ctx context.Context) int {
	if ctx == nil {
		return 0
	}

	depth, _ := ctx.Value(dispatchDepthContextKey{}).(int)
	return depth
}
