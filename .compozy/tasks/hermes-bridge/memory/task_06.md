# Task Memory: task_06

## Objective Snapshot

- Complete inbound bridge semantics with typed edit/delete events and bounded reply-parent context, then make Path A deliveries restart-safe through a workspace-isolated durable checkpoint ledger, boot reconciliation, and durable delivery metrics.

## Preflight Decisions

- The current 10-task graph, `task_06.md`, the approved TechSpec, `_tests.md`, and the realized Task 01-03 memories are authoritative. Historical 18-task review references are provenance only. No PRD or user-story catalog exists in this workflow corpus.
- `InboundEventFamilyEdit` remains the wire family required by the TechSpec. Its typed payload must identify the affected platform message, the replacement text, the original timestamp, and an explicit operation so Slack deletion is represented truthfully without prose sentinels or hidden provider metadata. Empty replacement text is valid only for the delete operation.
- Slack `message_changed` and `message_deleted`, plus Telegram `edited_message` and `edited_channel_post`, are requirements even where the numbered test catalog names only a subset. Assigned cases are the minimum, not permission to omit task requirements.
- Discord edits are explicitly unsupported in this task. The in-tree adapter consumes HTTP Interactions and Application Webhook Events only; `MESSAGE_UPDATE` is a Gateway event and the adapter has no Gateway connection/intents surface. Do not add an unreachable mapper.
- Reply context is cache-only. Slack, Telegram, and Google Chat may populate a bounded provider-local parent cache from already observed webhook payloads/events. A miss returns empty reply fields and must never trigger an on-demand provider fetch.
- The required singular `remote_message_id` remains in `bridge_deliveries`. Task 02's later multichunk contract adds ordered acknowledged handles and a partial-success cursor as normalized checkpoint data, without persisting message text, progress payloads, or a full-text audit trail. Unsafe replay must fail open rather than duplicate an acknowledged prefix.
- Delivery metrics need their own durable per-instance aggregate because terminal delivery rows may later be garbage-collected. Persisted metrics carry direct `scope`/`workspace_id` ownership and canonical redacted error text; backlog remains live/derivable from active rows.
- The durable store interface belongs in `internal/bridges`, where the broker consumes it; `internal/store/globaldb` implements it; only `internal/daemon` composes broker, store, and transport.
- The user override remains active: Task 06 uses only focused race tests, focused lint/vet, required codegen checks, and exact runtime scenarios that cover the changed behavior. Do not run broad package/global suites merely for convenience. Global test targets and the single `make verify` remain reserved for the workflow tail after all tasks and review remediation.

## Test Placement

- Invariant: each inbound family has one typed payload and all other family payloads are absent. Owning layer: bridge domain contract. Canonical suite: `internal/bridges/types_test.go`.
- Invariant: registration, textual ACK advance, and terminal transition are the only delivery-ledger checkpoint boundaries; delta/progress storms do not write. Owning layer: broker. Canonical suite: `internal/bridges/delivery_broker_test.go`.
- Invariant: the append-only migration tail creates queryable workspace-owned delivery and metrics state with an intact identity chain. Owning layer: GlobalDB schema/store. Canonical suites: the existing global migration suite plus bridge persistence tests under `internal/store/globaldb`.
- Invariant: provider-native edit/delete inputs and cached reply parents normalize without provider fetches. Owning layer: each provider mapper. Canonical suites: existing Slack, Telegram, and Google Chat `provider_test.go` files.
- Invariant: edit and reply context are rendered distinctly into the agent prompt. Owning layer: Extension Host ingest/render. Canonical suite: `internal/extension/host_api_bridges_render_test.go` and the existing bridge ingest integration owner.
- Invariant: persisted active deliveries are reconciled before registrations become admissible. Owning layer: daemon composition root. Canonical suite: existing bridge runtime/boot tests plus the existing daemon bridge E2E fixture.

## Important Decisions

- Do not grow existing over-cap `internal/bridges/types.go` or `internal/extension/host_api_bridges.go`. Extract inbound family/payload validation and prompt rendering into single-purpose files, shrinking the original owners in the same change.
- Store only routing/reconciliation metadata and checkpoint cursors. The fail-open terminal content is generated from the standard session-stopped message at reconciliation time; prior streamed text is never copied into the ledger.
- Store calls must not occur while holding the broker mutex. Broker state transitions produce immutable checkpoint/metric mutations under lock, then apply them synchronously after unlock so registration and ACK durability are truthful without blocking unrelated broker state.
- Boot recovery deliberately chooses the task's allowed fail-open branch for every provider: post a new standard terminal error, do not attempt to reconstruct/resume streamed text or depend on a stale remote anchor.

## Learnings

- Existing provider resume requests require a full in-memory `DeliverySnapshot`; the checkpoint-only ledger deliberately cannot recreate message content. Safe boot recovery therefore uses a dedicated persisted-record reconcile path that universally posts the standard visible terminal error without depending on a potentially stale/deleted anchor. This also covers append-only providers without persisting prior message content.
- Existing `DeliveryMetrics` includes dropped-by-reason, failure count, last error/time, and last success time. These values are not exactly derivable from the minimum delivery row schema, so a durable aggregate is required.
- Existing `RoutingKey.Serialize()` already gives a stable JSON representation with direct scope/workspace/instance/peer/thread/group identity. `DeliveryTarget` still carries mode separately, so reconciliation persistence must retain the target shape rather than guessing it from the routing key.

## Files / Surfaces

- `internal/bridges/`: inbound contract split, delivery checkpoint/store contract, broker persistence hooks, metrics loading/persistence, reconciliation API, canonical tests.
- `internal/store/globaldb/`: append-only migration after current v81, schema helpers, delivery/metric store methods, migration and isolation tests.
- `internal/daemon/`: durable broker composition and boot-order reconciliation before registration admission.
- `internal/extension/`: inbound prompt edit/reply rendering and SDK contract roots.
- `extensions/bridges/{slack,telegram,gchat}/`: typed edit/delete mapping and bounded reply caches; Discord evidence-only skip.
- Generated SDK TypeScript contracts and exact bridge E2E fixtures; OpenAPI changes only if the actual REST source contract exposes these inbound fields.
- `packages/site` routing family note and `skills/agh/` runtime reference if public behavior requires it.
- `docs/qa/scenarios/`: flag new/changed user-visible edit, reply-context, and restart-recovery behavior as `untested`; do not run QA here. Never edit or commit the generated `state.csv` view.

## Errors / Corrections

- RESOLVED: daemon boot runs extension startup, exact-scope metric hydration, active-row reconciliation, and only then opens broker admission; composed and real-restart E2E tests prove the order.
- RESOLVED: a deslop/file-cap audit found the new admission interface had grown the pre-existing 2,700-line `host_api.go`. Both delivery contracts now live in `host_api_bridges_delivery_contract.go`; the legacy owner shrank and the focused Host API race/lint lane stayed green.
- OpenAPI has no bridge-ingest REST operation, so adding the Host API envelope there would be untruthful. Canonical codegen still ran once and `make codegen-check` is green; TypeScript Host API contracts and the ACP E2E fixture carry the new wire shape.

## Completion Evidence

- Focused `-race` lanes passed for inbound/domain/broker/cache, GlobalDB migration/store/restart/isolation, Host API render/admission, Slack/Telegram/GChat mapping, the Telegram reference adapter, fresh-broker integration, daemon admission, and the two exact daemon E2Es.
- `make codegen-check`, TypeScript SDK Host API test/typecheck, site typecheck, QA CSV validation, gofmt/diff checks, and scoped/diff-aware golangci-lint passed. Global suites and `make verify` remain intentionally deferred to the workflow tail by user direction.
- QA scenarios `NB-bridge-edit-reply` and `NB-bridge-restart-recovery` are `untested`; no QA execution occurred.
- AGH Impact Audit: native tool IDs/toolsets/descriptors are unchanged; Host API JSON-RPC/SDK and bridge extensions changed; reply caches and ledger/metrics are exact workspace/instance/conversation scoped; the official `skills/agh/` runtime reference is updated; no hooks, bundles, bridge config keys, or `config.toml` defaults changed.
