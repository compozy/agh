---
status: resolved
file: internal/session/manager_prompt.go
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745bB,comment:PRRC_kwDOR5y4QM65BAQV
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Propagate cancellation/deadline errors from the augmenter.**

If `m.inputAugmenter(...)` returns `context.Canceled` or `context.DeadlineExceeded`, this path still falls back to the original message and can continue into `m.driver.Prompt(...)`. That weakens cancellation semantics and can dispatch a prompt after the caller has already aborted the request.

<details>
<summary>Suggested fix</summary>

```diff
 	dispatchMessage := message
 	if m.inputAugmenter != nil {
 		augmented, augmentErr := m.inputAugmenter(ctx, session, message)
 		if augmentErr != nil {
+			if errors.Is(augmentErr, context.Canceled) || errors.Is(augmentErr, context.DeadlineExceeded) {
+				return nil, fmt.Errorf("session: augment prompt input: %w", augmentErr)
+			}
 			m.sessionLogger(session).Warn("session: prompt input augmentation failed", "error", augmentErr)
 		} else if strings.TrimSpace(augmented) != "" {
 			dispatchMessage = augmented
 		}
 	}
```
</details>


As per coding guidelines, Use `context.Context` as first argument to functions crossing runtime boundaries — avoid `context.Background()` outside `main` and focused tests.

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Manager.Prompt` currently swallows every augmenter error and still calls `m.driver.Prompt(...)`. That is wrong for `context.Canceled` and `context.DeadlineExceeded`, because those errors mean the caller already aborted the request.
  - I will preserve best-effort behavior for ordinary augmentation failures while returning cancellation/deadline errors before dispatch.

## Resolution

- `Manager.Prompt` now propagates `context.Canceled` and `context.DeadlineExceeded` from the input augmenter instead of falling through to driver dispatch.
- Added a session-level regression test confirming canceled augmentation skips `driver.Prompt(...)`.
