# AGH Release QA Report

**Date:** 2026-04-24
**Scope:** Hermes production-grade comparison, AGH release hardening, full verification, network-focused QA, browser E2E, and real LLM smoke.
**Result:** Pass with one documented LLM-behavior caveat.

## Executive Summary

AGH is release-ready from the tested surfaces. The Hermes comparison identified one release-critical reliability gap in AGH: inbound network delivery retried immediately after `PromptNetwork` failure, which could create a tight loop and hide transient delivery failures. The implementation now schedules exponential capped retry after worker exit, preserves queued delivery state, and has deterministic regression tests.

QA also found and fixed stale integration contracts and runtime harness weaknesses that would have reduced release confidence: prompt disconnect draining, stop route parity, provider override resume semantics, recovered crash classification, runtime harness readiness, UDS socket collisions, and stale Playwright selectors/routes.

## Production Hardening Applied

- Added scheduled exponential retry/backoff for failed inbound network delivery.
- Preserved network queued-envelope attempt state and retry logging.
- Fixed HTTP/UDS prompt disconnect handling so client disconnects return immediately while agent turns drain to terminal persistence.
- Fixed HTTP/UDS transport parity tests to use the current stop route.
- Preserved recovered `agent_crashed` classification through missing ACP-state fallback.
- Fixed provider resolution so persisted provider values matching the effective provider preserve custom agent command/model on resume.
- Hardened runtime harness process-exit readiness and UDS socket collision recovery.
- Updated web E2E contracts for current automation, bridge, task, settings, Storybook, session, and network UI surfaces.

## Verification Evidence

| Area                           |           Result | Evidence                                                                                                                                |
| ------------------------------ | ---------------: | --------------------------------------------------------------------------------------------------------------------------------------- |
| Network unit regressions       |             Pass | `go test ./internal/network`                                                                                                            |
| Full integration               |             Pass | `make test-integration`: 6186 tests, 3 skipped, 68.903s                                                                                 |
| Runtime E2E                    |             Pass | `make test-e2e-runtime`                                                                                                                 |
| Browser/Web E2E                |             Pass | `make test-e2e-web`: 15/15 specs, including Network route                                                                               |
| Final release gate             |             Pass | `make verify`: web 189 files / 1401 tests; Go race DONE 5707 tests in 33.904s; lint 0; boundary check OK                                |
| Real LLM smoke                 |             Pass | `codex exec` returned `AGH-LLM-SMOKE-OK`                                                                                                |
| Real AGH + Codex ACP + Network | Pass with caveat | Isolated daemon created Codex ACP sessions, normal prompt returned `AGH_REAL_NETWORK_OK`, network direct reached `messages_delivered=1` |

## Network-Specific Result

The network stack was validated at four levels:

1. Unit/regression: delivery queue, prompt rendering, retry scheduling, and retry cap.
2. Integration: router, lifecycle, audit/timeline, manager, daemon network collaboration, and network-origin task reentry.
3. Browser E2E: operator creates/inspects network channel, peers, timeline state, and reload continuity.
4. Real provider smoke: isolated AGH daemon with Codex ACP joined peers to `release-smoke`, sent `direct` envelopes, and recorded delivery.

## LLM Caveat

The real AGH network LLM smoke proved transport and delivery to a live Codex ACP agent. The exact network token-response assertion is intentionally marked caveated because the real Codex agent treated network content as untrusted and followed AGH/network safety guidance plus agentic behavior, rather than simply echoing the token over the network message. This is acceptable for release confidence on AGH transport/delivery; deterministic token behavior remains covered by normal AGH prompt smoke and mock ACP E2E.

## Residual Risk

- Daytona integration skips remain environmental: 3 skips require `DAYTONA_API_KEY`.
- Real provider behavior is non-deterministic by design. The release suite now separates deterministic AGH correctness from live LLM/provider smoke.

## Release Recommendation

Proceed with release candidate. All blocking local verification gates pass, network-specific coverage is green, and the only caveat is provider behavior outside AGH's deterministic control.
