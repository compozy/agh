# Approved Plan — AGH Network Thread Routing and Prompt-Cost Accounting (Round 1)

- Status: APPROVED by controller (Claude Fable) with operator delegation, 2026-07-01.
- Author: Codex worker (gpt-5.5, reasoning xhigh, YOLO mode) in cmux workspace:3 / pane:23 / surface:38, Plan Mode.
- Accepted via option 1 ("Yes, implement this plan") — investigation context retained.
- Evidence base: `.compozy/tasks/network-token-optimization/analysis/` (slices 01–05 + summary.md).
- Delegation packet: `.compozy/tasks/network-token-optimization/codex-plan-packet.md`.

## Approval notes (controller)

- Plan correctly reuses the existing `network_thread_participants` table (already keyed `(workspace_id, channel, thread_id, peer_id)`, previously unused by routing) as the membership substrate — cheapest correct fix for the `localPeerMatchesConversation` pass-through root cause (router.go:1167).
- Embedded product decision, accepted consciously: an unaddressed new thread produces zero prompt deliveries (explicit `to` targeting or participant-only untargeted replies). Public thread history remains readable via existing surfaces.
- Deliberately deferred to Round 2 (do NOT discard): delivery composition diet (summary.md step 1), per-session queue digest/batching (step 2), volume-control verbs + cost config knobs (step 4), hook payload enrichment, and the new response-behavior finding (slice 06, `06_analysis_response-behavior.md` — agents answer every thread message as a huge final answer; no register/etiquette guidance exists).
- Watch items for verification: v43 must be the actual migration tail; `to` is single-target (multi-party thread bootstrap is one-at-a-time — mentions list is Round 2 material); no new config keys in this pass.

## Plan (verbatim from Codex)

# AGH Network Thread Routing and Prompt-Cost Accounting

## Summary

Cut thread prompt fan-out by making surface:"thread" a routing boundary for prompts, while keeping public
thread history readable through existing network inspection surfaces. This pass will not change NATS
transport subjects, add turn-taking/budget/interruption behavior, add CRDT/RL routing, or redesign web UI.

Evidence from investigation:

- router.go currently routes non-directed say and capability to all local peers in the channel, and
  localPeerMatchesConversation returns true for every non-direct surface.
- network_thread_participants already exists keyed by (workspace_id, channel, thread_id, peer_id), but only
  records historical participants and is not used by routing.
- Prompt cost is incurred at deliveryCoordinator.processQueuedItem, where one rendered network prompt is
  sent per delivery.
- network_audit_log.size is already persisted and scanned by the store, but not exposed in API payloads.
- Current token_stats is keyed by (session_id, agent_name), so the requested workspace/channel/thread/peer
  aggregate requires a new network-specific table.

Default chosen for this hard cut: an unaddressed new thread does not implicitly prompt every channel
member. The sender must target a peer with to, or later untargeted replies route only to peers already
recorded as members of that thread. This is the smallest behavior change that actually reduces prompt cost
without transport changes.

## Key Changes

- Thread routing:
  - Replace the non-direct pass-through behavior in localPeerMatchesConversation.
  - Keep surface:"direct" routing exactly as deterministic direct-room membership.
  - For surface:"thread" with to, deliver only to that local target peer, excluding the sender.
  - For surface:"thread" without to, deliver only to peers listed in network_thread_participants for
    (workspace_id, channel, thread_id), excluding the sender.
  - Keep greet/whois/discovery and no-surface channel traffic channel-scoped as today.
  - Add a narrow thread membership resolver interface in internal/network; the manager backs it with the
    store. Stop for approval if this requires callers outside the claimed file set.

- Thread membership persistence:
  - Add store read support for thread participant lookup/list by workspace/channel/thread.
  - Continue upserting peer_from for thread conversation messages.
  - Also upsert peer_to when present so an explicitly targeted thread recipient becomes a member before
    later untargeted replies.
  - Do not rename network_thread_participants; it already has the right key, and a rename would add
    migration/history blast radius without improving behavior.

- Prompt-size metadata:
  - Extend internal/acp.PromptNetworkMeta with additive JSON fields:
    - prompt_size_bytes
    - estimated_prompt_tokens
  - Compute both from the rendered prompt text in deliveryCoordinator.processQueuedItem, not from the
    envelope.
  - Use estimated_prompt_tokens = ceil(len(rendered_prompt_bytes) / 4).
  - This is measurement only. It must not enforce budgets, interrupt turns, reprioritize queues, or
    change turn-taking.

- Store-backed aggregate:
  - Append global DB migration v43, e.g. add_network_thread_peer_token_stats.
  - Create network_thread_peer_token_stats keyed by (workspace_id, channel, thread_id, peer_id).
  - Fields: workspace_id, channel, thread_id, peer_id, delivered_count, prompt_size_bytes,
    estimated_prompt_tokens, first_delivered_at, last_delivered_at, updated_at.
  - Add store.NetworkThreadPeerTokenStats, update/query types, validation, and GlobalDB update/list
    methods using the additive INSERT ... ON CONFLICT ... DO UPDATE pattern from UpdateTokenStats.
  - Enrich the internal delivery callback to pass prompt_size_bytes and estimated_prompt_tokens to
    Manager.recordDelivered.
  - Update the aggregate only after successful prompt delivery, beside delivered-audit recording.
    Dropped, rejected, render-failed, and interrupted deliveries do not increment prompt cost.

- API and generated contracts:
  - Add size_bytes to NetworkConversationMessagePayload, populated from persisted network_audit_log.size
    through store query support, not by recomputing in API handlers.
  - Add byte totals to NetworkPeerMetricsPayload: sent_size_bytes, received_size_bytes,
    rejected_size_bytes, delivered_size_bytes, total_size_bytes.
  - Add thread coordination-cost payloads: NetworkCoordinationCostPayload (delivered_count,
    prompt_size_bytes, estimated_prompt_tokens) and NetworkThreadPeerCostPayload (peer_id, same counters,
    first_delivered_at, last_delivered_at).
  - Add coordination_cost to NetworkThreadSummaryPayload.
  - Add peer_costs to thread detail responses only; omit from thread list responses unless explicitly
    requested later.
  - Run contract codegen. Generated openapi/agh.json and web/src/generated/agh-openapi.d.ts changes are
    part of internal/api (+ contract codegen). Stop if handwritten web UI changes beyond type/build fixes
    become necessary.

- Config:
  - No new config.toml keys. Existing internal/config.NetworkConfig has operational network limits only,
    and this pass is a protocol/runtime behavior hard cut.

- Docs and skill:
  - Update skills/agh/references/network.md so agents use to to prompt a peer into a thread and use
    untargeted thread replies only for existing participants.
  - Update packages/site protocol/runtime network docs where they say public thread prompt delivery
    reaches every channel member or that to does not affect delivery.
  - Preserve the distinction: public thread history remains inspectable/readable through existing runtime
    surfaces; prompt delivery becomes bounded to explicit targets or thread members.

## Hard-Cut Delete Targets

- Delete the router behavior that treats all non-direct conversation envelopes as channel-wide prompt
  deliveries.
- Delete docs and skill guidance claiming to in a public thread does not affect prompt delivery.
- Delete or rewrite tests that expect thread prompts to reach every channel member solely because they
  share a channel.
- Do not add compatibility flags, dual semantics, fallback broadcast routing, NATS subject changes, CRDT
  substrates, learned/RL routing, web redesign, or budget/interruption behavior.

## Test Plan

- Routing invariant (internal/network, extend existing router tests): targeted thread delivery reaches only
  to; untargeted thread delivery reaches existing participants only; unaddressed new thread produces no
  peer prompts; direct-room routing unchanged; discovery/no-surface channel messages remain channel-scoped.
- Prompt metadata invariant (internal/network + internal/acp, extend delivery_test.go and ACP prompt-meta
  serialization tests): rendered prompt size and estimated tokens appear in PromptNetworkMeta; estimate is
  deterministic; metadata valid for network turns only.
- Store and migration invariant (internal/store/globaldb, extend migration tests): v43 append-only after
  v42; fresh + upgraded DB create aggregate table; additive upserts accumulate per key; no cross-workspace
  merge; participant lookup workspace-scoped.
- API contract invariant (internal/api/core + contract, extend network handler tests, status+body):
  timeline messages include size_bytes; peer metrics include audit byte totals; thread list/detail include
  aggregate cost; thread detail includes peer costs; no cross-workspace leaks.
- Docs/static artifacts: no prose snapshot or filler tests; rely on make codegen-check, site build, make
  verify.

## Verification

- Focused: rtk go test ./internal/network -count=1 -race; rtk go test ./internal/store
  ./internal/store/globaldb -count=1 -race; rtk go test ./internal/api/... ./internal/acp
  ./internal/session -count=1 -race
- Regeneration: rtk make codegen; rtk make codegen-check
- Final gate: rtk make verify
- No commit/push — completion report with diff summary + verification evidence; controller owns the commit.

## Assumptions

- Accepted plan persisted post-approval (this file). Unrelated worktree entries untouched (skills-lock.json,
  docs/design/opendesign/_, .compozy/tasks/feature-stories/_).
- peer_id is the aggregate identity for prompt-cost accounting; provider TokenUsage stays session/turn
  accounting in this pass.
- Stop for approval if CLI/native tool schemas must change.

AGH Impact Audit:

- Native tools: No new tool IDs or verbs. Bundled AGH skill guidance changes so agents use `to` for thread
  prompt targeting and understand participant-only untargeted thread prompts.
- Extensibility and hooks: No extension, hook, bundle, registry, MCP sidecar, or config lifecycle change.
  ACP prompt metadata and HTTP/UDS/OpenAPI payloads change; codegen and docs co-ship.
- Workspace data isolation: New aggregate scoped by (workspace_id, channel, thread_id, peer_id); lookups,
  cost queries, timeline size joins, and API handlers filter by workspace with leak-proof tests.
- Official AGH skill: Update skills/agh/references/network.md for targeted thread prompt delivery,
  participant-only untargeted replies, public-history vs prompt-delivery distinction, and prompt-cost-aware
  coordination guidance.
