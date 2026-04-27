---
status: resolved
file: web/src/storybook/web-storybook-stories-and-fixtures.test.tsx
line: 30
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177749264,nitpick_hash:65c95664f0ad
review_hash: 65c95664f0ad
source_review_id: "4177749264"
source_review_submitted_at: "2026-04-27T01:09:45Z"
---

# Issue 003: Avoid new deep cross-system imports for agent stories.
## Review Comment

Lines 30-32 reach into `@/systems/agent/components/stories/...` internals from outside the agent system. Prefer importing through a public barrel (or an explicit `@/systems/agent/storybook` barrel) to keep boundaries stable.

As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

## Triage

- Decision: `valid`
- Notes:
  - The storybook regression test imports three agent story modules from `@/systems/agent/components/stories/...`, crossing into the agent system internals from the shared storybook test surface.
  - There is already a precedent for system storybook barrels in `web/src/systems/network/storybook.ts`.
  - The fix requires a minimal new support file, `web/src/systems/agent/storybook.ts`, that explicitly re-exports the agent story modules as named namespaces, then updates the test to import those modules through `@/systems/agent/storybook`.

## Resolution

- Added `web/src/systems/agent/storybook.ts` as an explicit public Storybook barrel for agent stories.
- Updated `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx` to load agent stories through `@/systems/agent/storybook`.
- Verified with targeted Vitest and full `make verify`.
