# AGH First Release QA Plan - OpenClaw Comparison

## Executive Summary

This plan validates AGH release readiness after comparing the local AGH implementation against `.resources/openclaw` production-grade operational patterns. The highest-risk surface is AGH Network because it coordinates live agent sessions, persisted audit/timeline state, NATS transport, and CLI/API/Web user workflows.

Objectives:

- Verify the repository contract and release gates from the current workspace.
- Compare AGH Network and runtime behavior against OpenClaw patterns for delivery, recovery, live testing, boundary checks, and operational diagnostics.
- Add or run automated coverage for any critical production-readiness gaps discovered.
- Exercise the network feature through public interfaces, including real LLM smoke where credentials and local tools allow.

Key risks:

- Network messages can be accepted but not delivered, then become invisible to operators.
- Live LLM flows can differ from mock ACP flows because real agents may call tools, stream unexpectedly, or obey safety guidance.
- Browser UI and API surfaces can drift from backend contracts.
- Release gates can pass while credentialed or e2e lanes fail.

## Scope

In scope:

- Go backend verification through `make verify`, `make test-integration`, and `make test-e2e`.
- Network message delivery, backpressure, audit, status counters, and public CLI/API flows.
- Web UI smoke flows for the network surface if the dev/e2e server can be started.
- Real local LLM smoke using installed ACP-compatible tools when credentials exist.
- QA artifacts under `.codex/release-qa/qa/`.

Out of scope:

- Publishing a release artifact.
- Modifying `.resources/openclaw`.
- Credentialed third-party channels without local secrets.
- Legacy compatibility with old AGH state.

## Test Strategy

1. Discovery: read Makefile, CI, release workflow, docs, and network implementation.
2. Baseline: run the canonical verification gate and focused network tests before final claims.
3. Comparison-driven hardening: use OpenClaw evidence to find production-readiness gaps; add targeted regression tests before production fixes.
4. Public-surface validation: prefer CLI, HTTP/UDS, e2e harness, and browser flows over internal helpers.
5. Live integration: run real LLM smoke if local credentials/tools are available, and document blockers exactly.
6. Final gate: rerun full verification after the last code change.

## Environment Requirements

- macOS local workspace at `/Users/pedronauck/Dev/compozy/agh`.
- Go and Bun versions compatible with CI (`GO_VERSION=1.25.4`, `BUN_VERSION=1.3.4`).
- Local `codex`, `claude`, or another ACP-compatible agent for live LLM smoke.
- Optional browser validation through the Codex in-app browser or repo Playwright lane.
- Optional live provider credentials such as `OPENAI_API_KEY`.

## Entry Criteria

- Worktree state reviewed with `git status --short`.
- Root instructions, Makefile, CI, release workflow, and relevant network docs read.
- QA artifact directory exists.
- No destructive git commands are used.

## Exit Criteria

- All P0 test cases pass or have a documented blocking prerequisite.
- `make verify` passes after the final code change.
- Network-focused tests pass after the fix.
- Integration/e2e/live validations are run where locally possible and documented.
- Verification report exists at `.codex/release-qa/qa/verification-report.md`.

## Risk Assessment

| Risk                                           | Probability |   Impact | Mitigation                                                                                  |
| ---------------------------------------------- | ----------: | -------: | ------------------------------------------------------------------------------------------- |
| Silent network message loss under backpressure |      Medium | Critical | Add audit/status coverage and production hook for queue drops.                              |
| Real LLM behavior diverges from mocks          |      Medium |     High | Run a real LLM smoke and capture exact command/output summary.                              |
| E2E lane flakes due to browser/runtime timing  |      Medium |     High | Use existing harness lanes and retry only after root-cause inspection.                      |
| Credentialed live scenarios unavailable        |        High |   Medium | Validate local boundaries and report blocked credentialed cases explicitly.                 |
| Release workflow misses heavy lanes            |         Low |     High | Run `make test-integration` and `make test-e2e` in addition to `make verify` when feasible. |

## Timeline and Deliverables

- Test plan and cases: `.codex/release-qa/qa/test-plans/`, `.codex/release-qa/qa/test-cases/`.
- Bug reports if found: `.codex/release-qa/qa/issues/`.
- Screenshots/browser evidence: `.codex/release-qa/qa/screenshots/`.
- Final verification report: `.codex/release-qa/qa/verification-report.md`.
