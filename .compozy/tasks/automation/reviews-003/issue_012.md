---
status: resolved
file: internal/automation/trigger.go
line: 390
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TkHB,comment:PRRC_kwDOR5y4QM624LnI
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Add an early nil-context guard in `FireHookCompletion`.**

`hookCompletionEnvelope` uses `ctx` immediately via `HookSessionResolver.Status`, so nil context can leak before `Fire` performs its own guard.

<details>
<summary>Suggested patch</summary>

```diff
 func (e *TriggerEngine) FireHookCompletion(ctx context.Context, sessionID string, record hookspkg.HookRunRecord) (TriggerResult, error) {
+	if ctx == nil {
+		return TriggerResult{}, errors.New("automation: trigger fire context is required")
+	}
 	envelope, err := e.hookCompletionEnvelope(ctx, sessionID, record)
 	if err != nil {
 		return TriggerResult{}, err
 	}
 	return e.Fire(ctx, envelope)
 }
```
</details>

As per coding guidelines, "Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside `main` and focused tests".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (e *TriggerEngine) FireHookCompletion(ctx context.Context, sessionID string, record hookspkg.HookRunRecord) (TriggerResult, error) {
	if ctx == nil {
		return TriggerResult{}, errors.New("automation: trigger fire context is required")
	}
	envelope, err := e.hookCompletionEnvelope(ctx, sessionID, record)
	if err != nil {
		return TriggerResult{}, err
	}
	return e.Fire(ctx, envelope)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/trigger.go` around lines 384 - 390, The
FireHookCompletion method should validate that ctx is non-nil before calling
hookCompletionEnvelope to prevent a nil-context from reaching
HookSessionResolver.Status; modify TriggerEngine.FireHookCompletion to check if
ctx == nil and return a clear error (e.g., fmt.Errorf("nil context") or a
package-specific error) before invoking hookCompletionEnvelope or Fire so the
early guard protects downstream calls.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `FireHookCompletion()` currently calls `hookCompletionEnvelope()` before any nil-context guard, and that helper immediately forwards the context to `HookSessionResolver.Status(...)`.
- A nil context should be rejected at the public API boundary before any downstream resolver call, consistent with the existing `Fire()` guard.
- Fix plan: add an early nil-context check in `FireHookCompletion()` and cover it with a regression that proves the hook session resolver is not invoked.
