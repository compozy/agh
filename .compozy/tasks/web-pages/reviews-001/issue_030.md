---
status: pending
file: web/src/systems/workspace/hooks/use-workspaces.ts
line: 5
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:a9617d94fa4c
review_hash: a9617d94fa4c
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 030: Use @/* alias imports in this updated hook module.
## Review Comment

The new import should follow the project alias convention instead of relative paths.

As per coding guidelines, `web/src/**/*.{ts,tsx}`: "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `UNREVIEWED`
- Notes:
