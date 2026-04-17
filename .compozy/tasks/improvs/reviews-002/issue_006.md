---
status: resolved
file: internal/hooks/async_clone.go
line: 263
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130784468,nitpick_hash:cb842accb7e1
review_hash: cb842accb7e1
source_review_id: "4130784468"
source_review_submitted_at: "2026-04-17T17:23:10Z"
---

# Issue 006: Reflection fallback for unknown container types.
## Review Comment

The reflection-based `cloneDynamicContainer` call at line 278 handles edge cases where `any`-typed values contain nested containers of unknown types. This is appropriate for correctness when cloning arbitrary data structures, though per coding guidelines, consider adding a brief comment explaining why reflection is needed here.

As per coding guidelines: "Never use reflection without performance justification" — while this usage is for correctness rather than performance, documenting the rationale would satisfy the spirit of this guideline.

## Triage

- Decision: `invalid`
- Reasoning: this is a documentation preference, not a correctness, test, or maintainability defect that warrants code churn in the scoped batch. The reflection fallback is already isolated behind concrete fast paths and a purpose-specific helper (`cloneDynamicContainer`), so adding a comment here would be narration rather than a root-cause fix under the local "comments only when needed" guidance.
- Resolution: no code change required.
- Verification: `make verify` passed on 2026-04-17 after the rest of the scoped batch changes.
