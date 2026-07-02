# Approved Plan — AGH Network Token Optimization Megaround

- Status: APPROVED by controller (Claude Fable) with operator delegation, 2026-07-01. Accepted via option 1.
- Author: Codex worker (gpt-5.5, reasoning xhigh, YOLO) on cmux workspace:3 / pane:23 / surface:40, Plan Mode.
- Contract: `.compozy/tasks/network-token-optimization/codex-megaround-packet.txt`.
- Evidence base: `.compozy/tasks/network-token-optimization/analysis/` (01–08 + summary.md) + Round 1 (`.claude/plans/2026-07-01-network-thread-routing-cost-round1.md`).
- Git context (user-intentional): work happens on branch `network-optimization`; savepoint `669f6b503` carries Round-1 content; main deliberately left at `1630bfe95` by the operator.

## Plan (verbatim from Codex)

# AGH Network Token Optimization Megaround

## Summary

Implement the packet as a hard-cut redesign that separates the conversation plane from the execution plane:

- Conversation threads decide, discuss, review, and promote work; they do not execute heavy or mutating work directly.
- Actionable thread work is promoted into kanban tasks with typed origin links, compact digest context, and task/run status posted back to the source thread.
- Network deliveries become cheaper and more targeted through mention-aware routing, channel fanout policy, durable delivery guidance suppression, digest mode, subscriptions, and a short response register.

Before any code changes, persist this approved plan under .codex/plans/<timestamp>\*network-token-optimization-megaround.md and set the detailed goal per the project instruction. No compatibility bridges or aliases will be added.

## Implementation Changes

### 1. Schema, Config, and Contracts

- Append migrations after the current v43 tail:
  - v44: add network_channels.fanout_policy, network_channels.coordinator_peer_id, and network_timeline_log.mentions_json.
  - v45: add network_subscriptions keyed by (workspace_id, channel, thread_id, peer_id) plus network_delivery_guidance_state for durable per-session guidance suppression.
  - v46: add network_task_thread_origins keyed by task_id, with workspace_id, channel, thread_id, origin_message_id, compact digest, source message IDs, and timestamps.
  - v47: add task_runs.designation_group_id, an index on (task_id, designation_group_id), and a small task_designation_rollups table keyed by designation group.
- Add config defaults:
  - network.activation_top_k = 3
  - network.digest_flush_interval = 250ms
  - network.digest_max_envelopes = 10
  - network.response_guidance_max_bytes = 512
  - network.delivery_structured_body_max_bytes = 4096
  - task.orchestration.designated_run_max = 5, validated in range 1..5
- Extend public contracts and regenerate OpenAPI/types with make codegen:
  - mentions []string on network send requests, send payloads, envelopes, timeline messages, and prompt metadata.
  - fanout_policy and coordinator_peer_id on channel create/detail/update payloads.
  - subscription payloads with mode: mute|digest|full and optional keyword_filters.
  - task thread origin payloads and designated fan-out run payloads.
- Add or update HTTP/UDS surfaces:
  - PATCH /api/workspaces/{workspace_id}/network/channels/{channel} for fanout policy and coordinator.
  - channel/thread subscription list, set, and delete routes.
  - POST /api/workspaces/{workspace_id}/network/channels/{channel}/threads/{thread_id}/promote.
  - POST /api/tasks/{task_id}/runs/fan-out.
- Add or update native tools and CLI:
  - update agh\_\_network_send and agh network send with repeatable --mention.
  - add channel update, subscribe, mute, unmute, and digest-mode tools/commands.
  - add agh\_\_task_promote_from_thread and agh task promote.
  - add agh\_\_task_fanout_runs and agh task fan-out.

### 2. Conversation Plane Routing and Delivery Diet

- Replace the current zero-delivery branch for unaddressed empty-participant threads with a deterministic activation resolver:
  - capability_match is the default channel policy.
  - coordinator routes to the configured coordinator_peer_id; missing or unavailable coordinator falls back to digest.
  - all_members is explicit opt-in and full-injects all eligible local peers.
- Capability matching is token-free and deterministic:
  - Use SayBody.Intent when present, otherwise SayBody.Text.
  - Score each local peer by normalized term overlap against capability ID and summary.
  - Select up to network.activation_top_k peers with positive scores, tie-breaking by peer ID.
  - If no peer matches, emit a digest fallback to eligible thread/channel subscribers and record an observable activation decision.
- Add mention semantics:
  - Envelope.Mentions []string is normalized, deduplicated, validated as peer IDs, cloned, logged, summarized, and rendered in prompt metadata/XML attrs.
  - Mention-only messages remain broadcast subjects; subjectForEnvelope stays direct only for To.
  - Mentioned peers receive full injection, become durable thread participants, and override fanout/digest defaults; mute suppresses only unmentioned/unaddressed traffic.
- Add subscription resolution at the delivery-set site:
  - Most-specific thread row wins over channel row.
  - Empty keyword_filters means always applicable.
  - Non-empty filters are lowercased substring guards over intent/text; if no filter matches, resolution falls through to the less-specific/default rule.
  - mute suppresses matching unmentioned traffic, digest converts matching full traffic to digest, and full preserves full injection.
- Add DeliveryMode with full, queued, and digest, plus activation/subscription reason fields for audit and hooks.
- Replace prompt bloat:
  - say bodies render as escaped plain text, not base64 JSON.
  - structured non-say bodies remain base64 JSON but are capped by network.delivery_structured_body_max_bytes.
  - protocol/reply guidance is suppressed through durable network_delivery_guidance_state, surviving reconnect and dropSession.
  - prompt metadata remains visible; if any adapter path cannot expose metadata, keep a compact in-band line rather than expanding guidance.
- Add digest formatting:
  - queue backlog and activation fallback can coalesce up to network.digest_max_envelopes into one digest prompt after network.digest_flush_interval.
  - prompt cost attribution is allocated across included envelopes by canonical envelope size, so totals match actual digest prompt cost instead of charging each envelope as if it received a full injection.
- Extend hook/audit payloads with activation reason, delivery mode, prompt size bytes, estimated tokens, digest envelope count, and subscription decision.

### 3. Response Register and Execution Plane Bridge

- Add a startup-gated, budget-capped network etiquette section through ComposedAssembler:
  - only active for network-capable sessions.
  - bounded by network.response_guidance_max_bytes.
  - states the golden rule: threads decide/discuss; actionable work is promoted to tasks; replies are short and only when addressed, activated, or adding value.
- Add a TurnSourceNetwork input augmenter:
  - one compact line before network prompts.
  - reinforces brevity, activation rules, and task promotion.
  - uses prompt metadata when available and falls back to compact channel/thread/message text only when required.
- Implement thread-to-task promotion:
  - create a compact pin-and-trim digest around the origin message using task.orchestration.summary_max_bytes.
  - store typed origin data in network_task_thread_origins.
  - create the task only; do not enqueue a run unless the caller separately uses an explicit enqueue/fan-out surface.
  - web thread detail shows a promote action and linked task card; task views link back to the source thread.
- Add status-back:
  - add a task event observer to composeTaskEventObserver.
  - for tasks with network thread origins, post one compact status message on meaningful lifecycle transitions and on designation rollup completion.
  - use a first-class runtime peer identity, agh.runtime, through a new runtime/system send path; do not impersonate a session peer.
- Implement designated sub-runs:
  - add FanOutRuns that validates N <= task.orchestration.designated_run_max and enqueues each run through authoritative EnqueueRun.
  - each run gets designation_group_id plus metadata containing index, count, brief, origin, and prompt budget.
  - render designation briefs through TaskRunPromptOverlay -> opts.PromptOverlay -> manager_start.go.
  - when all runs in a group are terminal, write a rollup artifact and emit one task status-back message to the originating thread.

### 4. Web, Docs, Skills, and Cleanup

- Update web network/thread surfaces to expose:
  - channel fanout policy and coordinator editing.
  - mentions in the composer.
  - subscription controls for mute, digest, and full.
  - thread promotion and linked task cards.
  - designated fan-out entry point from task surfaces.
- Keep UI token-compliant with packages/ui/src/tokens.css and DESIGN.md; no invented visual grammar.
- Update docs and agent guidance:
  - packages/site network, task, native-tool, and config docs.
  - skills/agh/SKILL.md and relevant references for mentions, subscriptions, task promotion, status-back, designated fan-out, and the two-plane golden rule.
  - COPY.md vocabulary remains authoritative; use "capability" terminology.
- Delete obsolete behavior instead of bridging:
  - remove silent zero-delivery for empty participant threads.
  - remove session-local-only protocol guidance suppression.
  - remove mandatory base64 JSON prompt rendering for chat say bodies.
  - remove single-envelope assumptions in delivery prompts where digest mode applies.
- Keep all changes workspace-scoped where network data crosses workspaces; every new store query and API route must include workspace_id.

## Test Plan

- Test placement decisions before writing tests:
  - Routing/mentions/subscriptions invariant: owning layer internal/network; canonical suite existing router/delivery tests.
  - Schema/storage invariant: owning layer internal/store/globaldb; canonical migration/store suites.
  - API/UDS/native/CLI parity invariant: owning layers internal/api, internal/daemon, and CLI suites.
  - Task promotion/status-back/fan-out invariant: owning layers internal/task and internal/daemon; use existing task manager and observer suites.
  - Web behavior invariant: existing web system/component tests only where runtime behavior is user-visible; no snapshot or config-existence filler tests.
- Focused checks during implementation:
  - migration fresh DB and reopen tests for v44-v47.
  - router tests for capability_match, coordinator, all_members, no-match digest fallback, mentions, To precedence, and subscription precedence.
  - delivery tests for plain-text say, structured body cap, durable guidance across reconnect/dropSession, digest coalescing, and honest cost allocation.
  - task tests for promotion origin storage, no auto-enqueue, status-back observer, runtime peer identity, fan-out validation, overlay rendering, and group rollup.
  - contract/API/CLI/native tests for all new public surfaces.
  - web tests for mentions, subscription controls, promotion links, and fan-out affordance.
- Required final verification:
  - make codegen
  - make codegen-check
  - focused Go and Bun/Turbo tests for changed packages.
  - agh-ui-screenshot captures for changed network/task UI stories or routes.
  - make verify passing with zero warnings/errors.

## Assumptions and Defaults

- fanout_policy = capability_match is the default for all new and existing channels after the hard-cut migration.
- coordinator_peer_id is explicit per channel; no naming convention or implicit "coordinator" capability is used.
- Mentions are explicit activation and full-injection signals; they do not change broadcast subject routing.
- Explicit To keeps existing directed semantics and remains stronger than mention/fanout policy.
- Digest fallback is observable even if no eligible local peer exists; in that case it records delivered=0 with reason rather than silently disappearing.
- Thread promotion uses the existing task summary byte budget instead of introducing another digest-size config key.
- Designated fan-out does not repurpose parent_run_id or previous_run_id; it uses designation_group_id and typed metadata.
- All artifacts, docs, code, task specs, and plans are written in English; conversation can remain BR-PT.

AGH Impact Audit:

- Native tools: update agh\_\_network_send; add channel update, subscription mode, thread promotion, and task fan-out tools; update descriptors, schemas, digests, capability gates, diagnostics, and tests.
- Extensibility and hooks: update network hook payloads with activation/subscription/delivery/cost fields; add config lifecycle for new network/task keys; update OpenAPI, CLI, UDS, HTTP, docs, bundled skill guidance, and task prompt overlay behavior.
- Workspace data isolation: all channel policy, mentions, subscriptions, task thread origins, digest guidance, status-back, fan-out runs, web queries, SSE/cache paths, and store queries must carry workspace_id; tests must prove cross-workspace thread/subscription/task-origin data does not leak.
- Official AGH skill: update skills/agh/SKILL.md and relevant references for the conversation/execution split, mentions, subscription controls, task promotion, status-back, designated fan-out, and native tool usage.
