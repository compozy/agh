---
status: resolved
file: web/src/systems/bridges/components/bridge-detail-panel.tsx
line: 375
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:3281cce86b36
review_hash: 3281cce86b36
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 023: Align the session metadata line with the metadata type rule.
## Review Comment

The new `session` line is metadata; style the label with uppercase + tracking to match the design system.

As per coding guidelines, `web/src/**/*.{tsx,css}` requires **JetBrains Mono for metadata with uppercase and tracking 0.06em+**.

## Triage

- Decision: `valid`
- Root cause: the route session metadata line renders a lowercase `session` label with no tracking. AGH metadata labels must use JetBrains Mono, uppercase text, and letter spacing of at least `0.06em`.
- Fix approach: split the metadata label from the session ID, style only the label as uppercase tracked mono text, and preserve the session ID casing exactly.
- Resolution: the route metadata line now renders a tracked uppercase `SESSION` label while keeping the session ID unchanged. The bridge component test asserts the label styling, and full `make verify` passed after the code change.
