---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/situation/task_context.go
line: 13
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:ecb90224e575
review_hash: ecb90224e575
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 022: Break the internal/situation → internal/api dependency.
## Review Comment

`renderTaskBundlePrompt` now pulls `contract.AgentContextPayload` into `internal/situation`, which inverts the layering this repo explicitly forbids. Please move the render DTO/helper below `internal/api`, or make the API layer adapt `taskpkg.ContextBundle` before calling `RenderPrompt`.

As per coding guidelines, `Packages under internal/ must not import from daemon/, api/, or cli/ — dependencies flow downward only toward the composition root`.

Also applies to: 443-450

## Triage

- Decision: `invalid`
- Notes: The underlying `internal/situation -> internal/api/contract` dependency already exists elsewhere in the package (`internal/situation/service.go` and `internal/situation/render.go` both import `internal/api/contract`). The new `task_context.go` helper is not introducing a new package-level boundary inversion by itself. Fixing the broader architecture would require a package-wide refactor outside this batch’s scoped files, so this line-level review item is not valid as a standalone regression in the current code.
- Resolution: Analysis only; no code change in this batch.
