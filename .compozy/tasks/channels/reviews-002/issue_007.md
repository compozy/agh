---
status: resolved
file: internal/channels/registry.go
line: 392
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857889,nitpick_hash:25d2c070d527
review_hash: 25d2c070d527
source_review_id: "4093857889"
source_review_submitted_at: "2026-04-11T14:16:28Z"
---

# Issue 007: ListRoutes returns routes without cloning, unlike ListInstances.
## Review Comment

`ListInstances` deep-clones each returned instance (lines 196-200), but `ListRoutes` returns the store's slice directly. If `ChannelRoute` contains mutable fields (e.g., JSON data, metadata maps), callers could mutate shared state.

Additionally, verify whether `cloneChannelRoute` needs to deep-copy any mutable fields (similar to how `cloneChannelInstance` copies `DeliveryDefaults`).

## Triage

- Decision: `Valid`
- Notes:
  `ListRoutes` returns the store slice directly, unlike `ListInstances`. Even though `ChannelRoute` only contains value fields today, mutating the returned slice elements can still alias an in-memory store result if the store reuses its backing array. The fix is to allocate a new slice and copy each route via `cloneChannelRoute`; no deeper clone is needed because `ChannelRoute` currently has no reference-typed fields. A minimal unit-test update outside the primary six code files will likely be needed to pin this aliasing behavior.
  Resolved by cloning `ListRoutes` results in `internal/channels/registry.go` and adding the minimal supporting regression test in `internal/channels/registry_test.go`, then verified with `go test ./internal/channels -count=1` and `make verify`.
