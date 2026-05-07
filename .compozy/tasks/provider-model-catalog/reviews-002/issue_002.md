---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/acp/config_options_test.go
line: 85
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:7473cd1cb5da
review_hash: 7473cd1cb5da
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 002: Wrap these cases in t.Run("Should ...") subtests.
## Review Comment

The file switches back to flat top-level assertions here, which makes failures coarser than the repo’s test convention. Breaking the matching and legacy-state checks into named subtests will keep failures targeted and consistent with the rest of the suite.

As per coding guidelines: MUST use `t.Run("Should...")` pattern for ALL test cases.

## Triage

- Decision: `valid`
- Notes:
  - `internal/acp/config_options_test.go` still keeps `TestConfigOptionMatching` and `TestLegacyModelStateAllows` as flat top-level assertions under a single `t.Parallel()` body.
  - This violates the enforced AGH test convention requiring explicit `t.Run("Should ...")` subtests for each case.
  - Fix plan: split the assertions into named `Should ...` subtests without changing the underlying coverage.
  - Fixed in `internal/acp/config_options_test.go` and verified with focused package tests plus `make verify`.
