---
status: resolved
file: web/src/systems/knowledge/components/stories/knowledge-detail-panel.stories.tsx
line: 7
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:5eb3dc9fae6c
review_hash: 5eb3dc9fae6c
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 022: Replace relative component import with @/* alias
## Review Comment

Line 7 should use the alias import style instead of `../knowledge-detail-panel`.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: `web/AGENTS.md` requires `@/*` alias imports for files under `web/src/**/*.{ts,tsx}`. This story still imports the panel via a relative path, so the review comment matches the active workspace policy. Fix by switching the component import to the `@/systems/knowledge/...` path.

## Resolution

- Switched the `KnowledgeDetailPanel` story to the `@/systems/knowledge/components/knowledge-detail-panel` import path and kept the edited story exports explicit with `args: {}`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
