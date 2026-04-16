---
status: resolved
file: internal/automation/resource.go
line: 43
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:d48f9da21de7
review_hash: d48f9da21de7
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 016: Wrap validation errors with resource-specific context.
## Review Comment

These paths currently return raw scope/binding/spec errors, which makes job-vs-trigger failures harder to diagnose once they bubble out of the codec. Wrap each failing step before returning it.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

Also applies to: 59-72

## Triage

- Decision: `VALID`
- Root cause: The referenced file was refactored; the equivalent logic now lives in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go#L1489) and [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go#L1529). `resolveConfigJob` and `resolveConfigTrigger` still return raw validation errors from `Validate(...)`, which loses the config resource context when those errors bubble out.
- Fix approach: Update the equivalent current implementation in `internal/automation/manager.go` to wrap job and trigger validation failures with resource-specific context, and document the path remap here because the reviewed filename no longer exists.

## Resolution

- Wrapped config job and trigger validation errors with resource-specific context in [internal/automation/manager.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager.go).
- Added regression coverage in [internal/automation/manager_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/automation/manager_test.go) to verify the wrapped errors.
