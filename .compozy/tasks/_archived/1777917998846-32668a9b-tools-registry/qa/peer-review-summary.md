# Tool Registry TechSpec Peer Review Summary

## Round 1

- Reviewer: Claude Opus via `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json`
- Prompt: `qa/peer-review-prompt.md`
- Raw stream: `qa/peer-review-result.json`
- Extracted verdict: `qa/peer-review-verdict.json`
- Readiness: `NEEDS_REWORK`

Summary: The TechSpec carries the required quality markers and the core architecture is strong, but Opus found four blockers that needed resolution before approval.

## Blockers

- `B-001` Hosted MCP authentication was undefined, so a local process could impersonate another session projection. Resolved by adding session-bound proxy token binding, UDS bind handshake, redaction, invalidation, and safety invariants.
- `B-002` The bounded `agh__task_*` set was still an open wildcard, risking bypass of task authoritative primitives. Resolved by enumerating MVP task tools, excluding claim/release/complete/fail/run-start tools, and mapping included tools to `task.Service` methods.
- `B-003` Hosted MCP approval flow had no bridge to ACP `session/request_permission` and could become either a bypass or dead surface. Resolved by adding a Hosted MCP Approval Bridge with fail-closed behavior.
- `B-004` Extension backend execution scope was internally contradictory. Resolved by making external `mcp`, `extension_host`, and `subprocess` backend tools descriptor-only in MVP and post-MVP for call-through.

## Nits

All ten round-1 nits were addressed inline in `_techspec.md` and recorded in the TechSpec `## Nits` section.

## Follow-Up

The blockers were resolved after the round-1 verdict. A second-round confirmation was not run because the skill requires an explicit user request for additional rounds.
