---
status: resolved
file: internal/network/lifecycle.go
line: 186
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4094005443,nitpick_hash:4ee3fce86572
review_hash: 4ee3fce86572
source_review_id: "4094005443"
source_review_submitted_at: "2026-04-11T17:12:00Z"
---

# Issue 013: Harden participant invariants for direct/recipe envelopes.
## Review Comment

For existing interactions, direct/recipe handling validates `env.From` but does not enforce that `env.To` is present and belongs to the same participant pair. Adding this check prevents out-of-pair routing from mutating lifecycle state.

## Triage

- Decision: `valid`
- Root cause: Existing-interaction handling only checks that `env.From` is one of the two participants; it does not require `env.To` to exist and match the opposite participant, so an out-of-pair direct or recipe envelope can mutate lifecycle state.
- Fix plan: Enforce directed participant-pair invariants for direct and recipe envelopes and add regression coverage for missing or mismatched targets.
