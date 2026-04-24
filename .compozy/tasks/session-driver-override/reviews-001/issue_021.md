---
status: resolved
file: web/src/systems/session/components/session-create-dialog.tsx
line: 75
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:7459824f76de
review_hash: 7459824f76de
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 021: Replace arbitrary sizing with shared design tokens.
## Review Comment

The new dialog introduces raw values like `sm:max-w-[30rem]` and repeated `text-[12px]`, which bypass the shared design scale used elsewhere in `web/`. Please move these to token-backed utilities or CSS vars.

As per coding guidelines, `Pull every color, font, radius, spacing step, and motion value from DESIGN.md — never invent tokens`.

Also applies to: 115-115, 154-154, 163-163, 173-173

## Triage

- Decision: `valid`
- Notes:
  - The dialog currently hardcodes `sm:max-w-[30rem]` and multiple `text-[12px]` helpers in a component governed by `DESIGN.md` and `web/AGENTS.md`, which require token-backed scale values instead of ad-hoc measurements.
  - Root cause: the new dialog copied one-off sizing values instead of reusing the shared width and typography utilities already used across `web/` and `@agh/ui`.
  - Fix approach: replace the arbitrary width with a shared max-width utility and swap the repeated `text-[12px]` usages to the shared text scale so the dialog stays on the design-system path.
  - Resolved: `session-create-dialog.tsx` now uses `sm:max-w-lg` and shared `text-xs` utilities for the helper and alert copy, with class-level regression coverage in `session-create-dialog.test.tsx`.
  - Verified: focused Vitest session tests passed, then `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify` all completed successfully.
