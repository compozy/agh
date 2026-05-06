---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/extension/manifest_test.go
line: 110
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232600854,nitpick_hash:9b0b8c62467d
review_hash: 9b0b8c62467d
source_review_id: "4232600854"
source_review_submitted_at: "2026-05-06T01:29:19Z"
---

# Issue 006: Wrap this new parser scenario in a named subtest.
## Review Comment

This new case should follow the repo’s `t.Run("Should ...")` convention for consistent failure reporting. Keep it serial here, since `withDaemonVersion` mutates process-wide test state.

As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures".

## Triage

- Decision: `valid`
- Notes:
  - `TestLoadManifestParsesNetworkHookMatcher` is a flat parser scenario in a file that otherwise groups behavior cases under named subtests.
  - `withDaemonVersion(...)` installs process-wide cleanup-backed version overrides, so this case should stay serial even after moving under `t.Run`.
  - Fix plan: wrap the scenario in `t.Run("Should parse network hook matcher", ...)` without `t.Parallel()`.

## Resolution

- Wrapped the parser scenario in `t.Run("Should parse network hook matcher", ...)` and kept it serial because `withDaemonVersion(...)` mutates process-wide test state.
- Verified with fresh full `make verify` (passed).
