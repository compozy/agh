---
status: resolved
file: internal/daemon/agent_skill_resources.go
line: 588
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:471298e4ddd9
review_hash: 471298e4ddd9
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 037: Silent encode errors in comparison methods may cause unnecessary updates.
## Review Comment

The `sameAgent`, `sameSkill`, and `sameMCPServer` methods return `false` when encoding fails, treating encode errors as "resource changed". While this is safe (it will trigger an update), it could mask codec issues and cause repeated unnecessary writes on every sync cycle if the codec consistently fails.

Consider logging encode errors at debug level to aid troubleshooting:

## Triage

- Decision: `INVALID`
- Notes: The `sameAgent`, `sameSkill`, and `sameMCPServer` encode paths operate on validated plain structs that are already marshaled successfully when desired resources are produced. If an unexpected encode failure ever does occur, returning `false` intentionally forces a rewrite on the next sync, which is the safe self-healing behavior. Adding debug logging here would add noise in a hot comparison path without changing correctness, so I am resolving this as non-actionable.
