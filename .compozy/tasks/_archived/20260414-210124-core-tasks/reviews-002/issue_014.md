---
status: resolved
file: internal/extension/host_api_test.go
line: 2785
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lga,comment:PRRC_kwDOR5y4QM63o2QY
---

# Issue 014: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap executor errors with operation context.**

The new task session executor returns raw errors from session operations, which makes failures harder to diagnose and violates the explicit wrapping rule.

<details>
<summary>🛠️ Suggested fix</summary>

```diff
 	created, err := e.sessions.Create(ctx, opts)
 	if err != nil {
-		return nil, err
+		return nil, fmt.Errorf("start task session: create session: %w", err)
 	}
@@
 	info, err := e.sessions.Status(ctx, strings.TrimSpace(sessionID))
 	if err != nil {
-		return nil, err
+		return nil, fmt.Errorf("attach task session: read session status: %w", err)
 	}
@@
 func (e *hostAPITestTaskSessionExecutor) RequestTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
-	return e.sessions.RequestStopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation")
+	if ctx == nil {
+		return errors.New("extension: host api test task request stop context is required")
+	}
+	if err := e.sessions.RequestStopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation"); err != nil {
+		return fmt.Errorf("request task stop: %w", err)
+	}
+	return nil
 }
 
 func (e *hostAPITestTaskSessionExecutor) ForceTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
-	return e.sessions.StopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation")
+	if ctx == nil {
+		return errors.New("extension: host api test task force stop context is required")
+	}
+	if err := e.sessions.StopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation"); err != nil {
+		return fmt.Errorf("force task stop: %w", err)
+	}
+	return nil
 }
```
</details>

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	created, err := e.sessions.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("start task session: create session: %w", err)
	}
	info := created.Info()
	if info == nil {
		return nil, fmt.Errorf("%w: task session create returned nil session info", taskpkg.ErrValidation)
	}
	return &taskpkg.SessionRef{
		SessionID:   info.ID,
		WorkspaceID: info.WorkspaceID,
		StartedAt:   info.CreatedAt,
	}, nil
}

func (e *hostAPITestTaskSessionExecutor) AttachTaskSession(ctx context.Context, _ string, sessionID string) (*taskpkg.SessionRef, error) {
	if ctx == nil {
		return nil, errors.New("extension: host api test task attach context is required")
	}

	info, err := e.sessions.Status(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, fmt.Errorf("attach task session: read session status: %w", err)
	}
	if info == nil || info.State != session.StateActive {
		return nil, fmt.Errorf("%w: session %q is not active", taskpkg.ErrSessionAttachNotAllowed, strings.TrimSpace(sessionID))
	}
	return &taskpkg.SessionRef{
		SessionID:   info.ID,
		WorkspaceID: info.WorkspaceID,
		StartedAt:   info.CreatedAt,
	}, nil
}

func (e *hostAPITestTaskSessionExecutor) RequestTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
	if ctx == nil {
		return errors.New("extension: host api test task request stop context is required")
	}
	if err := e.sessions.RequestStopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation"); err != nil {
		return fmt.Errorf("request task stop: %w", err)
	}
	return nil
}

func (e *hostAPITestTaskSessionExecutor) ForceTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
	if ctx == nil {
		return errors.New("extension: host api test task force stop context is required")
	}
	if err := e.sessions.StopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation"); err != nil {
		return fmt.Errorf("force task stop: %w", err)
	}
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 2745 - 2785, The executor
currently returns raw errors from session operations; wrap each returned error
with contextual text using fmt.Errorf so failures are identifiable: when
creating sessions wrap the error from e.sessions.Create (in the Create flow that
assigns created, err), when attaching wrap the error from e.sessions.Status in
AttachTaskSession, and when stopping wrap the errors returned by
e.sessions.RequestStopWithCause in RequestTaskStop and e.sessions.StopWithCause
in ForceTaskStop — use messages like "create session: %w", "attach session
<sessionID>: %w", "request stop <sessionID>: %w" and "force stop <sessionID>:
%w" (trim sessionID where appropriate) to preserve the original error while
adding operation context.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The host-api task session executor in the test harness returns raw errors from `sessions.Create`, `sessions.Status`, `RequestStopWithCause`, and `StopWithCause`, which strips operation context from failures.
  Root cause: these helper methods pass session-layer errors through directly instead of wrapping them at the boundary.
  Planned fix: wrap each returned error with operation-specific context while preserving the original error via `%w`.

## Resolution

- Wrapped the task-session executor helper errors in `internal/extension/host_api_test.go` with operation-specific context and added nil-context guards for the stop helpers.
- This keeps failures diagnosable while preserving the underlying errors for assertions.
