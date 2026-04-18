---
status: resolved
file: internal/observe/tasks_test.go
line: 392
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:51a93153b2f4
review_hash: 51a93153b2f4
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 022: Convert these observer scenarios into Should... subtests.
## Review Comment

The new dashboard/inbox coverage is solid, but it’s still organized as large standalone tests with many assertions per function. Recasting the scenarios as `t.Run("Should...")` cases would make the suite easier to maintain and much quicker to debug when one branch regresses.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `valid`
- Root cause: the new observer dashboard/inbox unit coverage is organized as broad top-level tests instead of repo-standard `t.Run("Should...")` subtests, which reduces failure locality.
- Fix approach: restructure the added observer scenarios into explicit `Should...` subtests without weakening the assertions.

## Resolution

- Wrapped the observer dashboard and inbox scenarios in `t.Run("Should ...")` subtests in `internal/observe/tasks_test.go`.
- Renamed the existing backlog/empty snapshot subtests to the same `Should ...` form for consistency.
- Verification: `go test ./internal/observe` and `go test -tags integration ./internal/observe`
