package loop

import (
	"time"

	"github.com/compozy/agh/internal/loop/gate"
	watchpkg "github.com/compozy/agh/internal/loop/watch"
)

// DefaultWatchSilenceWindow is the fallback inactivity clock for watch-source loops.
const DefaultWatchSilenceWindow = 30 * time.Minute

// WatchPoller is the coordinator seam for daemon -> extension watch/poll calls.
type WatchPoller = watchpkg.Poller

// CoordinatorRunnerOption configures the loop coordinator runner.
type CoordinatorRunnerOption func(*CoordinatorRunner)

// WithCoordinatorHookDispatcher injects loop hook dispatch for coordinator call sites.
func WithCoordinatorHookDispatcher(hooks HookDispatcher) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		r.hooks = hooks
	}
}

// WithCoordinatorWatchPoller injects the watch-source poll bridge used by source/watch-source nodes.
func WithCoordinatorWatchPoller(poller WatchPoller) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		r.watchPoller = poller
	}
}

// WithCoordinatorGateEvaluator injects runtime evaluation for gate control nodes.
func WithCoordinatorGateEvaluator(evaluator gate.GateEvaluator) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		r.gateEvaluator = evaluator
	}
}

// WithCoordinatorActionRegistry injects runtime action execution for worker node runs.
func WithCoordinatorActionRegistry(registry *ActionRegistry) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		r.actionRegistry = registry
	}
}

// WithCoordinatorDefaultsResolver injects workspace-aware `[loops.defaults.*]` resolution.
func WithCoordinatorDefaultsResolver(resolver DefaultsResolver) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		r.defaultsResolver = resolver
	}
}

// WithCoordinatorWatchSilenceWindow overrides the watch-source inactivity window.
func WithCoordinatorWatchSilenceWindow(window time.Duration) CoordinatorRunnerOption {
	return func(r *CoordinatorRunner) {
		if window > 0 {
			r.watchSilenceWindow = window
		}
	}
}
