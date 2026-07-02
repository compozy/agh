---
name: network-token-megaround-review
task: implementation peer review of AGH network token optimization megaround (round 1)
---

- Goal (incl. success criteria): Peer-review the megaround impl (channel fanout policy, mentions, subscriptions/digest/mute, durable guidance suppression, response register, thread→task promotion, agh.runtime status-back, designated fan-out runs, migrations v44-v47, contracts codegen, web+docs). Write findings to `.codex/peer-reviews/network-token-cost-impl/impl-review-findings-round1.md`. Verdict SHIP/FIX_BEFORE_SHIP/REWORK.
- Constraints/Assumptions: Greenfield, hard cuts only. task*runs single queue; ClaimNextRun authoritative; no peer claimer; hooks dispatch at call site; claim_token raw never crosses boundary; codegen co-ship; %w error wrapping; no *-discard; workspace_id isolation on every new datum. Scoped-write: only the target findings file.
- Key decisions: Migrations v44-v47 registered (confirmed at global_db.go:1168-1186). Dispatch review subagents per slice; verify blockers myself before writing.
- State:
- Done: Read packet + plan. Confirmed migration tail v44-v47.
- Now: Dispatch parallel review subagents (store/migrations, network, task/fan-out, api/codegen, daemon/prompt, cli/tools/config, web, docs).
- Next: Verify blocker-candidates myself; write findings file.
- Open questions (UNCONFIRMED): none yet.
- Working set: .codex/peer-reviews/network-token-cost-impl/impl-review-diff-round1.patch; internal/network, internal/task, internal/store/globaldb, internal/daemon, internal/api, internal/cli, internal/tools, web/src/systems.
