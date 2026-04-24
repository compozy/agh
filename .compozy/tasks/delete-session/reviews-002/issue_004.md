---
status: resolved
file: internal/session/manager_delete.go
line: 28
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RYXr,comment:PRRC_kwDOR5y4QM6622vO
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Treat a concurrent stop as benign here.**

This is a check-then-act race: `m.Get(target)` can succeed, the session can finish and disappear from runtime state, and `StopWithCause` can then return `ErrSessionNotFound`. In that case the persisted session history should still be deleted instead of aborting the whole operation.

<details>
<summary>Suggested hardening</summary>

```diff
 	if _, ok := m.Get(target); ok {
-		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil {
+		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil &&
+			!errors.Is(err, ErrSessionNotFound) {
 			return fmt.Errorf("session: stop %q before delete: %w", target, err)
 		}
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if _, ok := m.Get(target); ok {
		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil &&
			!errors.Is(err, ErrSessionNotFound) {
			return fmt.Errorf("session: stop %q before delete: %w", target, err)
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_delete.go` around lines 25 - 28, The current
check-then-act around m.Get(target) + m.StopWithCause can race: if the session
disappears between calls StopWithCause may return ErrSessionNotFound and abort
delete; treat that outcome as benign. Modify the code that calls
m.StopWithCause(ctx, target, CauseUserRequested, "session deleted") to continue
on ErrSessionNotFound (i.e., ignore that specific error) and only return on
other errors; reference m.Get, m.StopWithCause, and the ErrSessionNotFound
sentinel when implementing the conditional handling so the persisted session
history deletion proceeds even if the runtime session vanished concurrently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/session/manager_delete.go` does a `m.Get(target)` check and then calls `m.StopWithCause(...)`. If another goroutine finalizes and removes the session between those two calls, `StopWithCause` can return `ErrSessionNotFound` even though delete should still proceed against the persisted session directory.
  - Aborting the delete on that sentinel error is a real check-then-act race.
  - Planned fix: treat `ErrSessionNotFound` from the pre-delete stop path as benign and continue with artifact removal. A focused test helper is likely required outside the scoped file list to exercise the new error policy.

## Resolution

- Refactored the pre-delete stop logic in `internal/session/manager_delete.go` into a helper that treats `ErrSessionNotFound` from `StopWithCause` as benign, so persisted session deletion continues after a concurrent stop race.
- Added focused regression coverage in `internal/session/manager_delete_test.go` for the race where runtime state disappears between `Get` and `StopWithCause`.
- Verified with `make verify` (exit `0`).
