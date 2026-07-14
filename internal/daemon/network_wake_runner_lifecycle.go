package daemon

import (
	"context"
	"errors"
)

// Start binds the wake runner to the daemon lifecycle.
func (r *networkWakeRunner) Start(ctx context.Context) error {
	if r == nil {
		return errors.New("daemon: network wake runner is required")
	}
	if ctx == nil {
		return errors.New("daemon: network wake runner context is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runDone != nil {
		return errors.New("daemon: network wake runner already started")
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	r.runCancel = cancel
	r.runDone = done
	go func() {
		done <- r.Run(runCtx)
	}()
	return nil
}

// Shutdown cancels active provider turns and waits for the runner to stop.
func (r *networkWakeRunner) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: network wake runner shutdown context is required")
	}
	r.mu.Lock()
	cancel := r.runCancel
	done := r.runDone
	r.mu.Unlock()
	if done == nil {
		return nil
	}
	cancel()
	select {
	case err := <-done:
		r.mu.Lock()
		r.runCancel = nil
		r.runDone = nil
		r.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
