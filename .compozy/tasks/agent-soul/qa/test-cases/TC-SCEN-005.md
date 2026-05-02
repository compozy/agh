# TC-SCEN-005: Live UDS Parity And Agent Context Projection Bounds

**Priority:** P0

## Objective

Prove that the UDS daemon surface can read and write Soul/Heartbeat data with the same DTO shape as HTTP, and that `/api/agent/context` keeps Soul projection bounded by `context_projection_bytes`.

## Preconditions

- Reused QA lab daemon is running.
- `AGH_UDS_PATH` points at the isolated daemon socket.
- Workspace `agent-soul-lab` contains `reviewer` and `ops` agents.
- At least one active session identity is available for agent-facing `/api/agent/context`.

## Test Steps

1. Write a managed Soul update through UDS `PUT /api/agents/reviewer/soul`.
   **Expected:** UDS response returns a valid mutation response with revision id and redacted relative source path.
2. Read the same Soul through UDS and HTTP.
   **Expected:** `digest`, `revision_id`, `agent_name`, and `limits` match.
3. Write a managed Heartbeat update through UDS `PUT /api/agents/ops/heartbeat`.
   **Expected:** UDS response returns a valid policy payload and revision id.
4. Read Heartbeat status through UDS and HTTP with recent wake events enabled.
   **Expected:** Policy digest, health fields, and wake event shape are equivalent.
5. Install or reuse a long Soul body whose compact projection must exceed `context_projection_bytes`.
   **Expected:** Full authoring read model can retain body content, but `/api/agent/context` returns only bounded compact projection.
6. Call `/api/agent/context` with a valid active session identity.
   **Expected:** Response includes Soul projection with `truncated=true` or bounded compact fields, and no raw full prompt transcript.

## Behavioral Evidence

- Operator journey: UDS write/read mirrors HTTP write/read for authored context.
- Agent journey: active session reads compact self-context without receiving unbounded Soul prose.
- Artifacts: UDS curl responses, HTTP curl responses, CLI context output, and byte-length/truncation checks.

## Disruption Probes

- Bounded context projection prevents accidental full Soul prompt leakage into agent self-context.
- UDS parity is proven by live socket calls, not only package tests.

