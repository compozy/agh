---
status: resolved
file: web/src/systems/knowledge/components/stories/knowledge-list-panel.stories.tsx
line: 8
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:7414166e7919
review_hash: 7414166e7919
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 023: Replace relative panel import with alias import
## Review Comment

Line 8 should use `@/*` aliasing instead of `../knowledge-list-panel`.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: This story file is under `web/src` and currently uses a relative import for the panel component, which violates the `@/*` alias policy in `web/AGENTS.md`. Fix by importing `KnowledgeListPanel` through the `@/systems/knowledge/...` alias path.

## Resolution

- Replaced the relative `KnowledgeListPanel` import with the aliased system path and added explicit empty `args` on the touched stories.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
