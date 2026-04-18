---
status: resolved
file: magefile_test.go
line: 56
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133244010,nitpick_hash:4c0919e7ca32
review_hash: 4c0919e7ca32
source_review_id: "4133244010"
source_review_submitted_at: "2026-04-18T02:06:20Z"
---

# Issue 004: Prefer a table-driven shape for these env variants.
## Review Comment

These two subtests are just input/output variations of the same behavior. A small table would remove the duplicated setup/assertion flow and make future env cases cheaper to add.

As per coding guidelines, Use table-driven tests with subtests (`t.Run`) as default.

## Triage

- Decision: `valid`
- Notes:
  - `TestWithRaceEnabledEnv` currently uses two subtests that differ only by input and expected values.
  - The repo guidelines require table-driven tests with subtests by default, and this case is a straightforward fit.
  - Fix approach: collapse the env-variant coverage into a single table-driven subtest structure while preserving the mutation and nil-input assertions.
  - Resolved by rewriting `TestWithRaceEnabledEnv` into a table-driven test while keeping the mutation and nil-input coverage intact.
  - Verified with `go test -tags mage . -run 'TestWithRaceEnabledEnv|TestRunRaceEnabledGoCommand'` and the full `make verify` gate.
