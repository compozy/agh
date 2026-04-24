---
status: resolved
file: internal/session/manager_delete.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6c,comment:PRRC_kwDOR5y4QM66Aomx
---

# Issue 008: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap propagated errors with delete-specific context.**

Line 22 and Line 27 return raw errors, which makes operational diagnosis harder in call chains.


<details>
<summary>Proposed fix</summary>

```diff
 	target, err := normalizeStoredSessionID(id)
 	if err != nil {
-		return err
+		return fmt.Errorf("session: normalize delete id %q: %w", id, err)
 	}

 	if _, ok := m.Get(target); ok {
 		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil {
-			return err
+			return fmt.Errorf("session: stop %q before delete: %w", target, err)
 		}
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
	target, err := normalizeStoredSessionID(id)
	if err != nil {
		return fmt.Errorf("session: normalize delete id %q: %w", id, err)
	}

	if _, ok := m.Get(target); ok {
		if err := m.StopWithCause(ctx, target, CauseUserRequested, "session deleted"); err != nil {
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

In `@internal/session/manager_delete.go` around lines 21 - 27, The returns are
propagating raw errors; update the error returns to wrap them with
delete-specific context using fmt.Errorf so callers see cause and operation
(e.g., when checking m.Get(target) and when calling m.StopWithCause(ctx, target,
CauseUserRequested, "session deleted") wrap the err values as fmt.Errorf("delete
session %s: %w", target, err) or similar to add clear delete-specific context
while preserving the original error.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `Manager.Delete` in the session package still returns raw errors from ID normalization and `StopWithCause`, so callers lose delete-specific context in logs and error chains. I will wrap those error returns with operation-specific context while preserving the original sentinels via `%w`.
