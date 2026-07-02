---
schema_version: 1
review_kind: implementation
round: 1
verdict: FIX_BEFORE_SHIP
reviewer_runtime: claude
reviewer_model: opus
generated_at: 2026-07-02T01:58:32Z
---

# Summary

The change shape is right — single `task_runs` queue with `ClaimNextRun` authoritative, no parallel queue or peer claimer, append-only migrations v44–v47, clean contract/codegen co-ship, full HTTP/UDS/CLI parity, `agh.runtime` as a first-class runtime peer, guidance-only response register, and preserved `claim_token` redaction — but three flagship behaviors are stubbed or wrong while every surface (contract, CLI, native tool, web, docs, skill) was completed to look done. Channel `fanout_policy`/`coordinator_peer_id` are stored and surfaced everywhere yet never consulted in routing; designated fan-out is non-functional for N≥2; digest "coalescing" is unimplemented (its config knobs are dead); mute drops directly-addressed traffic; and the entire subscription/mute/digest/activation delivery machinery ships with zero behavioral tests — which is exactly why the broken behaviors slipped through.

# Blockers

## B-001 — Channel `fanout_policy`/`coordinator_peer_id` are fully surfaced but never enforced in routing

- File: internal/network/router.go
- Line: 1207
- Issue: The delivery-set computation `(*Router).deliveriesFromLocalPeers` (router.go:1207-1233) and `selectActivatedThreadPeers` (1234-1272) never read the channel's `fanout_policy` or `coordinator_peer_id`. For an unaddressed empty-participant thread they always run the same capability-ranked top-K activation regardless of the channel's policy. I confirmed first-hand: `grep -rni "fanout|coordinator_peer|capability_match|all_members" internal/network` (non-test) returns nothing. The field is persisted (network_channels, migration v44), validated (`store.ValidateNetworkFanoutPolicy`), settable via CLI (`agh network channels update`), native tool (`agh__network_channel_update`), HTTP+UDS PATCH, the web policy dialog, and documented in the site + bundled skill — but selecting `coordinator` or `all_members` has zero runtime effect, and `capability_match` runs unconditionally. This is the central A1 deliverable ("channel activation policy") and it is not wired into delivery.
- Rationale: This is a spec-completeness failure on the round's headline feature plus a Truthful-UI/Truthful-tool violation. CLAUDE.md: "Don't render controls or metrics the runtime doesn't actually support"; `internal/CLAUDE.md` agent-manageability requires control surfaces to actually control. Operators and agents set a policy that silently does nothing, and the web dialog (channel-policy-dialog.tsx:46-52,170) and docs (threads.mdx, channels-and-peers.mdx, delivery.mdx, skills/agh/references/network.md:84) assert `coordinator`/`all_members` routing behaviors that do not exist. Verified independently by me and by two review agents that each read router.go directly.
- Suggested fix: In `deliveriesFromLocalPeers`, resolve the channel entry (workspace_id, channel) and branch on `fanout_policy`: `coordinator` → full-inject the `coordinator_peer_id` (digest-fallback when missing/unavailable), `all_members` → full fan-out, `capability_match` (default) → the current activation with digest fallback (see B-004). Then correct the web/docs/skill copy in the same cut. If policy enforcement cannot land this round, do a truthful descope: remove/relabel the coordinator + all_members controls and doc claims rather than shipping controls that lie.

## B-002 — Designated fan-out with N≥2 is non-functional (2nd sibling rejected at enqueue)

- File: internal/store/globaldb/global_db_task_aux.go
- Line: 1477
- Issue: `enqueueFanOutTaskRuns` (api/core/tasks.go:1213-1240) loops `manager.EnqueueRun` once per designation. `EnqueueRun` → `ReserveQueuedRun` (global_db_task_aux.go:618) calls `findOpenRunIDForQueuedRunReservation` (1477-1504), which selects ANY run `WHERE task_id = ? AND status NOT IN (completed, failed, canceled)` with no `designation_group_id` awareness. The first designation creates a `queued` (non-terminal) run; the second designation's reservation finds that open run and returns `ErrInvalidStatusTransition` ("finish or cancel it before enqueueing another run"). So a fan-out of N≥2 aborts after creating exactly one run — the core C1 feature (N parallel sibling runs, default cap 5) never produces siblings. Per-designation idempotency keys differ, so the idempotency short-circuit does not save it.
- Rationale: Correctness bug that makes the flagship execution-plane deliverable non-functional. It went undetected because there is no test that enqueues 2+ designated runs through the real store/`ReserveQueuedRun` path — the observer test seeds sibling runs directly via `CreateTaskRun`, and the in-memory manager stub replicates the same designation-blind `hasOpenRun` guard (manager_test.go:1305). This is the single-durable-queue invariant reused with a single-open-run guard that fan-out contradicts.
- Suggested fix: Make the open-run guard designation-aware: when reserving a run whose `designation_group_id` is set, exclude runs sharing that same group id from the open-run check (siblings are expected to coexist as `queued`), while keeping the guard for non-designated enqueues. Apply the same fix to the in-memory test stub and add a store-level test that enqueues N=2..5 designated runs and asserts N distinct `queued` rows share one group id.

## B-003 — Digest mode does not coalesce; `digest_flush_interval`/`digest_max_envelopes` are dead knobs; cost still charged per-envelope

- File: internal/network/delivery.go
- Line: 1170
- Issue: `formatNetworkDigestMessage` (delivery.go:1170-1204) takes ONE `Envelope` and renders a compact preview-only wrapper. There is no batching loop and no flush timer; the delivery worker dequeues one item at a time (delivery.go ~595-651). The config keys `DigestFlushInterval` (250ms) and `DigestMaxEnvelopes` (10) are defined, defaulted, and validated (config.go:468-469,1745-1749) but never read in `internal/network` (confirmed: `grep DigestFlushInterval|DigestMaxEnvelopes internal --include=*.go` hits only config/merge/tool_surface/api-settings). Because there is no coalescing, prompt cost is still attributed per-envelope as a full injection (`recordThreadPeerPromptCost` with `DeliveredCount: 1`, manager.go ~1885) — the plan's explicit "allocate cost across included envelopes by canonical size" is not implemented.
- Rationale: Named A3 deliverable and Test Plan item ("digest coalescing, and honest cost allocation") unimplemented, with validated-but-dead config (a Truthful-config problem: settings UI and CLI accept knobs that do nothing). The web dialog (channel-policy-dialog.tsx:171), site docs (delivery-and-safety.mdx:41 and the `digest_max_envelopes` = "Maximum envelopes summarized into one digest prompt" row at :96; threads.mdx:73), and bundled skill (network.md:90) all claim digest "batches"/"coalesces" — a Truthful-UI/docs violation. Note: digest _rendering_ as a per-envelope compact form does work and is usable via subscription mode; only the coalescing + honest cost half is missing.
- Suggested fix: Either implement queue coalescing driven by `DigestFlushInterval`/`DigestMaxEnvelopes` (render one prompt from up to N envelopes and split prompt cost across them by canonical body size), or truthfully descope: drop the two dead config keys, remove the "batches/coalesces/summarized into one prompt" claims from web/docs/skill, and keep digest as the compact per-envelope mode it actually is.

## B-004 — capability_match no-match path full-injects to top-K instead of digest fallback

- File: internal/network/router.go
- Line: 1234
- Issue: When no peer's capability terms overlap the message, `activationScore` returns 0 for every peer (router.go:1290-1348), but `selectActivatedThreadPeers` still fills up to `activationTopK` peers by score-desc/PeerID-asc order and returns them as normal deliveries with empty `Mode`, which `applyDeliverySubscriptions` forces to `full` (manager.go:1155). There is no score threshold that drops zero-score peers and no downgrade to digest. `router_test.go:436` ("Should activate channel peers when no thread participants exist") asserts a full delivery to the sole channel peer with no capability match — i.e. the test encodes the divergent behavior.
- Rationale: Directly contradicts the spec's explicit rule: "If no match → digest fallback to thread/channel members — NEVER silently nothing, NEVER all-members full injection." The current behavior is bounded to K (default 3) but is still full injection to irrelevant peers with no digest downgrade, defeating the round's token-cost goal on exactly the unaddressed-thread case the round targets.
- Suggested fix: When the matcher yields no positive-score peer, emit the fallback deliveries with `Mode = digest` (to thread/channel members) rather than `full`, and record the observable activation reason. Update router_test.go:436 to assert digest fallback, not full injection.

## B-005 — `mute` suppresses directly-addressed traffic (silent message loss)

- File: internal/network/manager.go
- Line: 1179
- Issue: `deliverySubscriptionMode` (manager.go:1179-1239) exempts only mentioned peers (`deliveryMentionsPeer` → `Full` at 1187-1189). It has no exemption for addressed/directed (`To`) traffic. Directed messages reach `applyDeliverySubscriptions` on the inbound path (manager.go:1134); a direct-surface delivery is filtered to exactly the `To` peer via `localPeerMatchesConversation` (router.go), then `deliverySubscriptionMode` consults the channel-level subscription. If that peer has a `mute` row for the channel and is not mentioned, the mode resolves to `mute` and `applyDeliverySubscriptions` drops the delivery (manager.go:1161-1162).
- Rationale: Violates the explicit spec/plan invariant "mute suppresses only unmentioned/**unaddressed**" and the plan's "mute suppresses only unmentioned/unaddressed traffic." The failure is silent message loss: a peer who muted a channel will never receive a 1:1 direct message addressed to them. Addressed traffic must always be delivered.
- Suggested fix: Add an addressed exemption alongside the mention exemption, e.g. `if delivery.Envelope.IsDirected() && strings.TrimSpace(*delivery.Envelope.To) == delivery.PeerID { return store.NetworkSubscriptionModeFull, nil }`, and add a test asserting a muted peer still receives directed/direct messages.

## B-006 — Entire subscription/mute/digest/keyword + activation-fallback delivery machinery ships with zero behavioral tests

- File: internal/network/manager_test.go
- Line: null
- Issue: No test in `internal/network/*_test.go` references `deliverySubscriptionMode`, `firstApplicableSubscription`, `formatNetworkDigestMessage`, `applyDeliverySubscriptions`, `ListNetworkSubscriptions`, `selectActivatedThreadPeers`, `rankActivationCandidates`, or `activationScore` (confirmed by grep). The only new activation tests (router_test.go:436,501) assert the full-injection behavior from B-004. There is no coverage for subscription mode precedence (thread row wins over channel), empty-vs-non-empty keyword filters, mute-suppresses-only-unmentioned/unaddressed, digest rendering, or activation top-K ordering. Combined with the missing N≥2 fan-out test (B-002) and the missing status+body handler tests (see R-005), the round's new behaviors are largely unverified.
- Rationale: Violates the plan's own Test Plan (D7: "delivery tests for … durable guidance across reconnect/dropSession, digest coalescing, and honest cost allocation"; "router tests for … subscription precedence") and `agh-test-conventions`. This absence is the proximate cause of B-001…B-005 shipping broken — the invariants are unowned. Per project rule, tests exist to discover bugs; here they would have.
- Suggested fix: Extend the canonical suites (router_test, delivery_test, manager_test) with `t.Run("Should …")` subtests covering subscription precedence, keyword match/miss, mute-exempts-addressed/mentioned, digest fallback on no-match, and (once implemented) digest coalescing + honest cost split. No new standalone files — these invariants belong to the existing network suites.

# Risks

## R-001 — `delivery_structured_body_max_bytes` cap is unenforced (dead knob)

- File: internal/network/delivery.go
- Line: 1137
- Issue: `formatNetworkMessageWithDeliveryMode` base64-encodes the full canonical body (`encodedBody = base64.StdEncoding.EncodeToString(deliveryBody.canonicalBody)`) with no truncation. `DeliveryStructuredBodyMaxBytes` (default 4096) is defined and validated (config.go:471,1757) and surfaced in settings, but never read in `internal/network`. The plan's "structured non-say bodies remain base64 JSON but are capped by network.delivery_structured_body_max_bytes" is not implemented.
- Suggested fix: Enforce the cap on the base64-encoded structured body (with a truncation marker + observability path for canonical bytes, which audit already stores), or delete the config key if the cap is not intended.

## R-002 — Durable guidance store read is held under the delivery coordinator mutex (lock-during-IO)

- File: internal/network/delivery.go
- Line: 514
- Issue: `guidanceModeForDelivery` holds `c.mu.Lock()` while calling `c.guidanceStore.GetNetworkDeliveryGuidanceState` — a synchronous SQLite `QueryRowContext`. The same `c.mu` guards accept/enqueue/`notifyWaiters`/`dropSession`/stats across every session on the daemon. On the first delivery per session (cold cache) a slow/contended global-DB read blocks all inbound accept/enqueue and stats. Bounded (once per session) but a real contention hazard under load.
- Suggested fix: Read the durable guidance state outside the lock, then re-acquire `c.mu` only to memoize the result.

## R-003 — Fan-out partial failure leaves orphaned sibling runs (no rollback)

- File: internal/api/core/tasks.go
- Line: 1213
- Issue: `enqueueFanOutTaskRuns` returns on the first `EnqueueRun` error with already-enqueued sibling runs left persisted, and the handler returns 500. A caller retry re-enqueues (per-designation idempotency keys mean prior successes may re-create under different keys). Result: live orphan runs sharing a `designation_group_id` with no rollup. (Moot until B-002 is fixed, but relevant afterward.)
- Suggested fix: Enqueue the group atomically (single tx) or define + document partial-fan-out semantics and return the partial set with the group id so callers can reconcile.

## R-004 — Status-back per-sibling flooding + rollup completion lacks persistent transition de-dupe

- File: internal/daemon/network_task_status_observer.go
- Line: 179
- Issue: For an N-sibling fan-out, each sibling's run_started/run_completed/run_failed posts a separate status message into the originating thread plus the final aggregate — noisier than the spec's "ONE compact message on meaningful transitions." Separately, `recomputeAndSendDesignationRollup` (232-258) sends the completion message from recomputed state and overwrites the persisted rollup without checking whether the previously-persisted rollup was already `Complete`; any future event replay/reconciliation that re-delivers a terminal event for a fully-terminal group re-sends the completion message. The happy path fires once (single-worker queue serializes), so this is latent.
- Suggested fix: Suppress per-run status posts when `run.DesignationGroupID != ""` (rely on the rollup); gate the completion send on a transition (previously-persisted rollup `Complete == false`), mirroring `bridgeTerminalTaskNotificationObserver`'s de-dupe.

## R-005 — New API handlers have no status+body behavioral tests

- File: internal/api/core/tasks.go
- Line: 1160
- Issue: `internal/api/core` has no subtests exercising `UpdateNetworkChannel`, `NetworkSubscriptions`/`UpsertNetworkSubscription`/`DeleteNetworkSubscription`, `PromoteNetworkThreadTask`, or `FanOutTaskRuns`. Coverage is limited to route-registration parity tables. Per `agh-test-conventions`, new agent-facing endpoints need assertions on status code AND body (e.g. fanout_policy normalization/validation error, the promote 201 body with `origin`, the fan-out 201 body with `designation_group_id`).
- Suggested fix: Add behavioral subtests to the existing `TestBaseHandlersNetworkEndpoints` and task surface suites asserting status + body for each new endpoint (including validation failures).

## R-006 — Subscribing to a channel with no persisted `network_channels` row fails with an opaque FK error

- File: internal/daemon/native_tools.go
- Line: 1776
- Issue: `networkSetSubscription` calls `PutNetworkSubscription` directly; the `network_subscriptions` FK to `network_channels(workspace_id, channel)` (global_db.go:174) is enforced, so subscribing/muting a channel that exists only as ephemeral traffic (or a typo, or not-yet-created channel) fails with a generic `FOREIGN KEY constraint failed` surfaced as an opaque native error. No test covers the missing-channel path.
- Suggested fix: Verify/upsert the channel first and return a typed `ErrNetworkChannelNotFound` (mapped to a clean 404/validation error), and add a subscribe-before-channel-exists test.

# Nits

## N-001 — Router `activationTopK` hardcoded fallback (8) disagrees with config default (3)

- File: internal/network/router.go
- Line: 219
- Issue: `newRouter` defaults `activationTopK` to 8 (router.go:219,234) while config default is 3 and docs state 3; only reachable if `WithRouterActivationTopK` is not called (production wires cfg=3), but confusing.
- Suggested fix: Align the fallback with the config default or drop the literal fallback.

## N-002 — Raw-byte truncation of promotion digest / status text can split a UTF-8 rune

- File: internal/daemon/native_tools.go
- Line: 3022
- Issue: `digest[:4000]` (native_tools.go) and `truncateStatusText` (network_task_status_observer.go:396) cut on byte boundaries, producing invalid UTF-8 mid-rune. `internal/situation/task_context.go:700` already has `truncateUTF8Bytes`.
- Suggested fix: Use rune-safe truncation for the pinned digest and status text.

## N-003 — `NetworkConversationMessagePayloadFromStore` drops enrichment fields populated by the view converter

- File: internal/api/core/network_conversations.go
- Line: 938
- Issue: The `FromStore` converter leaves `DisplayName`, `Local`, and presence fields unset, whereas `NetworkConversationMessagePayloadFromView` populates them; thread/direct timelines will report empty `display_name`/`local=false`.
- Suggested fix: Populate the enrichment fields in `FromStore` or confirm the thread/direct UI does not depend on them.

## N-004 — `PromoteNetworkThreadTask` can leave an orphaned task if the origin-link write fails

- File: internal/api/core/network_conversations.go
- Line: 143
- Issue: The task is created via `CreateTask`, then `writePromotedNetworkThreadOrigin`; if the origin write fails the handler returns 500 but the created task is not removed, leaving a task with no thread-origin link (inconsistent with the paired `Task`+`Origin` response contract).
- Suggested fix: Wrap create + origin write so a failed origin write rolls back the task, or document the create-only-on-partial-failure semantics.

## N-005 — `task_designation_rollups` is currently write-only (no production reader)

- File: internal/store/globaldb/global_db_network_preferences.go
- Line: 344
- Issue: `ListTaskDesignationRollups` has no non-test caller; the rollup is written (FanOutTaskRuns, status observer) but never read back, so the "read rollup" half is unexercised end-to-end.
- Suggested fix: Wire the rollup reader into the status-back/thread projection or confirm it is intentionally deferred.

## N-006 — `networkResponseRegisterPromptSectionProvider.PromptSection` fallback returns the register unconditionally

- File: internal/daemon/network_response_register_prompt.go
- Line: 31
- Issue: The bare `PromptSection` interface method returns the register with an empty channel regardless of predicate; not reached in production (ComposedAssembler prefers the startup-section path and the descriptor predicate gates emission), but a latent footgun if a future caller uses the bare seam.
- Suggested fix: Add a short comment documenting the method is a non-selecting interface stub, or return empty from it.

# Evidence

Read in full: the packet (`.compozy/tasks/network-token-optimization/codex-megaround-packet.txt`) and plan (`.codex/plans/20260701T200023-0300*…megaround.md`); root `CLAUDE.md` and `internal/CLAUDE.md`.

First-hand verification by me (grep + targeted reads):

- internal/network/router.go:1207-1348 (delivery-set computation, activation, no policy branch), :1362-1418 (directed routing, `localPeerMatchesConversation`).
- internal/network/manager.go:1090-1239 (inbound path, `applyDeliverySubscriptions`, `deliverySubscriptionMode` — mention-only exemption).
- internal/network/delivery.go:505-545 (guidance under mutex), :1122-1204 (structured base64 uncapped, single-envelope digest formatter).
- internal/store/globaldb/global_db_task_aux.go:735-775, 1477-1504 (designation-blind open-run guard) and internal/api/core/tasks.go:1195-1240 (`enqueueFanOutTaskRuns` sequential EnqueueRun).
- Config wiring greps confirming `DigestFlushInterval`/`DigestMaxEnvelopes`/`DeliveryStructuredBodyMaxBytes` are defined+validated but unread in `internal/network`; `fanout_policy`/`coordinator_peer_id` present across store/CLI/native/API/contract but absent from routing.
- Test-coverage grep confirming no `internal/network/*_test.go` references the subscription/mute/digest/activation functions; migration registry tail v44–v47 at global_db.go:1168-1186.

Corroborating review-agent passes (read-only, evidence cross-checked against the above): persistence/migrations slice (append-only, workspace-scoped, BEGIN IMMEDIATE, no swallowed errors — clean); network delivery slice; task/fan-out slice (confirmed no parallel queue, `ClaimNextRun` authoritative, `agh.runtime` no impersonation, N ceiling range-validated); API/contract slice (parity matrix PASS, codegen co-ship PASS across openapi/agh.json + web/src/generated/agh-openapi.d.ts + sdk contracts); daemon prompt/register slice (response register wired, budget-enforced, guidance-only — clean); web+docs slice (confirmed the truthful-UI/docs violations for coordinator/all_members routing and digest coalescing, and that the round-1 zero-delivery path was cleanly deleted).

Limitations: I did not run `make verify`, `make codegen-check`, or the test suites (read-only review; the diff is the working-tree state under commit `669f6b503 savepoint`). Findings are static-analysis + cross-referenced reads; B-001, B-002, B-003, B-005, and B-006 were verified by direct source reads, not merely subagent report.

# Deferred Or Follow-Up

- Confirm whether digest coalescing (B-003) is in scope for this round or deferred; if deferred, do the truthful descope (drop dead config keys + fix web/docs/skill copy) rather than leaving aspirational surfaces.
- After B-002 fix, add an integration test enqueuing N=2..5 designated runs through the real store and asserting group rollup fires once at quiescence.
- Re-review round is required after remediation given the blocker count and the "surfaces complete, behavior absent" pattern; do not treat green surfaces/parity as done until the delivery-plane and execution-plane behaviors are tested end-to-end.
- Backend note (out of this diff's fault): `networkChannelEntry` lookup at global_db_task_claim.go:935 filters by `channel` only (not workspace_id); pre-existing, guarded downstream, but worth closing when next touched.
