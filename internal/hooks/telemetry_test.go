package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestHookTelemetrySecurityPatchPersistsAllFields(t *testing.T) {
	t.Parallel()

	writer := &captureHookRunWriter{}
	hooks := newTelemetryTestHooks(t, false, HookDecl{
		Name:         "permission-hook",
		Event:        HookPermissionRequest,
		Mode:         HookModeSync,
		ExecutorKind: HookExecutorNative,
	}, map[string]Executor{
		"permission-hook": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ PermissionRequestPayload) (PermissionRequestPatch, error) {
			deny := "deny"
			return PermissionRequestPatch{
				Decision: &deny,
				Reason:   stringPointer("policy"),
			}, nil
		}),
	})

	ctx := WithHookRunWriter(t.Context(), writer)
	_, err := hooks.DispatchPermissionRequest(ctx, PermissionRequestPayload{
		PayloadBase: PayloadBase{Event: HookPermissionRequest},
		SessionContext: SessionContext{
			SessionID: "sess-security",
		},
		Decision: "allow",
	})
	if err != nil {
		t.Fatalf("DispatchPermissionRequest() error = %v", err)
	}

	record := writer.singleRecord(t)
	if record.HookName != "permission-hook" {
		t.Fatalf("record.HookName = %q, want permission-hook", record.HookName)
	}
	if record.Event != HookPermissionRequest {
		t.Fatalf("record.Event = %q, want %q", record.Event, HookPermissionRequest)
	}
	if record.Source != HookSourceNative {
		t.Fatalf("record.Source = %q, want %q", record.Source, HookSourceNative)
	}
	if record.Mode != HookModeSync {
		t.Fatalf("record.Mode = %q, want %q", record.Mode, HookModeSync)
	}
	if record.Outcome != HookRunOutcomeDenied {
		t.Fatalf("record.Outcome = %q, want %q", record.Outcome, HookRunOutcomeDenied)
	}
	if record.DispatchDepth != 1 {
		t.Fatalf("record.DispatchDepth = %d, want 1", record.DispatchDepth)
	}
	if len(record.PatchApplied) == 0 {
		t.Fatal("record.PatchApplied = nil, want captured security patch")
	}
	if record.Duration <= 0 {
		t.Fatalf("record.Duration = %s, want > 0", record.Duration)
	}
	if record.RecordedAt.IsZero() {
		t.Fatal("record.RecordedAt is zero")
	}
}

func TestHookTelemetryOmitsNonSecurityPatchOutsideDebug(t *testing.T) {
	t.Parallel()

	writer := &captureHookRunWriter{}
	hooks := newTelemetryTestHooks(t, false, HookDecl{
		Name:         "session-hook",
		Event:        HookSessionPostCreate,
		Mode:         HookModeSync,
		ExecutorKind: HookExecutorNative,
	}, map[string]Executor{
		"session-hook": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ SessionPostCreatePayload) (SessionPostCreatePatch, error) {
			return SessionPostCreatePatch{SessionName: stringPointer("patched")}, nil
		}),
	})

	ctx := WithHookRunWriter(t.Context(), writer)
	_, err := hooks.DispatchSessionPostCreate(ctx, SessionPostCreatePayload{
		PayloadBase: PayloadBase{Event: HookSessionPostCreate},
		SessionContext: SessionContext{
			SessionID: "sess-normal",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}

	if patch := writer.singleRecord(t).PatchApplied; len(patch) != 0 {
		t.Fatalf("PatchApplied = %s, want omitted in normal mode", patch)
	}
}

func TestHookTelemetryCapturesNonSecurityPatchInDebugMode(t *testing.T) {
	t.Parallel()

	writer := &captureHookRunWriter{}
	hooks := newTelemetryTestHooks(t, true, HookDecl{
		Name:         "session-hook",
		Event:        HookSessionPostCreate,
		Mode:         HookModeSync,
		ExecutorKind: HookExecutorNative,
	}, map[string]Executor{
		"session-hook": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ SessionPostCreatePayload) (SessionPostCreatePatch, error) {
			return SessionPostCreatePatch{SessionName: stringPointer("patched")}, nil
		}),
	})

	ctx := WithHookRunWriter(t.Context(), writer)
	_, err := hooks.DispatchSessionPostCreate(ctx, SessionPostCreatePayload{
		PayloadBase: PayloadBase{Event: HookSessionPostCreate},
		SessionContext: SessionContext{
			SessionID: "sess-debug",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}

	record := writer.singleRecord(t)
	if len(record.PatchApplied) == 0 {
		t.Fatal("PatchApplied = nil, want captured patch in debug mode")
	}
}

func TestHookTelemetryRecordsFailureOutcomeAndDuration(t *testing.T) {
	t.Parallel()

	writer := &captureHookRunWriter{}
	hooks := newTelemetryTestHooks(t, false, HookDecl{
		Name:         "failing-hook",
		Event:        HookSessionPostCreate,
		Mode:         HookModeSync,
		ExecutorKind: HookExecutorNative,
	}, map[string]Executor{
		"failing-hook": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ SessionPostCreatePayload) (SessionPostCreatePatch, error) {
			return SessionPostCreatePatch{}, errors.New("boom")
		}),
	})

	ctx := WithHookRunWriter(t.Context(), writer)
	_, err := hooks.DispatchSessionPostCreate(ctx, SessionPostCreatePayload{
		PayloadBase: PayloadBase{Event: HookSessionPostCreate},
		SessionContext: SessionContext{
			SessionID: "sess-failure",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v, want nil for non-required failure", err)
	}

	record := writer.singleRecord(t)
	if record.Outcome != HookRunOutcomeFailed {
		t.Fatalf("record.Outcome = %q, want %q", record.Outcome, HookRunOutcomeFailed)
	}
	if record.Error != "boom" {
		t.Fatalf("record.Error = %q, want boom", record.Error)
	}
	if record.Duration <= 0 {
		t.Fatalf("record.Duration = %s, want > 0", record.Duration)
	}
}

type captureHookRunWriter struct {
	records []HookRunRecord
}

func (c *captureHookRunWriter) RecordHookRun(_ context.Context, record HookRunRecord) error {
	c.records = append(c.records, cloneTelemetryRecord(record))
	return nil
}

func (c *captureHookRunWriter) singleRecord(t *testing.T) HookRunRecord {
	t.Helper()
	if got, want := len(c.records), 1; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	return c.records[0]
}

func newTelemetryTestHooks(t *testing.T, debug bool, decl HookDecl, executors map[string]Executor) *Hooks {
	t.Helper()

	hooks := NewHooks(
		WithLogger(discardPoolLogger()),
		WithDebugPatchAudit(debug),
		WithNativeDeclarations([]HookDecl{decl}),
		WithExecutorResolver(func(decl HookDecl) (Executor, error) {
			executor, ok := executors[decl.Name]
			if !ok {
				return nil, errors.New("missing executor")
			}
			return executor, nil
		}),
	)
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	return hooks
}

func cloneTelemetryRecord(src HookRunRecord) HookRunRecord {
	cloned := src
	cloned.PatchApplied = append(json.RawMessage(nil), src.PatchApplied...)
	return cloned
}

func stringPointer(value string) *string {
	return &value
}
