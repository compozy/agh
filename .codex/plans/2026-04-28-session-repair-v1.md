# Session Repair V1: Interrupted Transcript Terminalization

## Summary

Implement a focused session repair feature that fixes the root cause: daemon crashes can leave a persisted session transcript with an incomplete final turn, causing AI SDK replay parts and tool calls to remain permanently "streaming" even after the session is stopped.

The v1 repair is append-only. It provides automatic boot repair for crashed/error sessions and explicit agent-manageable repair via HTTP, UDS, and CLI with dry-run support.

## Key Changes

- Add `session.RepairSession(ctx, opts)` in `internal/session`.
- Add stable repair types:
  - `RepairOpts{SessionID, DryRun, Force}`
  - `RepairResult{SessionID, Issues, Actions, Persisted}`
  - `RepairIssue{Code, Severity, TurnID, Detail}`
  - `RepairAction{Code, TurnID, EventID, Persisted}`
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

## Web/Docs Impact

- `web/`:
  - `web/src/generated/agh-openapi.d.ts` — regenerated `repairSession` operation and payload types from OpenAPI.
  - `web/src/systems/session/types.ts` — session repair response/query aliases derived from generated contract types.
  - `web/src/systems/session/adapters/session-api.ts` — `repairSession` client for `POST /api/sessions/{id}/repair`.
  - `web/src/systems/session/hooks/use-session-actions.ts` — `useRepairSession` mutation invalidating session detail, history, transcript, events, and lists.
  - `web/src/systems/session/mocks/fixtures.ts` and `web/src/systems/session/mocks/handlers.ts` — MSW fixture/handler for session repair.
  - No route/component UI change — checked `web/src/routes/_app/**` and `web/src/systems/session/components/**`; v1 is intentionally agent/operator-manageable via CLI/HTTP/UDS, not a visible web control.
- `packages/site`:
  - `packages/site/content/runtime/cli-reference/session/repair.mdx` — generated CLI reference for `agh session repair`.
  - `packages/site/content/runtime/cli-reference/session/index.mdx` and `packages/site/content/runtime/cli-reference/session/meta.json` — generated CLI navigation updates.
  - `packages/site/content/runtime/core/sessions/lifecycle.mdx` — conceptual crash/transcript repair behavior.
  - `packages/site/content/runtime/core/sessions/resume.mdx` — resume flow mentions append-only transcript repair before replay.
  - `packages/site/content/runtime/core/operations/troubleshooting.mdx` — operator runbook includes dry-run/manual repair.
  - `packages/site/content/runtime/api-reference/index.mdx` — no direct file edit; OpenAPI-backed API reference receives `repairSession` via `openapi/agh.json`.

## Extensibility / Agent Manageability / Config Lifecycle

- `Extensibility`:
  - none — checked extension manifests, hooks, skills/capabilities, tools/resources, bundles, registries, bridge SDKs, MCP sidecars, and protocol docs; repair is a session persistence operation and does not add an extension point or protocol surface.
- `Agent manageability`:
  - CLI: `agh session repair <id> --dry-run --force -o json`.
  - HTTP: `POST /api/sessions/{id}/repair?dry_run=true&force=false`.
  - UDS: same route and payload as HTTP.
  - Structured output: `SessionRepairResponse` with `issues`, `actions`, `persisted`, `tool_call_id`, and `tool_name`.
  - Error contracts: 400 for invalid repair options, 404 for unknown sessions, non-persisting diagnostics for invalid transcript/event conditions.
- `Config lifecycle`:
  - none — checked `config.toml` keys/defaults, structs, merge/overlay behavior, validation, examples, docs, and tests; v1 adds no configuration.

## Assumptions

- Default repair mode is Boot + manual.
- No schema migration is required for v1.
- No truncation, resequencing, row deletion, UI-only state coercion, lint suppression, ignored errors, sleeps, or retry loops are acceptable.
