# AGH First Release Network QA Plan

**Date:** 2026-04-24
**Scope:** AGH daemon, CLI/API network workflows, web network surface, runtime/harness reentry, release verification.
**Status:** Drafted before execution.

## Objective

Validate that AGH is operationally release-ready as a local-first Agent OS, with special emphasis on the network feature. The plan compares production-grade behavior observed in `.resources/hermes` against current AGH and turns release risks into executable checks.

## Release-Critical Areas

| Area                           | Why it matters                                                                                | Primary evidence                                                               |
| ------------------------------ | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Network message routing        | Agents must exchange directed/broadcast/control messages safely and deterministically.        | `internal/network` unit/integration tests, daemon e2e lanes, live CLI/API flow |
| Network delivery resilience    | Failed agent prompts must not drop messages or spin retry loops.                              | `internal/network/delivery_test.go`, runtime stats/audit                       |
| Audit and timeline persistence | Operators need durable visibility into sent/received/rejected/delivered network events.       | global DB tests, API timeline checks, UI reload checks                         |
| Task ingress and reentry       | Network-originated detached work must reconnect to the owning session after async completion. | daemon task runtime tests and e2e harness                                      |
| Web network operations         | The operator UI must reflect channels, peers, timeline and reload continuity.                 | Playwright/browser test with screenshots                                       |
| Full release gate              | Formatting, lint, race tests, web tests/build and Go build must all pass.                     | `make verify`                                                                  |

## Hermes Comparison Summary

| Hermes production-grade behavior                                                                  | AGH equivalent                                                                                          | Release action                                                                                 |
| ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Background work/process registry with persisted recovery checkpoint and completion notifications. | Detached harness run metadata plus reentry bridge and synthetic prompts.                                | Validate with daemon task/runtime e2e and restart/reentry scenarios.                           |
| Inactivity-aware runtime timeouts and graceful drain instead of wall-clock cancellation.          | Session/daemon shutdown and tracked agent processes; network delivery drains through turn-end notifier. | Validate no network prompt interrupts active turn; inspect shutdown logs for pending messages. |
| Jitter/backoff for retries and reconnects to avoid hot loops.                                     | NATS reconnect handlers exist; inbound delivery retry previously restarted immediately.                 | Fixed: delivery failures now schedule exponential capped retry; unit regression added.         |
| Durable platform message logs and dedupe safeguards.                                              | Network audit log and timeline DB with duplicate message-id ignore and router replay checks.            | Validate audit/timeline after direct, whois, rejected and delivered flows.                     |
| Operator diagnostics for background status and platform health.                                   | `NetworkStatus`, queued/inflight/worker metrics, API/CLI/web surfaces.                                  | Validate status includes queue/worker counters and channel details.                            |

## Execution Order

1. Smoke gate: `go test ./internal/network`, then `make verify`.
2. Integration gate: `make test-integration` for tagged daemon/network/store scenarios.
3. E2E runtime gate: `make test-e2e-runtime` for real daemon/harness flows.
4. E2E web gate: `make test-e2e-web` plus browser inspection of network pages where available.
5. Real LLM gate: detect configured providers without printing secrets; run a small network-capable real-agent flow if credentials and agent binaries are available.
6. Evidence report: record commands, results, issues, screenshots/log paths, and remaining risk.

## Pass/Fail Criteria

- P0 tests must pass: network routing/delivery, audit/timeline, task reentry, release verify.
- Any data loss, unbounded retry, message misdelivery, unaudited accepted message, or UI inability to operate network channels blocks release.
- Real LLM validation may be marked blocked only if local credentials or provider binaries are unavailable; the mocked/runtime/e2e gates still must pass.
