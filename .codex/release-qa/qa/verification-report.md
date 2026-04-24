# AGH Release Verification Report - OpenClaw Comparison

Date: 2026-04-24

## Scope

Validate AGH for first-release readiness by comparing `.resources/openclaw` production-grade behavior against AGH, applying critical fixes, and running release-grade QA with special focus on the agent network.

## OpenClaw comparison result

OpenClaw treats delivery/backpressure as an auditable production concern: outbound delivery state is persisted, failures are classified, and recovery is observable. The critical AGH gap found during comparison was that inbound network queue overflow dropped the oldest envelope with log-only visibility. For an Agent OS, invisible message loss is a P0 operational risk because operators cannot distinguish "no work happened" from "work was dropped under load".

## Fixes applied

- Network delivery overflow is now surfaced through the manager audit path as a rejected delivery with reason `queue_overflow`.
- A regression test now proves overflowed inbound network envelopes create rejected audit records instead of disappearing silently.
- Daemon restart readiness handling now preserves replacement-process exit evidence when readiness timeout and process exit race at the boundary.
- Bridge route details now expose route session IDs in the UI so operators can trace bridge-created routes into sessions.
- Teams provider integration now waits for the specific managed instances to report `ready`, eliminating the repeated false failure caused by counting any two state records.

## Verification matrix

| Area                          | Command / evidence                                                                                                      | Result                                                                                    |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Network overflow regression   | `go test ./internal/network -run TestManagerAuditsBusyQueueOverflowAsRejected -count=1`                                 | Pass                                                                                      |
| Network package               | `go test ./internal/network -count=1`                                                                                   | Pass                                                                                      |
| Restart race regression       | `go test -race ./internal/daemon -run TestRunRelaunchHelperWrapperUsesDefaultLauncherAndPersistsFailure -count=10`      | Pass                                                                                      |
| Daemon package                | `go test -race ./internal/daemon -count=1`                                                                              | Pass                                                                                      |
| Teams integration flake       | `go test -race -tags integration ./internal/extension -run TestTeamsProviderLaunchNegotiatesBridgeRuntime -count=10 -v` | Pass                                                                                      |
| Extension integration package | `go test -race -tags integration ./internal/extension -count=1`                                                         | Pass                                                                                      |
| Web bridge route regression   | `bun run --cwd web test:raw src/systems/bridges/components/bridge-detail-panel.test.tsx`                                | Pass, 7 tests                                                                             |
| Web nightly route flow        | `bun run --cwd web test:e2e:nightly`                                                                                    | Pass, 1 spec                                                                              |
| Full integration              | `make test-integration`                                                                                                 | Pass, 6187 tests, 3 skipped                                                               |
| Blocking repo gate            | `make verify`                                                                                                           | Pass: web format/lint/typecheck/unit/build, Go lint/race tests/build, package boundaries  |
| Full e2e                      | `make test-e2e`                                                                                                         | Pass: daemon/API/testutil lanes and 15 daemon-served Playwright specs                     |
| Nightly e2e                   | `make test-e2e-nightly`                                                                                                 | Pass: runtime/nightly lanes, daemon-served Playwright 15 specs, nightly Playwright 1 spec |
| Patch hygiene                 | `git diff --check`                                                                                                      | Pass                                                                                      |

## Live LLM and network validation

- Direct Codex CLI LLM smoke passed: `codex exec --ephemeral --skip-git-repo-check --sandbox read-only -C /tmp --json "Reply with exactly AGH-OPENCLAW-LLM-SMOKE-OK and nothing else."` returned exactly `AGH-OPENCLAW-LLM-SMOKE-OK`.
- Live AGH/Codex ACP prompt smoke passed with a short deterministic token: `prompt_text=OK`.
- Live AGH network smoke passed with two Codex ACP sessions joined to the `release` channel:
  - Direct message from sender to receiver was audited as `sent` and `received`.
  - Receiver replied directly to sender.
  - Reply was audited as `sent` and `received`.
  - Final status had `messages_rejected=0` and direct-kind sent/received metrics for both directions.

## Skips and caveats

- Daytona credentialed integration/nightly tests were skipped by their own guard because `DAYTONA_API_KEY` is not present in the environment.
- This report therefore validates all available local, integration, e2e, browser, and live Codex/LLM lanes. Credentialed Daytona validation still requires providing `DAYTONA_API_KEY` and rerunning the Daytona lane.

## Release assessment

Release QA status: PASS for all available gates.

The OpenClaw comparison produced one critical network production-readiness fix and the verification cycle exposed and fixed two additional release blockers: a daemon restart race and a nondeterministic Teams integration wait. The agent network has unit, integration, e2e browser, and live LLM-backed validation evidence after the fixes.
