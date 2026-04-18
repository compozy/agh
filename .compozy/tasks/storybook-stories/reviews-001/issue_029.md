---
status: resolved
file: web/src/systems/session/components/tool-renderers/stories/search-content.stories.tsx
line: 6
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:1135a0a8c18b
review_hash: 1135a0a8c18b
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 029: Replace relative import with @/* alias.
## Review Comment

Line 6 uses a relative path import; switch it to the project alias for consistency and policy compliance.

As per coding guidelines, `web/src/**/*.{ts,tsx}`: "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: The story is under `web/src` and still imports `SearchContent` via a relative path, which violates the required `@/*` alias convention. Fix by importing the renderer through the aliased session-system path.

## Resolution

- Replaced the relative `SearchContent` import with the aliased session-system path and added explicit empty `args` to the touched stories.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
