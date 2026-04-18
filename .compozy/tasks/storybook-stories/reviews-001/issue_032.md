---
status: resolved
file: web/src/systems/workspace/mocks/fixtures.ts
line: 1
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:a5fc66c8f59e
review_hash: a5fc66c8f59e
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 032: Use @/* alias import for workspace types.
## Review Comment

Replace the relative import with the configured path alias.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `valid`
- Notes: The workspace mock fixture file is under `web/src` and still imports types through a relative path, which conflicts with the required `@/*` alias convention. Fix by switching the type import to the workspace-system alias path.

## Resolution

- Switched the workspace fixture type import to `@/systems/workspace/types`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
