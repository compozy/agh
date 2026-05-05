---
status: resolved
file: web/src/systems/session/components/session-resume-failure.tsx
line: 47
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:c309947d72f4
review_hash: c309947d72f4
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 023: Replace the ad-hoc typography values with shared tokens.
## Review Comment

`text-[13px]`, `text-[12px]`, `text-[11px]`, and `tracking-[0.08em]` introduce new type values inside `web/`. Please switch these to the shared typography/token utilities so this panel stays aligned with the design system.

As per coding guidelines, `web/src/**/*.{tsx,ts,css}`: Pull every color, font, radius, spacing step, and motion value from DESIGN.md in the repo root — never invent tokens.

## Triage

- Decision: `valid`
- Notes:
  - The resume failure panel introduces raw typography values (`text-[13px]`, `text-[12px]`, `text-[11px]`, `tracking-[0.08em]`) instead of the shared text scale and mono-tracking token defined for the web surface.
  - Root cause: local panel styling bypassed the established type utilities and `--tracking-mono`, making this component harder to keep aligned with the operator UI system.
  - Fix approach: convert the panel text to shared text utilities and use the mono tracking token for metadata, then lock the output down with the existing component test file as a justified scope exception.
  - Resolved: `session-resume-failure.tsx` now uses shared `text-sm` / `text-xs` utilities and `tracking-[var(--tracking-mono)]`, with matching assertions added to `session-resume-failure.test.tsx`.
  - Verified: focused Vitest session tests passed, then `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify` all completed successfully.
