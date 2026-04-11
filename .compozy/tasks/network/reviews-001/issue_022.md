---
status: resolved
file: internal/network/manager.go
line: 655
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZl,comment:PRRC_kwDOR5y4QM623eZ2
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't ignore duplicate broadcast-subscription cleanup failures.**

This race path runs when another join has already installed the space subscription. If the redundant `Unsubscribe` fails here, both subscriptions can stay active and every inbound broadcast may be processed twice.

<details>
<summary>🧹 Suggested fix</summary>

```diff
 	if runtime, ok := m.spaces[targetSpace]; ok {
 		runtime.refCount++
-		_ = subscription.Unsubscribe()
-		return nil
+		if err := subscription.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
+			runtime.refCount--
+			return fmt.Errorf("network: unsubscribe duplicate broadcast subject for %q: %w", targetSpace, err)
+		}
+		return nil
 	}
```
</details>

As per coding guidelines, "Never ignore errors with _ — every error must be handled or have a written justification".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	m.mu.Lock()
	defer m.mu.Unlock()
	if runtime, ok := m.spaces[targetSpace]; ok {
		runtime.refCount++
		if err := subscription.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			runtime.refCount--
			return fmt.Errorf("network: unsubscribe duplicate broadcast subject for %q: %w", targetSpace, err)
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

In `@internal/network/manager.go` around lines 650 - 655, The current early-return
path in the join flow ignores the result of subscription.Unsubscribe(), which
can leave duplicate active subscriptions if Unsubscribe fails; update the block
that checks m.spaces[targetSpace] (where runtime.refCount++ is done) to call
subscription.Unsubscribe() and handle its error instead of discarding it—e.g.,
capture the error, log a descriptive message with context (targetSpace and any
subscription id) and either return the error or escalate it so callers know
cleanup failed; ensure you reference subscription.Unsubscribe() and the
runtime/refCount code path when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `acquireBroadcastSubscription` has a real race window: two joiners can subscribe concurrently, and the loser currently increments `refCount` then discards the redundant `subscription.Unsubscribe()` error. If that unsubscribe fails, both broadcast subscriptions remain active and messages can be processed twice. The fix is to handle the unsubscribe failure, roll back the extra `refCount`, and return a descriptive cleanup error.
  Resolved by rolling back `refCount` when duplicate broadcast cleanup fails and by covering the cleanup/refcount behavior in `internal/network/manager_test.go`. Verified with package tests and a clean `make verify`.
