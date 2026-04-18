---
status: resolved
file: web/src/components/ui/stories/button-group.stories.tsx
line: 5
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:6827bacf61fc
review_hash: 6827bacf61fc
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 002: Replace relative imports with @/* aliases.
## Review Comment

Lines 5–8 should use the app alias to keep import boundaries consistent across `web/src`.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - `web/AGENTS.md` and `web/CLAUDE.md` both require `@/*` imports for files under `web/src`.
  - This story currently imports sibling UI modules via `../...`, so it violates the active import-path contract for touched web files.
  - Fix approach: replace the relative imports with `@/components/ui/...` aliases and rely on typecheck/lint to prove the rewritten paths resolve correctly.
