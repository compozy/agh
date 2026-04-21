---
status: resolved
file: web/src/systems/workspace/mocks/fixtures.ts
line: 6
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:c0afbd985019
review_hash: c0afbd985019
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 033: Avoid committing personal local filesystem paths in shared fixtures.
## Review Comment

These values leak a local username and make fixture output environment-specific. Prefer neutral, non-identifying paths.

Also applies to: 14-15

## Triage

- Decision: `valid`
- Notes: The workspace fixtures embed `/Users/pedro/...` filesystem paths, which leak a personal username and make the story data unnecessarily environment-specific. Fix by replacing them with neutral shared sample paths and add a regression check that the published fixture paths do not contain a personal home directory segment.

## Resolution

- Replaced the personal sample filesystem paths with neutral `/workspaces/...` paths and added a regression test that rejects personal home-directory paths in the exported workspace fixtures.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-typecheck`, `make web-test`, and `make verify`.
