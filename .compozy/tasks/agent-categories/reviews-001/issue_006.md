---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: packages/ui/src/index.ts
line: 258
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239418533,nitpick_hash:58382fbe8808
review_hash: 58382fbe8808
source_review_id: "4239418533"
source_review_submitted_at: "2026-05-06T20:41:36Z"
---

# Issue 006: Re-export the Tree prop types from the package barrel.
## Review Comment

The components are public via `@agh/ui`, but their prop types still require a deep import into `./components/reui/tree`. Export the `Tree*Props` types here too so consumers can stay on the public entrypoint.

## Triage

- Decision: `valid`
- Notes:
  - `packages/ui/src/index.ts` exports the `Tree` components but not the public `Tree*Props` types.
  - Root cause: consumers need a deep import into `components/reui/tree` to reference the prop contracts, which bypasses the package barrel.
  - Fix approach: re-export the tree prop types from the package entrypoint and add package-level regression coverage for those exports.
  - Additional scope note: `packages/ui/README.md` was updated because `packages/ui/tests/readme.test.ts` requires every public barrel export to be documented.
