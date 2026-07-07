package loop

import (
	"context"
	"time"
)

// DefaultsResolver resolves the `[loops.defaults.*]` layer for one workspace.
type DefaultsResolver func(context.Context, WorkspaceID) (LoopDefaults, error)

// Option configures a loop service.
type Option func(*service)

// WithDefaults injects the `[loops.defaults.*]` layer consumed by the resolver.
func WithDefaults(defaults LoopDefaults) Option {
	return func(s *service) {
		s.defaults = defaults
	}
}

// WithDefaultsResolver injects workspace-aware `[loops.defaults.*]` resolution.
func WithDefaultsResolver(resolver DefaultsResolver) Option {
	return func(s *service) {
		s.defaultsResolver = resolver
	}
}

// WithClock injects a deterministic clock.
func WithClock(now func() time.Time) Option {
	return func(s *service) {
		s.now = now
	}
}

// WithRunIDFactory injects a deterministic run ID factory.
func WithRunIDFactory(factory func() RunID) Option {
	return func(s *service) {
		s.newRunID = factory
	}
}

// WithHookDispatcher injects loop hook dispatch at aggregate call sites.
func WithHookDispatcher(hooks HookDispatcher) Option {
	return func(s *service) {
		s.hooks = hooks
	}
}

func (s *service) resolveDefaults(ctx context.Context, ws WorkspaceID) (LoopDefaults, error) {
	if s.defaultsResolver == nil {
		return s.defaults, nil
	}
	return s.defaultsResolver(ctx, ws)
}
