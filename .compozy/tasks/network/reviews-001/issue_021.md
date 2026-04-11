---
status: resolved
file: internal/network/manager.go
line: 313
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZj,comment:PRRC_kwDOR5y4QM623eZ0
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Handle direct-subscription rollback errors.**

If heartbeat startup fails and `Unsubscribe` also fails, this path currently returns only the heartbeat error and can leave a live direct subscription behind for a session that never finished joining.

<details>
<summary>🧹 Suggested fix</summary>

```diff
 	heartbeat, err := m.router.StartHeartbeat(m.lifecycleCtx, local.SessionID, "")
 	if err != nil {
-		_ = directSub.Unsubscribe()
+		if unsubErr := directSub.Unsubscribe(); unsubErr != nil && !errors.Is(unsubErr, nats.ErrConnectionClosed) {
+			err = errors.Join(err, fmt.Errorf("network: unsubscribe direct subject for %q: %w", local.SessionID, unsubErr))
+		}
 		if releaseErr := m.releaseBroadcastSubscription(local.Space); releaseErr != nil {
 			err = errors.Join(err, releaseErr)
 		}
 		m.router.Leave(local.SessionID)
 		return err
```
</details>

As per coding guidelines, "Never ignore errors with _ — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/manager.go` around lines 306 - 313, The heartbeat startup
error path currently ignores any error from directSub.Unsubscribe and returns
only the StartHeartbeat error; update the error handling so that when
StartHeartbeat fails you capture the error from directSub.Unsubscribe and any
error from m.releaseBroadcastSubscription(local.Space), join them with the
original err (e.g., using errors.Join or similar) before returning, while still
calling m.router.Leave(local.SessionID); specifically modify the block around
m.router.StartHeartbeat, directSub.Unsubscribe, m.releaseBroadcastSubscription,
and m.router.Leave to aggregate all failures (heartbeat, unsubscribe, release)
into the returned error so no rollback error is ignored.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The heartbeat-start rollback path in `JoinSpace` currently ignores `directSub.Unsubscribe()` with `_ = ...`. If `StartHeartbeat` fails and the unsubscribe also fails, the method returns only the heartbeat/release errors and can leave a live direct subscription behind for a session that never finished joining. The fix is to capture the unsubscribe failure, ignore only benign connection-closed shutdowns, and join it into the returned rollback error.
  Resolved by aggregating direct-subscription cleanup failures in `internal/network/manager.go` and adding helper coverage in `internal/network/manager_test.go`. Verified with package tests and a clean `make verify`.
