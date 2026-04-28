---
status: resolved
file: web/src/lib/pill-variant.ts
line: 17
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:5e61a9769644
review_hash: 5e61a9769644
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 008: Rename pillVariantFromTone to match its current semantics.
## Review Comment

The function now returns `PillTone`, so the `Variant` name is misleading and increases API confusion at call sites.

## Triage

- Decision: `valid`
- Root cause: `pillVariantFromTone` now returns `PillTone`, so `Variant` no longer matches the function's semantics and leaks outdated design-system vocabulary into call sites.
- Fix approach: hard-rename the function to `pillToneFromLegacyTone` and update all imports/call sites so typecheck does not rely on a legacy alias.
