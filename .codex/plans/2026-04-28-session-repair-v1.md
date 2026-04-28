# Session Repair V1: Interrupted Transcript Terminalization

## Summary

Implement a focused session repair feature that fixes the root cause: daemon crashes can leave a persisted session transcript with an incomplete final turn, causing AI SDK replay parts and tool calls to remain permanently "streaming" even after the session is stopped.

The v1 repair is append-only. It provides automatic boot repair for crashed/error sessions and explicit agent-manageable repair via HTTP, UDS, and CLI with dry-run support.

## Key Changes

- Add `session.RepairSession(ctx, opts)` in `internal/session`.
- Add stable repair types:
  - `SessionRepairOpts{SessionID, DryRun, Force}`
  - `SessionRepairResult{SessionID, Issues, Actions, Persisted}`
  - `SessionRepairIssue{Code, Severity, TurnID, Detail}`
  - `SessionRepairAction{Code, TurnID, EventID, Persisted}`
- Repair behavior:
  - Append a canonical `error` event when the final persisted turn lacks `done` or `error` and the session is stopped with `agent_crashed` or `error`.
  - Append interrupted synthetic `tool_result` events before the terminal `error` when the interrupted turn has dangling `tool_call` events.
  - Report invalid JSON, corrupt metadata, event DB failures, sequence anomalies, and lineage concerns without destructive mutation.
  - Treat sequence gaps as diagnostics only; do not truncate or resequence.
- Boot integration:
  - Run automatic repair during boot for stopped sessions with `agent_crashed` or `error`.
  - Log structured summary counts.
- Public surfaces:
  - `POST /api/sessions/:id/repair?dry_run=true&force=false` for HTTP and UDS.
  - `agh session repair <id> --dry-run --force -o json`.
  - Add OpenAPI/contract `SessionRepairResponse`.

## Test Plan

- Unit tests in `internal/session` for terminal event repair, dangling tool repair, dry-run, consistency no-op, force behavior, invalid JSON, corrupt metadata, and sequence diagnostics.
- Transcript/store tests proving repaired UI messages are no longer stuck in streaming states and appended repair events preserve monotonic sequence ordering.
- HTTP/UDS/CLI tests for repair endpoint/command and error mapping.
- Boot/integration tests proving automatic repair is idempotent after daemon restart.
- Verification: focused Go tests, `make codegen`, `make codegen-check`, web type/test gates if generated types change, and full `make verify`.

## Assumptions

- Default repair mode is Boot + manual.
- No schema migration is required for v1.
- No truncation, resequencing, row deletion, UI-only state coercion, lint suppression, ignored errors, sleeps, or retry loops are acceptable.
