---
status: completed
title: "Delivery robustness: inbound edits, reply context, and durable ledger"
type: backend
complexity: high
---

# Task 6: Delivery robustness: inbound edits, reply context, and durable ledger

## Overview
Completes the inbound contract (edit family + reply-to context) and persists per-turn
deliveries in a durable ledger with boot reconciliation. Two gaps lose user intent today —
platform edits are dropped and threaded replies lose parent text — while a mid-turn daemon
restart silently orphans in-memory Path A deliveries. This slice closes both so agents see
edits/reply context and channels never stay half-streamed after restart.

<critical>
- ALWAYS READ the PRD, the TechSpec, and their catalogs (`_user_stories.md`, `_tests.md`) before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>
- MUST add `InboundEventFamilyEdit` with a typed payload (edited message id, new text, original
  timestamp) and mutual-exclusion validation consistent with the existing families.
- MUST route Slack `message_changed`/`message_deleted` subtypes and Telegram
  `edited_message`/`edited_channel_post` into the new family; Discord message-edit events where
  the interactions surface provides them (UNCONFIRMED — verify during implementation; skip with
  evidence if the webhook lacks it).
- MUST add optional `ReplyToText`/`ReplyToAuthorID`/`ReplyToAuthorName` to
  `InboundMessageEnvelope`, filled by Slack/Telegram/GChat providers on threaded replies (with a
  bounded per-provider parent-text cache; cache miss degrades to empty fields — no fetch storm).
- MUST render both in the inbound prompt so the agent sees "user edited …" and the replied-to
  context.
- MUST co-ship contract codegen (OpenAPI + TS types + E2E mocks) for the new family and fields
  (`agh-contract-codegen-coship`).
- MUST add a `bridge_deliveries` table via a numbered, append-only migration (activate
  `agh-schema-migration`; never touch existing migrations — L-021): delivery id, session/turn,
  route key, instance id, `scope` + `workspace_id` (indexed — direct workspace-scoped queries
  without an instance join, matching the `bridge_task_subscriptions` precedent), state, last
  sent/acked seq, `remote_message_id`, timestamps, terminal error.
- MUST checkpoint on registration, ack advance, and terminal transitions — not per delta
  (write-amplification guard). Checkpoint-only v1: MUST NOT persist message content beyond what
  reconciliation needs; progress events are NOT checkpointed (text stream only).
- MUST reconcile on boot: unfinished deliveries either resume via the existing `resume`
  machinery (adapter reconciles by `remote_message_id`) or fail-open with the standard
  "session stopped" terminal error to the channel — never silently drop. Boot reconcile runs
  from the composition root (`internal/daemon`) BEFORE new registrations (inv5, SD-008).
- MUST make `DeliveryMetrics` counters durable (persisted alongside or derivable from the
  ledger).
- MUST carry `scope`/`workspace_id` on every row with an index so workspace A never lists or
  reconciles workspace B's rows.
</requirements>

## Subtasks
- [x] 6.1 Contract: `InboundEventFamilyEdit` + envelope reply-to fields + mutual-exclusion
      validation + OpenAPI/TS/E2E mock codegen co-ship
- [x] 6.2 Slack provider: `message_changed`/`message_deleted` subtype routing + bounded
      thread-parent text cache for reply-to fields
- [x] 6.3 Telegram provider: `edited_message`/`edited_channel_post` routing + reply context
- [x] 6.4 GChat reply context; Discord edit routing where available (verify surface; skip with
      evidence if absent)
- [x] 6.5 Prompt renderer emits distinct blocks for edits and reply-to context
- [x] 6.6 Migration + globaldb store methods for `bridge_deliveries` (append-only, indexed
      `scope`/`workspace_id`)
- [x] 6.7 Broker checkpoints on register/ack/terminal; write-amplification guard (no per-delta
      writes)
- [x] 6.8 Boot reconciliation from `internal/daemon`: resume-or-fail-open before new
      registrations; adapter reconcile via existing `FindDeliveryMessage`-style paths
- [x] 6.9 Durable `DeliveryMetrics` counters surviving restart

## Implementation Details
Reference `_techspec.md` §3 (D9 row), §3.5 (durable ledger), and Open Decision #5
(checkpoint-only v1). Prompt rendering extends the inbound family renderer in
`internal/extension/host_api_bridges.go`. Path B cursor durability
(`bridge_task_subscriptions` / task notifier) is the in-repo pattern to mirror — Hermes has no
delivery durability. Contract change → activate `agh-contract-codegen-coship`. Schema change →
activate `agh-schema-migration`. Skills: `agh-code-guidelines`, `golang-pro`,
`agh-test-conventions`, `consolidate-test-suites`, `agh-cleanup-failure-paths`.

### Relevant Files
- `internal/bridges/types.go` — `InboundEventFamilyEdit` + reply-to envelope fields
- `internal/extension/host_api_bridges.go` — family validation + inbound prompt rendering
- `extensions/bridges/{slack,telegram,gchat,discord}/provider.go` — edit routing + parent-text
  caches
- `internal/store/globaldb/` — numbered `bridge_deliveries` migration + queries
- `internal/bridges/delivery_broker.go` — checkpoint hooks on register/ack/terminal
- `internal/daemon/bridges.go` — boot reconciliation wiring (composition root)

### Dependent Files
- Generated OpenAPI/TS artifacts + E2E contract mocks (codegen co-ship)
- `internal/bridges/types_test.go`, `delivery_broker_test.go`
- `extensions/bridges/{slack,telegram}/provider_test.go`
- `internal/extension/` host_api ingest suite + `bridge_delivery_integration_test.go`
- `internal/daemon/` boot tests; `packages/site` `routing.mdx` families table (docs note —
  full docs parity is task_08)

### Competitor References
- `.resources/hermes/plugins/platforms/slack/adapter.py:73,2619` — thread-context cache +
  subtype handling
- `.resources/hermes/gateway/platforms/base.py:1716` — `reply_to_text` on normalized events
- In-repo Path B: `internal/bridges/task_notifier.go` reconciliation guard (Hermes lacks
  delivery durability)

## Deliverables
- `edit` family + reply-to context live end-to-end for Slack/Telegram (+ GChat reply context)
- Durable `bridge_deliveries` checkpoints + boot reconciliation + persistent metrics
- Unit, integration, and E2E cases assigned below implemented and passing **(REQUIRED)**
- Contract codegen co-ship green; migration append-only proof green

## Tests

Cases assigned from `_tests.md` (remap: old task_12+13 → this task). Read full definitions
there before writing tests.

- Unit tests (suite: `internal/bridges/types_test.go` — `_tests.md` §1 case 21):
  - [x] `edit` family payload validates (edited message id, new text, original timestamp);
        mutual exclusion with message/command/action/reaction enforced; envelope with
        `ReplyToText`/`ReplyToAuthorID`/`ReplyToAuthorName` round-trips JSON (D9)
- Unit tests (suite: `internal/bridges/delivery_broker_test.go` — `_tests.md` §1 cases 22–23):
  - [x] Register → checkpoint row exists with state `active`; ack advance updates seq +
        `remote_message_id`; terminal marks the row terminal; delta storms produce ZERO
        per-delta writes (store call-count guard; inv4)
  - [x] `DeliveryMetrics` counters are durable — persisted alongside or exactly derivable from
        ledger rows (D8)
- Unit tests (suite: `extensions/bridges/slack/provider_test.go` — `_tests.md` §4 cases 18–19):
  - [x] `message_changed` webhook → envelope with family `edit`, new text, original message id
  - [x] Threaded reply → envelope carries parent text/author from the cache; cache miss
        degrades to empty fields (no fetch storm)
- Unit tests (suite: `extensions/bridges/telegram/provider_test.go` — `_tests.md` §4 case 20):
  - [x] `edited_message` update → `edit` family envelope
- Unit tests (suite: `internal/store/globaldb/` canonical migration suite — `_tests.md` §5
  case 1):
  - [x] `bridge_deliveries` migration applies at the registry tail; checksum chain intact
        (append-only proof; L-021)
- Integration tests (suite: `internal/extension` host_api ingest +
  `bridge_delivery_integration_test.go` + `internal/daemon` boot — `_tests.md` §5 cases
  12–15, 19):
  - [x] Ingested `edit` envelope renders an "edited" prompt block; reply context appears in the
        rendered prompt
  - [x] Kill the broker mid-stream (fresh broker over the same store): unfinished delivery
        resumes to the same `remote_message_id` OR fails open with the standard terminal error
        — channel never left half-streamed silently (inv4)
  - [x] Metrics survive the restart
  - [x] Boot reconciliation completes BEFORE the broker accepts new delivery registrations
        (inv5)
  - [x] `bridge_deliveries` isolated by `scope`/`workspace_id`: workspace A never lists/queries
        workspace B's rows through metrics or reconcile paths
- E2E tests (lane: `make test-e2e-runtime` bridge lane — `_tests.md` §6 cases 5–6):
  - [x] Contract mocks updated for the inbound `edit` family + reply-to fields (codegen
        co-ship green)
  - [x] Daemon restart scenario exercises boot reconciliation: resume-or-fail-open observed at
        the fake adapter
- Test coverage target: >=80% for touched packages
- All tests must pass under the repo gates (`-race` for Go)

## Success Criteria
- Every assigned test case implemented and passing
- A user editing their Slack/Telegram message produces an agent-visible edit prompt; a
  threaded reply carries what it replied to
- Simulated mid-turn restart leaves zero orphaned in-flight deliveries and a consistent
  channel state (resume or explicit terminal error — never silent half-stream)
- Coverage >=80% on touched packages; `make lint` + scoped `go test -race` green
