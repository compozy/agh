---
status: resolved
file: internal/extension/manager.go
line: 1539
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:580bbc238cd2
review_hash: 580bbc238cd2
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 026: Get() is returning live mutable state, not an isolated snapshot.
## Review Comment

`manifest := *ext.manifest` is only a shallow copy, so nested maps/slices/pointers are still shared, and `clone.Skills = append(clone.Skills, ext.skills...)` reuses the original skill pointers. A caller mutating the returned value can corrupt manager state or race with the supervisor.

## Triage

- Decision: `valid`
- Notes:
  `cloneExtension` only shallow-copies `ext.manifest` and reuses the original skill pointers. Callers mutating the returned snapshot can therefore mutate manager-owned state.
  Fix approach: deep-clone manifest data and skills before returning the snapshot from `Get`.
  Additional test scope needed: `internal/extension/manager_test.go` is outside the batch file list but is the minimal place to prove snapshot isolation.
