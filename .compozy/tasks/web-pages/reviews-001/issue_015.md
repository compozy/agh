---
status: resolved
file: web/src/routes/_app/bridges.tsx
line: 33
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:1950adb7eeb7
review_hash: 1950adb7eeb7
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 015: Import WorkspacePageShell through the workspace barrel.
## Review Comment

This route is reaching into another system’s internals. Please re-export the shell from `@/systems/workspace` and import it from there so the boundary stays stable.

As per coding guidelines, "Only import from cross-system dependencies through the public barrel export (`@/systems/<domain>`), never reach into another system's internals".

## Triage

- Decision: `valid`
- Root cause: the bridges route imports `WorkspacePageShell` from the workspace system's internal component path instead of its public barrel, which violates the system boundary rule.
- Fix approach: add a minimal barrel export in `web/src/systems/workspace/index.ts` and switch the scoped route import to `@/systems/workspace`. This requires one small out-of-scope barrel change to satisfy the public-API boundary.
