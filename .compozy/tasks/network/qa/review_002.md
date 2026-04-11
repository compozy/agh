# QA Review 002

Date: 2026-04-11
Environment: isolated rebuilt daemons with dedicated homes under `/tmp/agh-network-rfc-qa-*`, custom HTTP/network ports (`2341+`, `4541+`), provider `codex`
Status: resolved in this run

## Issue 005: SQLite connection pragmas were not applied to pooled connections

Severity: high

Evidence:
- Live runtime on the isolated daemon emitted `observe: write event summary failed` with `database is locked (5) (SQLITE_BUSY)` during network fan-out.
- The root cause was not the network feature itself; it was SQLite connection setup. `sqliteDSN()` opened a pooled DB, but `busy_timeout`, `foreign_keys`, `journal_mode`, and `synchronous` were only set by one-time `PRAGMA` statements after `sql.Open`.
- New pooled connections therefore missed the connection-scoped pragmas and behaved differently under concurrent writes.

Impact:
- Network delivery could trigger intermittent store contention and lost observability writes under real traffic.
- This undermined confidence in audit/history behavior during exactly the protocol fan-out scenarios this task depends on.

## Issue 006: Expired directed envelopes lost protocol context on receive-side rejection

Severity: high

Evidence:
- Receiver-side expiry is an RFC/techspec requirement: expired directed messages should be rejected and may emit `receipt(expired)`.
- The pre-fix router returned a rejection reason from `ParseEnvelope`, but discarded the partial envelope metadata on validation failure.
- As a result, the daemon had no target/session context for auditing the rejection and could not emit a protocol `receipt(expired)`.

Impact:
- Expired directed traffic produced a lower-fidelity local rejection than the protocol/spec expect.
- Operators lost audit context, and initiators did not get the expected receipt signal.

## Issue 007: `whois` responses updated remote cache but never reached the requester

Severity: high

Evidence:
- In real runtime, a broadcast `whois` request generated response envelopes from matching peers, but the requester did not receive the response as an inbound network message.
- Root cause: `handleWhois()` for `WhoisTypeResponse` refreshed remote presence and returned without building a local delivery for the directed requester.

Impact:
- Discovery partially worked internally but failed at the user-facing protocol layer.
- Requesters could not reliably consume `whois` responses, which breaks the discovery contract in RFC 003.

## Issue 008: Control-plane messages bypassed audit completeness

Severity: high

Evidence:
- Manual `greet` sends recorded only a `sent` row; there were no `received` audit rows for the local peers that observed the greet.
- Broadcast `whois` requests likewise recorded only the sender row, even though matching local peers responded.
- Automatic greets emitted by session join / heartbeat / reconnect also bypassed `sent` auditing because they were published through the router heartbeat path, not through `Manager.Send()`.
- The techspec explicitly states that all network messages are recorded in the audit log.

Impact:
- The persisted audit trail was incomplete for discovery/heartbeat traffic.
- That made protocol reconstruction and incident analysis unreliable for exactly the RFC control-plane flows this QA round was meant to certify.

## Resolution Notes

- Issue 005 fixed in `internal/store/sqlite.go` by moving the required PRAGMAs into the SQLite DSN so every pooled connection inherits them.
  - Locked in with stronger assertions in `internal/store/store_extra_test.go`.
  - Live revalidation on a fresh daemon no longer produced `SQLITE_BUSY` / `observe: write event summary failed` warnings during protocol fan-out.

- Issue 006 fixed in `internal/network/router.go`.
  - The router now preserves a partial envelope summary on receive-side validation failure and can emit rejection receipts for recoverable directed failures, including `receipt(expired)`.
  - Locked in with `TestRouterReceiveExpiredDirectGeneratesExpiredReceipt`.

- Issue 007 fixed in `internal/network/router.go`.
  - `whois` responses now both refresh remote presence and deliver to the directed local requester.
  - Locked in with `TestRouterWhoisResponseRefreshesRemotePresenceAndDeliversToRequester`.

- Issue 008 fixed in `internal/network/manager.go`.
  - The manager now audits daemon-generated greets (initial join, periodic heartbeat, reconnect re-greet).
  - The manager also records `received` audit rows for control-plane messages that are consumed by local peers without prompt delivery, specifically `greet` and `whois` request.
  - Locked in with `TestManagerAuditsGeneratedGreetsAndControlReceivers` and updated manager audit coverage.

## Additional QA Coverage

- Real runtime validation on fresh isolated daemons covered:
  - startup and periodic `greet` auditing
  - manual `greet` send with `sent + received` audit rows
  - broadcast `whois` request with `sent + received` request auditing
  - `say` and `recipe` broadcast fan-out
  - `direct -> receipt(accepted) -> trace(working) -> trace(completed)` lifecycle
  - post-terminal `direct` rejection with `interaction_closed`
  - duplicate retry rejection with `duplicate`
  - third-party lifecycle trace ignored after send
  - cross-daemon send rejected locally with `target peer not found`, matching the current v0 out-of-scope multi-daemon design

- Targeted protocol verification covered:
  - subject/route-token helpers and known vectors
  - lifecycle state machine matrix
  - receive-side expiry / replay-window behavior
  - greet expiry and fresh-greet recovery
  - reconnect-triggered re-greet path
  - CLI direct retry/resume integration

- Observability checks:
  - structured logs preserve `reply_to`, `trace_id`, and `causation_id` on lifecycle messages
  - no fresh `SQLITE_BUSY` warning was observed after the SQLite fix
