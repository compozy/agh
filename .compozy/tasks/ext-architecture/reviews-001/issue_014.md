---
status: resolved
file: internal/codegen/sdkts/generate.go
line: 553
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:adbc5cbab312
review_hash: adbc5cbab312
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 014: Hardcoded package path prefix may break if the module is forked or renamed.
## Review Comment

The check `strings.HasPrefix(t.PkgPath(), "github.com/pedronauck/agh/internal/")` couples the generator to a specific module path.

Consider extracting this as a configurable parameter or deriving it from the module's `go.mod` if the codebase is expected to be forked.

## Triage

- Decision: `invalid`
- Notes: The generator is intentionally scoped to this repository's internal packages, and the hardcoded module prefix is part of that contract. Supporting arbitrary forks or renamed module paths would add extra discovery/configuration complexity for a scenario this greenfield project does not support today. This is portability feedback, not a correctness bug in the current codebase.
