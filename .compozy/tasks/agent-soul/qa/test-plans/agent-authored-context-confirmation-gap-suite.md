# Agent Authored Context Confirmation Gap Suite

## Executive Summary

This suite closes the independent Codex Loop confirmation gaps from the first Agent Soul QA pass. It is behavior-first: each case ties a TechSpec MUST or Safety Invariant to a live operator or agent-facing journey, then backs the live evidence with focused regression tests when the surface is race-sensitive or easier to prove deterministically at the package boundary.

## Scope

In scope:

- Soul refresh for idle sessions and conflict rejection while a task run is active.
- Active prompt race safety for Heartbeat wake dispatch.
- Extension and hook bypass negatives for authored Soul and Heartbeat files.
- Spawn lineage and `sessions.parent_soul_digest` provenance.
- `ClaimNextRun` task-run Soul provenance in `task_runs.metadata_json`.
- Wake coalescing, rate limiting, and wake event retention.
- `[agents.soul]` and `[agents.heartbeat]` config overlays and invalid-value rejection.
- `agh session heartbeat` absence.
- `/api/agent/context` projection truncation and missing/invalid session identity rejection.
- Live UDS read/write parity for Soul and Heartbeat authoring surfaces.

Out of scope:

- Web editor flows for Soul or Heartbeat, because the MVP intentionally exposes no Web editor for these authored files.
- Deterministic LLM token-for-token assertions. Provider output is judged by behavior and runtime state, not exact prose.

## Behavioral Scenario Charter

Operator intent: a startup operator has shipped authored agent context and needs confidence that agents, extensions, hooks, and CLI/API/UDS callers cannot bypass the managed authoring and task-claim contracts during real work.

Startup situation: the existing `agent-soul-lab` workspace contains `reviewer` and `ops` agents, managed `SOUL.md` and `HEARTBEAT.md` fixtures, a running isolated daemon, and provider-backed sessions from the first QA pass.

Agent roles:

- `reviewer`: owns Soul/persona validation and child-session lineage checks.
- `ops`: owns Heartbeat policy, wake coalescing, rate limit, and retention checks.
- Agent-session caller: exercises UDS `/api/agent/*` and task-claim surfaces through daemon-issued identity.
- Extension/hook actor: attempts denied authored-context mutations without grants and verifies no raw file-write control exists in hook patches.

Expected evidence:

- CLI JSON and HTTP/UDS JSON responses for every public journey.
- SQLite readbacks for `sessions.parent_soul_digest`, `task_runs.metadata_json`, and wake event retention.
- Focused Go regression logs for race-sensitive or security-boundary contracts.
- Final `make MAGE= verify` evidence from the current repo state.

## Execution Order

1. Run `TC-SCEN-004` to prove Soul refresh, active-run conflict, spawn lineage, and task claim provenance.
2. Run `TC-SCEN-005` to prove live UDS parity and context projection truncation.
3. Run `TC-REG-005` to prove wake coalescing/rate limiting/retention and config validation.
4. Run `TC-SEC-001` to prove extension/hook bypass denial and HTTP identity rejection.
5. Re-run focused package tests for any code-level regressions added during this pass.
6. Run the full monorepo verification gate with `make MAGE= verify`.

## Exit Criteria

- Every new P0/P1 case in this suite is PASS or has a filed/fixed bug with rerun evidence.
- No bypass path can mutate authored files outside managed Soul/Heartbeat services.
- Live UDS evidence exists for both read and write paths.
- Final verification gate passes from the current worktree.

