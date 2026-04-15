---
status: resolved
file: web/src/systems/session/components/tool-call-card.tsx
line: 71
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:a590e114b2c7
review_hash: a590e114b2c7
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 005: Tooltip logic is Bash-specific; other tools with truncated summaries won't show tooltips.

## Review Comment

The tooltip for truncated content only triggers for `Bash` tools. Other tools (like `Read`, `Write`, `Grep`) also truncate at 60 characters via `getToolCompactSummary` but won't display a tooltip with the full value. This may be intentional given Bash commands are typically longer, but worth confirming.

Also applies to: 165-183

## Triage

- Decision: `valid`
- Root cause: `ToolCallCard` only computes tooltip content for Bash commands even though `getToolCompactSummary` truncates Read/Write/Edit paths and Grep/Glob/Web tool inputs too; for those tools, the collapsed card can hide the full summary with no way to inspect it inline.
- Fix approach: Generalize the tooltip decision to any tool whose raw summary is longer than the compact summary and show the full untruncated value in the tooltip content.
- Resolution: Added raw summary extraction in `tool-labels.ts`, reused it for truncation, and updated `ToolCallCard` so any truncated supported tool summary gets the full tooltip content. Added a regression test for a long Read path.
