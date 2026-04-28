// Package providertest contains reusable provider conformance checks.
package providertest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/pedronauck/agh/internal/sandbox"
)

// LifecycleCase configures the shared Provider lifecycle compliance suite.
type LifecycleCase struct {
	Provider         sandbox.Provider
	Backend          sandbox.Backend
	PrepareRequest   sandbox.PrepareRequest
	AssertPrepared   func(*testing.T, sandbox.Prepared)
	AssertFinalState func(*testing.T, sandbox.SessionState)
}

// RunLifecycle exercises the common prepare, sync-to, sync-from, destroy lifecycle.
func RunLifecycle(t *testing.T, tc LifecycleCase) sandbox.Prepared {
	t.Helper()

	prepared, err := runLifecycle(context.Background(), tc)
	if err != nil {
		t.Fatal(err)
	}
	if tc.AssertPrepared != nil {
		tc.AssertPrepared(t, prepared)
	}
	if tc.AssertFinalState != nil {
		tc.AssertFinalState(t, prepared.State)
	}

	return prepared
}

func runLifecycle(ctx context.Context, tc LifecycleCase) (sandbox.Prepared, error) {
	if tc.Provider == nil {
		return sandbox.Prepared{}, errors.New("provider = nil, want provider")
	}
	if tc.Backend != "" {
		if got := tc.Provider.Backend(); got != tc.Backend {
			return sandbox.Prepared{}, fmt.Errorf("Provider.Backend() = %q, want %q", got, tc.Backend)
		}
	}

	prepared, err := tc.Provider.Prepare(ctx, tc.PrepareRequest)
	if err != nil {
		return sandbox.Prepared{}, fmt.Errorf("Provider.Prepare() error = %w", err)
	}
	if prepared.Launcher == nil {
		return sandbox.Prepared{}, errors.New("Prepared.Launcher = nil, want launcher")
	}
	if prepared.ToolHost == nil {
		return sandbox.Prepared{}, errors.New("Prepared.ToolHost = nil, want tool host")
	}

	if _, err := tc.Provider.SyncToRuntime(ctx, prepared.State, sandbox.SyncOptions{
		Reason: sandbox.SyncReasonStart,
	}); err != nil {
		return sandbox.Prepared{}, fmt.Errorf("Provider.SyncToRuntime() error = %w", err)
	}
	if _, err := tc.Provider.SyncFromRuntime(ctx, prepared.State, sandbox.SyncOptions{
		Reason: sandbox.SyncReasonStop,
	}); err != nil {
		return sandbox.Prepared{}, fmt.Errorf("Provider.SyncFromRuntime() error = %w", err)
	}
	if err := tc.Provider.Destroy(ctx, prepared.State); err != nil {
		return sandbox.Prepared{}, fmt.Errorf("Provider.Destroy() error = %w", err)
	}

	return prepared, nil
}
