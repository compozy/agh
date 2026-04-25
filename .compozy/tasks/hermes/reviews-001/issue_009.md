---
status: resolved
file: internal/config/release_config_test.go
line: 11
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:79c023b2ee5a
review_hash: 79c023b2ee5a
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 009: Refactor into t.Run("Should...") subtests (table-driven where repeated).
## Review Comment

This test is currently monolithic; split assertions into subtests so failures are isolated and guideline-compliant.

As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

## Triage

- Decision: `VALID`
- Notes: `TestGoReleaserConfigPreservesTrustArtifactsAndPackageTargets` checks checksum, signing, SBOM, Homebrew cask, and nFPM package guarantees in a single body. Split the assertion groups into `t.Run("Should...")` subtests while keeping the same YAML fixture coverage.
