---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 678
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Odc3,comment:PRRC_kwDOR5y4QM64G4Y2
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't swallow readiness-report failures.**

If `reportState` fails here, the instance can stay stale while the surrounding delivery/ingest path still clears `lastError` as if everything succeeded.

<details>
<summary>Suggested fix</summary>

```diff
 func (p *gchatProvider) reportReadyIfNeeded(ctx context.Context, session *bridgesdk.Session, bridgeInstanceID string) {
 	p.mu.RLock()
 	status := p.reportedStatus[strings.TrimSpace(bridgeInstanceID)]
 	p.mu.RUnlock()
 	if status == bridgepkg.BridgeStatusReady {
 		return
 	}
-	_, _ = p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil)
+	if _, err := p.reportState(ctx, session, bridgeInstanceID, bridgepkg.BridgeStatusReady, nil); err != nil {
+		p.setLastError(err)
+	}
 }
```

</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 670 - 678, In
reportReadyIfNeeded, don't ignore the error returned by reportState; capture its
error and handle it instead of using the blank identifier. Call reportState from
reportReadyIfNeeded and if it returns an error, log the failure (using the
provider/session logger), do not update reportedStatus to BridgeStatusReady, and
return early so the instance doesn't become stale; only set
reportedStatus[p.trimmed bridgeInstanceID] = bridgepkg.BridgeStatusReady (under
p.mu) after a successful reportState call. Ensure you remove the "_ = " usage
and handle the error explicitly in reportReadyIfNeeded.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `reportReadyIfNeeded` currently ignores `reportState` failures with `_, _ = ...`, and `handleBridgesDeliver` immediately clears `lastError` afterward.
  - Root cause: readiness reporting errors are silently discarded, so a failed state report can be masked by an otherwise successful delivery or ingest path.
  - Outcome: made `reportReadyIfNeeded` return an error, preserved `lastError` when readiness reporting fails, and added focused coverage in `extensions/bridges/gchat/provider_test.go` for the failure path. That extra test file was outside the listed batch code files but was required to validate the production fix. Verified with `go test ./extensions/bridges/discord ./extensions/bridges/gchat` and `make verify`.
