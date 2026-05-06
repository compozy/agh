---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/cli/memory.go
line: 47
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233215394,nitpick_hash:e86a47ea675c
review_hash: e86a47ea675c
source_review_id: "4233215394"
source_review_submitted_at: "2026-05-06T04:43:05Z"
---

# Issue 005: Preserve agent tier in memory search output.
## Review Comment

`memorySearchItem` drops `AgentTier`, and the renderers print `string(item.Scope)`. That makes `agent:workspace` and `agent:global` results both show up as just `agent`, unlike the other memory commands.

Also applies to: 1500-1539

## Triage

- Decision: `valid`
- Root cause: `memorySearchItem` omits `AgentTier`, and `memorySearchBundle` renders `string(item.Scope)` instead of the existing `memoryScopeLabel` helper.
- Evidence: agent-global and agent-workspace search results both collapse to `agent` in human/toon output even though other memory commands preserve the tier suffix.
- Fix plan: carry `AgentTier` through `memorySearchItem` and render search scope via `memoryScopeLabel` so CLI output matches the rest of the memory surface.
- Resolution: implemented in `internal/cli/memory.go` and validated with targeted CLI regression coverage in `internal/cli/memory_test.go`, which was the minimal extra file needed to prove the renderer behavior.
- Verification: targeted `go test` for `internal/cli` passed, and fresh `make verify` passed on 2026-05-06.
