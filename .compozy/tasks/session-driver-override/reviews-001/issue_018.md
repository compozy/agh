---
status: resolved
file: web/src/routes/_app/session.$id.tsx
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4155866948,nitpick_hash:3e4521a891c1
review_hash: 3e4521a891c1
source_review_id: "4155866948"
source_review_submitted_at: "2026-04-22T15:22:24Z"
---

# Issue 018: Use the session public barrel import instead of a deep component path.
## Review Comment

Line 11 should import through `@/systems/session` to preserve system boundaries.

As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals."

## Triage

- Decision: `valid`
- Root cause: the route imports session-system internals directly instead of using the public `@/systems/session` barrel, which violates the web system-boundary rule for cross-system imports.
- Fix plan: switch the route to barrel imports for the session-system components, hooks, and types it consumes.
