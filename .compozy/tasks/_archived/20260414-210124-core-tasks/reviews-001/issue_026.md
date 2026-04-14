---
status: resolved
file: internal/observe/health.go
line: 52
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aot,comment:PRRC_kwDOR5y4QM63mgSG
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap task health errors with operation context.**

At Line 51, returning raw `err` drops useful context in health failures.

<details>
<summary>🛠️ Proposed fix</summary>

```diff
 	taskHealth, err := o.collectTaskHealth(ctx)
 	if err != nil {
-		return Health{}, err
+		return Health{}, fmt.Errorf("observe: collect task health: %w", err)
 	}
```
</details>


As per coding guidelines, Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	taskHealth, err := o.collectTaskHealth(ctx)
	if err != nil {
		return Health{}, fmt.Errorf("observe: collect task health: %w", err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/health.go` around lines 49 - 52, The call to
collectTaskHealth in observe.health.go returns an unwrapped error which loses
context; update the error return in the observe.collectTaskHealth call inside
function Get/Collect/whatever the surrounding function is (the one that
currently assigns taskHealth, err := o.collectTaskHealth(ctx)) to wrap the error
with contextual text using fmt.Errorf, e.g. return Health{},
fmt.Errorf("collecting task health: %w", err), so callers see the operation that
failed; ensure you import fmt if not already present.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `Observer.Health` wraps neighboring operations with contextual `fmt.Errorf(...)` messages, but the `collectTaskHealth` failure path returns the raw error. That drops operation context and makes health failures harder to trace.
  I will wrap the task-health failure with an `observe: ...` message so the exported health surface is consistent with the rest of the method.
  Resolution: Wrapped the `collectTaskHealth` failure with `observe: collect task health: ...` and added a regression test that forces a task-health query failure and checks the wrapped error text.
