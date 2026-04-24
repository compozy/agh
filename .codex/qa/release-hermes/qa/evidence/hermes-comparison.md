# Hermes vs AGH Production-Grade Comparison

**Date:** 2026-04-24
**Purpose:** Identify release-critical AGH gaps by comparing with `.resources/hermes`.

## Production-grade traits found in Hermes

1. **Persistent background work registry**
   - Hermes tracks long-running/background processes with checkpoint recovery, output buffers, completion queues, watcher metadata, and crash recovery.
   - Relevant source: `.resources/hermes/tools/process_registry.py`, `.resources/hermes/gateway/run.py`.

2. **Inactivity-aware runtime supervision**
   - Hermes distinguishes active work from idle agents, drains running agents during shutdown/restart, and queues follow-up input instead of interrupting active work.
   - Relevant source: `.resources/hermes/gateway/run.py`, release notes for inactivity timeouts and shutdown drain.

3. **Retry/backoff hardening**
   - Hermes avoids tight retry loops in gateway reconnects/API operations and uses retry windows/backoff for transient failures.
   - Relevant source: `.resources/hermes/gateway/run.py`, `.resources/hermes/hermes_state.py`, release notes.

4. **Durable state and observability**
   - Hermes uses SQLite WAL, write retry, checkpoints, message/session persistence, safe query fallback, and operator-visible status.
   - Relevant source: `.resources/hermes/hermes_state.py`, `.resources/hermes/gateway/status.py`.

5. **Message/session safety**
   - Hermes has deterministic session keys, reset safeguards, platform-specific redaction, duplicate-delivery prevention and bounded caches.
   - Relevant source: `.resources/hermes/gateway/session.py`, `.resources/hermes/gateway/run.py`.

## AGH coverage and actions

| Production need                         | AGH coverage                                                                                                             | Release disposition                                                                     |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------- |
| Persistent async work and owner reentry | Detached harness metadata, task runtime state, reentry bridge, synthetic prompts.                                        | Covered by existing runtime design; validate through integration/e2e.                   |
| Network routing and delivery            | `internal/network` router, transport, peer registry, delivery coordinator, audit writer.                                 | Covered; expanded delivery retry hardening.                                             |
| Retry/backoff for transient failures    | NATS reconnects, bridge retry policies, extension restart backoff existed; inbound network delivery was immediate retry. | Fixed in this session: scheduled exponential capped retry after failed `PromptNetwork`. |
| Durable audit/timeline                  | Global DB network audit and timeline stores, JSONL audit sink, duplicate message-id ignore.                              | Covered; validate through store/API/e2e.                                                |
| Web/operator visibility                 | Network status/channel/peer/timeline UI and e2e selectors/artifacts exist.                                               | Covered; validate through web e2e/browser.                                              |
| Real provider confidence                | ACP-compatible provider execution depends on local binaries/credentials.                                                 | Validate if available; otherwise record blocker and rely on runtime/e2e harness.        |

## Code change made

- `internal/network/delivery.go`
  - Added retry attempt tracking per queued envelope.
  - Added exponential retry delay capped at 5 seconds.
  - Added scheduled retry after worker exit, instead of immediate worker restart.
  - Added retry attempt logging.

- `internal/network/delivery_test.go`
  - Changed prompt-failure regression to prove retry is scheduled and not executed until the scheduler fires.
  - Added retry delay cap coverage.

## Current result

- `go test ./internal/network` passes.
- Full `make test-integration` passes.
- Full runtime E2E passes.
- Full daemon-served web/browser E2E passes, including Network route.
- Final `make verify` passes.
- Real Codex/OpenAI smoke passes, including AGH daemon + Codex ACP + network direct delivery evidence.
