---
provider: manual
pr:
round: 1
round_created_at: 2026-05-06T02:21:18Z
status: resolved
file: internal/api/core/memory.go
line: 471
severity: high
author: claude-code
provider_ref:
---

# Issue 003: Twenty-two Slice 1 HTTP/UDS routes return memory.unsupported

## Review Comment

The TechSpec API Endpoints section is explicit that every Slice 1 CLI verb has HTTP and UDS parity (`.compozy/tasks/mem-v2/_techspec.md` §API Endpoints, §Agent Manageability Plan):

> "Every memory operation is reachable from CLI, HTTP, and UDS, with structured outputs (`-o json` / `-o jsonl`) and deterministic error contracts. … UI-only manageability is incomplete — the API surface above is the contract."

The shared `BaseHandlers` in `internal/api/core/memory.go` registers 22 production-facing handlers as stubs that always reply with `memoryUnsupportedStatus` via `respondUnsupportedMemoryOperation`:

| Handler | File:line | Route surfaced via the routers |
|---|---|---|
| `PromoteMemory` | `memory.go:471` | `POST /api/memory/promote` (CLI: `agh memory promote`) |
| `ResetMemory` | `memory.go:477` | `POST /api/memory/reset` (CLI: `agh memory reset`) |
| `ReloadMemory` | `memory.go:483` | `POST /api/memory/reload` (CLI: `agh memory reload`) |
| `ListMemoryDecisions` | `memory.go:489` | `GET /api/memory/decisions` (CLI: `agh memory decisions list` via `client.go:2530`) |
| `GetMemoryDecision` | `memory.go:495` | `GET /api/memory/decisions/{id}` |
| `RevertMemoryDecision` | `memory.go:501` | `POST /api/memory/decisions/{id}/revert` |
| `GetMemoryRecallTrace` | `memory.go:507` | `GET /api/memory/recall-traces/{session_id}/{turn_seq}` |
| `ListMemoryDreams` | `memory.go:513` | `GET /api/memory/dreams` |
| `GetMemoryDream` | `memory.go:519` | `GET /api/memory/dreams/{date}` |
| `RetryMemoryDream` | `memory.go:525` | `POST /api/memory/dreams/{run_id}/retry` |
| `ListMemoryDailyLogs` | `memory.go:537` | `GET /api/memory/daily` and family |
| `RetryMemoryExtractor` | `memory.go:573` (when handler is nil) | `POST /api/memory/extractor/retry` |
| `DrainMemoryExtractor` | `memory.go:591` (when handler is nil) | `POST /api/memory/extractor/drain` |
| `SelectMemoryProvider` / `EnableMemoryProvider` / `DisableMemoryProvider` | `memory.go:640,662,685` | `POST /api/memory/providers/*` |
| `CreateMemoryAdhocNote` | `memory.go:709` | `POST /api/memory/adhoc` |
| `GetMemorySessionLedger` | `memory.go:714` | `GET /api/memory/sessions/{id}/ledger` |
| `ReplayMemorySession` | `memory.go:727` | `GET /api/memory/sessions/{id}/replay` |
| `PruneMemorySessions` | `memory.go:745` | `POST /api/memory/sessions/prune` |
| `RepairMemorySessions` | `memory.go:763` | `POST /api/memory/sessions/repair` |

Each stub explicitly admits to unfinished Slice 1 work — comments such as `"task 19 wires the daemon-owned decision query service into API core"` (`memory.go:487-489`, repeated for `GetMemoryDecision`, `RevertMemoryDecision`, etc.). Task 19 is marked completed in `state.yaml`. The QA verification report claims TC-INT-005 covered "HTTP, UDS, OpenAPI/codegen, and full `make verify`" (`qa/verification-report.md`), but a code search for `memory/decisions`, `ListMemoryDecisions`, and `RevertMemoryDecision` in `internal/daemon/daemon_memory_e2e_integration_test.go` finds zero exercise of those routes. The CLI client (`internal/cli/client.go:2530-2575`) genuinely calls these endpoints, so any operator running `agh memory decisions list` or `agh memory decisions revert <id>` against the daemon will receive the deterministic-error `memory.unsupported` payload instead of the documented response.

This violates the §Agent Manageability Plan ("CLI / HTTP / Native tool" parity rows for `decisions list`, `decisions show`, `decisions revert`, `recall trace`, `dream show`, `dream retry`, `dream trigger`, `daily *`, `sessions *`, `extractor *`, `provider *`, `adhoc *`), the §Hard-cut public surface row that promotes `agh memory consolidate` → `agh memory dream trigger`, and the standing directive "agent operations must not depend on the web UI".

Suggested fix:

- Wire the daemon-owned services already implemented in `internal/memory/decision.go` (`Store.ListDecisions`, `Store.LoadDecision`, `Store.RevertDecision`), `internal/memory/dream*.go`, `internal/sessions/ledger`, and the provider registry into `BaseHandlers`. Pattern after `GetMemoryExtractorStatus` (`memory.go:541`), which already handles a nil-injected service correctly.
- Add at minimum these gating tests so the regression cannot reappear: `TestHandlers_MemoryDecisionsListReturnsRows`, `TestHandlers_MemoryDecisionsRevertCallsService`, `TestHandlers_PromoteCopiesByDefault`, `TestHandlers_ReloadInvalidatesNextBoot`, `TestHandlers_DailyListReturnsLogs`. They should run against `make test-e2e-runtime` so the CLI/daemon round-trip is verified.
- Until the wiring lands, downgrade the CLI client to surface the 501 path with a clear "not yet wired" hint, OR remove the stubs from the registered routes so callers see a 404 instead of pretending the route exists.

## Triage

- Decision: `VALID`
- Root cause: the issue over-counts a few service-backed routes whose nil-service fallback is intentionally deterministic, but the core finding is correct: several Slice 1 HTTP/UDS handlers remained unconditional `memory.unsupported` placeholders after task 19 was marked complete. In particular promote/reset/reload, decision list/show/revert, recall trace, dreaming list/show/retry, daily list, and ad-hoc note creation were still registered but not backed by Memory v2 store behavior.
- Fix approach: replace those placeholders with store-backed handlers where state exists, use truthful 404/empty responses where Slice 1 has no materialized record, keep service-backed extractor/provider/session routes unchanged, and add focused API tests for promote, decisions, reset/reload, daily, dreams, and ad-hoc behavior.

## Resolution

- Replaced Slice 1 API placeholders with store-backed handlers for promote, derived reset/reload, decision list/show/revert, dream list/show/retry, daily logs, and ad-hoc notes; recall traces now return a truthful not-found response because traces are not materialized.
- Added query helpers for decision, dream, and daily log records plus focused API tests for the newly wired routes.
- Verification: `go test ./internal/api/core -run 'TestMemoryHandlersAndHelpers|TestMemoryExtractorHandlersUseInjectedService|TestMemoryProviderHandlersUseInjectedService|TestMemorySessionLedgerHandlersUseInjectedService' -count=1` passed; `make verify` passed with Bun 334 files / 2150 tests, Go `DONE 8393 tests in 90.274s`, and boundaries OK.
