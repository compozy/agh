---
status: resolved
file: internal/settings/classify.go
line: 80
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:7a9f31533e4d
review_hash: 7a9f31533e4d
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 022: Control flow between section-specific and collection field handling could be clearer.
## Review Comment

The `classifyField` function has two layers of field matching:
1. Section-specific handling (lines 86-103) that returns early for known sections
2. Collection prefix fallthrough (lines 105-120) for fields not matching section patterns

This means `hooks.` prefix is matched in two places (line 100 for `SectionHooksExtensions`, line 112 for fallthrough), but the section case returns first. The structure works correctly, but adding a brief comment explaining that lines 105-120 handle collection fields for sections without explicit field patterns would improve readability.

## Triage

- Decision: `invalid`
- Notes:
  - The review identifies a stylistic readability preference, not a behavioral defect or ambiguous branch outcome.
  - `classifyField` already makes the control flow explicit: section-specific handling happens first, then the collection-prefix fallback handles cross-section collection namespaces.
  - The duplicated `hooks.` prefix is intentional because `SectionHooksExtensions` accepts it directly while the later fallback preserves the same classification for collection fields outside that section-specific branch.
  - Adding a comment here would not change semantics, remove risk, or clarify an actually confusing implementation enough to justify code churn in this scoped review batch.
  - Verification: no code change required; repository gate `make verify` still passed after resolving the batch.
