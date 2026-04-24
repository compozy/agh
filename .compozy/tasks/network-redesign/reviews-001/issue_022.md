---
status: resolved
file: web/src/systems/network/lib/network-formatters.test.ts
line: 99
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4166737115,nitpick_hash:0b044330b8ff
review_hash: 0b044330b8ff
source_review_id: "4166737115"
source_review_submitted_at: "2026-04-23T23:14:00Z"
---

# Issue 022: Consider expanding test coverage for formatNetworkKindLabel.
## Review Comment

The current test only validates one kind (`"capability"`). Consider adding tests for:
- Other valid kinds (`say`, `direct`, `trace`, etc.)
- Unknown/unrecognized kind strings (should return the original string per implementation)

## Triage

- Decision: `valid`
- Notes:
- `web/src/systems/network/lib/network-formatters.test.ts` only covers `formatNetworkKindLabel("capability")`, leaving the rest of the supported network kinds and the passthrough fallback behavior unprotected.
- The implementation in `network-formatters.ts` intentionally preserves unknown strings, so missing coverage here creates an easy regression gap for future label-map edits.
- Fix approach: extend the existing formatter test file with representative known kinds and an unknown-kind passthrough assertion.

## Resolution

- Expanded `network-formatters.test.ts` to cover every supported network kind label and the unknown-kind passthrough behavior.
- Verified with `bun x vitest run src/systems/network/lib/network-formatters.test.ts`, `make web-test`, and `make verify`.
