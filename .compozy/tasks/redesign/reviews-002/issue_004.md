---
status: resolved
file: packages/ui/scripts/serve-storybook.ts
line: 9
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:8bd45eb3e694
review_hash: 8bd45eb3e694
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 004: Also verify the bundle root is a directory.
## Review Comment

Lines 9-13 only check existence. If `root` points to a file, the server starts but cannot serve expected paths.

## Triage

- Decision: `valid`
- Reasoning: The current startup guard only checks that the Storybook root exists. If the path resolves to a file instead of a directory, the server still starts and later fails to serve the bundle as expected.
- Root cause: Startup validation does not confirm that the bundle root is a directory.
- Fix plan: Reuse `statSync(root)` during startup validation and exit early if the resolved root is not a directory.

## Resolution

- Strengthened startup validation in `packages/ui/scripts/serve-storybook.ts` to reject non-directory bundle roots before serving.
- Verified with `make verify` after all batch changes.
