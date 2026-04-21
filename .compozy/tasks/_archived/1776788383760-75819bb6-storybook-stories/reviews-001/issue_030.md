---
status: resolved
file: web/src/systems/session/mocks/fixtures.ts
line: 1
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:37aae40ae4dc
review_hash: 37aae40ae4dc
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 030: Use @/* alias for this types import.
## Review Comment

Please switch this relative import to the source alias to align with `web/src` conventions.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: `web/src/systems/session/mocks/fixtures.ts` currently imports its types via `../types`, but `web/AGENTS.md` requires `@/*` aliases for `web/src` imports. Fix by switching the type import to the session-system alias path.

## Resolution

- Switched the session fixture type import to `@/systems/session/types`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
