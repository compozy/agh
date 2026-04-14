---
status: resolved
file: internal/observe/health.go
line: 49
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107556463,nitpick_hash:48134b88e4d0
review_hash: 48134b88e4d0
source_review_id: "4107556463"
source_review_submitted_at: "2026-04-14T16:23:06Z"
---

# Issue 017: Consider graceful degradation instead of failing the full health snapshot.
## Review Comment

`collectTaskHealth` now gates the entire `Health(ctx)` response. Since task health aggregation depends on multiple queries, transient task-store failures can make the whole health endpoint fail. Consider returning core health with a degraded task status (plus telemetry) rather than hard-failing the full snapshot.

## Triage

- Decision: `invalid`
- Notes:
  This comment proposes a different health-endpoint contract rather than identifying a correctness bug in the current implementation. The current behavior is intentional and already codified by `TestObserverHealthWrapsTaskHealthErrors`, which expects task-health failures to bubble out of `Health()`.
  Changing `Health()` to degrade gracefully would require a broader API/observability contract decision and corresponding test updates, not a localized fix in this review batch.

## Resolution

- No code change was made. This is an API-contract/design suggestion, and the current fail-fast `Health()` behavior remains intentional and covered by existing tests.
