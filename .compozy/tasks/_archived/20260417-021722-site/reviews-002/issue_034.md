---
status: resolved
file: internal/daemon/automation_resources.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:f1fd9bf84f21
review_hash: f1fd9bf84f21
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 034: Avoid any(...) for this type assertion
## Review Comment

You can assert directly from `runtime` without widening to `any`.

As per coding guidelines: `Never use interface{}/any when a concrete type is known`.

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed file has been consolidated, but the same concern survives at `internal/daemon/boot.go:685`.
  - Root cause: the code widens `state.deps.Extensions` through `any(...)` before a concrete type assertion, which is unnecessary because the source value is already an interface.
  - Intended fix: use a direct type assertion from the interface value in `boot.go`.
  - Result: replaced the unnecessary `any(...)` hop with a direct type assertion in `internal/daemon/boot.go`; verified by package compile and `make verify`.
