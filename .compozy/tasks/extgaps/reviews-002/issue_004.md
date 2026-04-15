---
status: resolved
file: internal/daemon/bridges_test.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110597069,nitpick_hash:a1861dbc0c22
review_hash: a1861dbc0c22
source_review_id: "4110597069"
source_review_submitted_at: "2026-04-15T03:35:44Z"
---

# Issue 004: Add explicit compile-time interface verification for bridgeRuntimeStoreStub.
## Review Comment

The test stub embeds `bridgeRuntimeStore`, which is an interface. Per coding guidelines ("Use compile-time interface verification: `var _ Interface = (*Type)(nil)`"), add an explicit assertion to catch silent drift if the interface changes:

```go
var _ bridgeRuntimeStore = (*bridgeRuntimeStoreStub)(nil)
```

Add this line after the struct definition (around line 32).

## Triage

- Decision: `invalid`
- Reasoning: adding `var _ bridgeRuntimeStore = (*bridgeRuntimeStoreStub)(nil)` to the current stub shape would be redundant and would not meaningfully improve drift detection, because the stub embeds the interface type directly. The assertion would continue to pass as long as the embedded field exists, even if the stub remained unsafe for unexpected calls.
- Why not fixing: the actual risk in this test helper is hidden success on unconfigured calls, which is addressed directly by issue `005`. A compile-time assertion on the current stub would not catch that failure mode.
- Resolution: no code change for the requested assertion. The real test-safety gap was addressed under issue `005`.
