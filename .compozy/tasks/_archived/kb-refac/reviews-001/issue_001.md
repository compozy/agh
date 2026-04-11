---
status: resolved
file: internal/acp/handlers.go
line: 105
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4090986708,nitpick_hash:8b3e741fa573
review_hash: 8b3e741fa573
source_review_id: "4090986708"
source_review_submitted_at: "2026-04-10T16:14:03Z"
---

# Issue 001: Consider hoisting the handler map to a struct field for reduced allocations.
## Review Comment

The `handlers` map is recreated on every `handleInbound` call. While the map size is small (9 entries), moving it to a field initialized once during `AgentProcess` construction would eliminate per-call allocations in a potentially hot path.

## Triage

- Decision: `INVALID`
- Reasoning: `handleInbound` recreates a 9-entry dispatch map per call, but the review is a speculative allocation micro-optimization rather than a correctness defect. There is no benchmark or observed hot-path regression in this batch, and hoisting the method map into `AgentProcess` would add lifecycle/state surface without fixing broken behavior.
- Fix approach: No code change. Keep the local dispatch table until there is measured evidence that this path needs optimization.
