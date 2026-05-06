# Memory v2 Slice 1 QA Test Plan

**qa-output-path:** `.compozy/tasks/mem-v2`
**Artifact root:** `.compozy/tasks/mem-v2/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Executive Summary

Memory v2 Slice 1 is a cross-cutting runtime hard cut. Tasks 01-24 replaced path-keyed, two-scope memory behavior with stable `workspace_id` identity, per-workspace databases, controller-gated mutations, deterministic recall, live recall signals, provider registry, frozen prompt snapshots, extractor and dreaming runtimes, forensic session ledgers, CLI/HTTP/UDS/native-tool/extension manageability, web Knowledge/Settings/Session Inspector surfaces, and generated docs/reference co-ship.

This plan converts those completed slices into an execution-ready release QA dossier for task_26. It is behavior-first: smoke checks are only entry criteria, and release evidence must come from real runtime seams, isolated daemon state, CLI/API parity, browser-observable web surfaces, generated docs, and durable artifacts.

The highest-risk finding carried from shared memory is mandatory in this plan: controller-backed CLI/UDS writes were previously observed as not visible to search before explicit reindex in an e2e path. TC-SCEN-001 makes that a P0 scenario. If it reproduces during task_26, it is a fix-before-ship bug, not a waived expectation.

## Objectives

- Prove every curated memory mutation enters through the controller and persists decision WAL material before file/catalog mutation.
- Prove controller-backed CLI, HTTP, UDS, and native-tool writes are immediately discoverable through the supported read/search surfaces without requiring an undocumented manual reindex.
- Prove deterministic recall behavior: trivial-query skip, FTS/trigram retrieval, top-K caps, freshness banners, scope shadowing, live recall-signal writes, and system namespace exclusion.
- Prove stable workspace identity: `.agh/workspace.toml` survives workspace moves and keeps catalog/search rows attached to the same `workspace_id`.
- Prove frozen prompt snapshots: current sessions do not mutate mid-session, `agh memory reload` affects the next session boot only, and sub-agents inherit read-only parent snapshots.
- Prove extractor and dreaming flows: persisted-message hook, bounded queue/coalescing, `_inbox/`, `_system/extractor/failures`, signal gates, `_system/dreaming`, `_system/dream/failures`, and retry/idempotency behavior.
- Prove provider extensibility: bundled local provider is active, all MemoryProvider lifecycle hooks are reachable, extension host registry rejects collisions, and provider-first Host API recall falls back only for absent/not-implemented providers.
- Prove agent-manageability parity across CLI, HTTP, UDS, native tools, structured outputs, deterministic error envelopes, and docs.
- Prove UI truth: Knowledge, Memory Settings, and Session Inspector expose only implemented daemon behavior and keep error/loading/empty states actionable.
- Prove docs/reference truth: generated CLI/API docs and narrative docs match runtime routes, config keys, and hard-cut deleted surfaces.

## Scope

In scope:

- Backend runtime packages from tasks 01-19: `internal/memory`, `internal/memory/contract`, `internal/memory/controller`, `internal/memory/recall`, `internal/memory/extractor`, `internal/memory/provider/local`, `internal/sessions/ledger`, `internal/store/workspacedb`, `internal/workspace`, `internal/config`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli`, `internal/tools`, `internal/extension`, `internal/daemon`, `internal/sse`, and `internal/situation`.
- Public surfaces from tasks 14-18: OpenAPI/TypeScript contracts, CLI verbs, HTTP routes, UDS routes, native built-in tool IDs, extension Host API memory/provider surfaces, and deterministic error payloads.
- Web surfaces from tasks 20-22: `web/src/routes/_app/knowledge.tsx`, `web/src/routes/_app/settings/memory.tsx`, `web/src/routes/_app/agents.$name.sessions.$id.tsx`, `web/src/systems/knowledge/**`, `web/src/systems/settings/**`, and `web/src/systems/session/**`.
- Docs/reference surfaces from tasks 23-24: runtime memory/config/workspace/session/hooks/extensions docs, generated CLI reference, generated API reference, docs truth tests, and discoverability tests.
- QA execution preparation for task_26: isolated runtime prerequisites, scenario data, command matrix, evidence paths, and bug filing rules.

Out of scope for task_25:

- Executing live daemon/browser/provider scenarios. That belongs to task_26.
- Fixing defects found during execution. Task_26 owns reproduction, root-cause fixes, and reruns.
- Compatibility with removed Memory v1 surfaces. Slice 1 is a hard cut; old verbs/routes/tool IDs are negative assertions only.
- Slice 2+ features: compaction-flush extraction, vector recall, external providers, network federation, KG/bi-temporal memory.

## Behavioral Scenario Charter

- Startup situation: a fresh isolated QA lab with unique `AGH_HOME`, daemon ports, UDS socket, web proxy target, provider home, and a realistic workspace containing multiple agents and memory scopes.
- Operator intent: configure Memory v2, create and inspect memories through public surfaces, run provider-backed sessions, verify automatic extraction/dreaming/ledger behavior, and confirm web/docs truth.
- Agent roles: root agent with write permission, sub-agent with read-only inherited memory, dreaming-curator for synthesis, and a fixture extension provider for provider lifecycle/collision checks.
- Expected business outcome: an operator can trust Memory v2 as a durable, inspectable, agent-manageable memory subsystem without hidden write bypasses, search drift, prompt leaks, or UI/docs misrepresentation.
- Live provider/LLM expectation: use a real provider-backed session when credentials are available. If provider access is unavailable, task_26 must document the boundary and still execute the daemon-controlled surfaces with the project e2e harness.
- Expected artifacts: memory files, catalog rows, `memory_decisions`, `memory_events`, recall signals, `_inbox/` records, `_system/` DLQ/artifacts, session `ledger.jsonl`, CLI/API JSON, browser screenshots, docs build logs, and final `qa/verification-report.md`.

## Test Strategy

1. Run smoke readiness first: repo gates, isolated bootstrap, daemon/API/web readiness, generated docs/codegen truth, and public surface smoke commands.
2. Run P0 behavioral scenarios: controller-backed write/search visibility, transport/native-tool parity, session snapshot/ledger journey, redaction/system namespace exclusion, and config/settings lifecycle.
3. Run targeted runtime lanes: controller/WAL/replay, recall/signals/shadowing, provider/extension host, extractor/dreaming, workspace move, and transport parity.
4. Run targeted UI/docs lanes: Knowledge, Memory Settings, Session Inspector, generated CLI/API docs, site build, and web tests.
5. For every reproduced defect, file `qa/issues/BUG-NNN.md`, fix the root cause, rerun the failing case, rerun affected focused tests, and finish with `make verify`.
6. Treat smoke, CRUD-only, page-render-only, or mock-only evidence as insufficient for release. Every P0 case must cite at least one real persisted artifact or public cross-surface assertion.

## Environment Requirements

| Environment | Purpose | Required Evidence In Task 26 |
|---|---|---|
| Fresh isolated AGH lab | Prevent cross-run contamination and prove local-first behavior | `qa/bootstrap-manifest.json`, `qa/bootstrap.env`, daemon URL, UDS socket, runtime home |
| Unique ports and sockets | Avoid parallel worktree collisions | Manifest fields for HTTP base URL, UDS path, tmux-bridge socket, web proxy target |
| Isolated provider home | Preserve provider credential boundary | `PROVIDER_HOME` and `PROVIDER_CODEX_HOME` from manifest or documented native-cli exception |
| Scenario workspace | Realistic workspace and agent definitions | Workspace path, `.agh/workspace.toml`, agent files, sample memory input files |
| SQLite inspection tools | Verify durable state | Saved SQL query outputs for `memory_decisions`, `memory_events`, `memory_recall_signals`, `memory_consolidations` |
| Browser desktop and mobile | Verify web Knowledge/Settings/Session Inspector | Screenshots or DOM snapshots under `qa/screenshots/<TC-ID>/` |
| Site/build toolchain | Verify docs/reference truth | Logs for `make cli-docs`, `make codegen-check`, site tests/build |
| Full repo gate | Final correctness guard | `make verify` log after any task_26 fix set |

## Entry Criteria

- `state.yaml` marks tasks 01-24 completed.
- `detect-phase.py mem-v2` points at task_25 before planning and task_26 after planning.
- No unresolved critical/high blocker exists in `memory/MEMORY.md` except the search-visibility risk now mapped to TC-SCEN-001.
- Generated API and CLI references are current after task_24.
- Dependencies are installed for Go, Bun workspaces, web, and site.
- Task_26 must create a fresh QA lab by default unless it is explicitly resuming the same manifest from the same active QA pass.

## Exit Criteria

- All P0 test cases in this dossier are executed and pass, or each failure has a `BUG-NNN.md` with a root-cause fix and rerun evidence.
- At least 90 percent of P1 cases pass. No P1 exception may hide data loss, redaction failure, cross-surface drift, or undocumented reindex requirement.
- `make verify` passes after the final task_26 change.
- `make test-e2e-runtime` and `make test-e2e-web` pass for Memory v2 lanes or any unavailable provider/browser boundary is documented with concrete reason and substitute evidence.
- `make codegen-check`, `make cli-docs`, focused site tests, and site build pass after final fixes.
- `qa/verification-report.md` lists commands, evidence paths, bugs, reruns, residual risks, and final verdict.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---:|---:|---|
| Controller-backed CLI/UDS writes are not visible to search before explicit reindex | Medium | Critical | TC-SCEN-001 requires immediate cross-surface search visibility and marks reproduction as fix-before-ship. |
| A mutation bypasses `Controller.Decide` or lacks WAL replay material | Medium | Critical | TC-INT-001 checks `memory_decisions`, replay, revert, events, and direct-write negative paths. |
| `_system/`, prompt-only memory, raw LLM traces, or secrets leak into recall/SSE/UI/docs | Medium | Critical | TC-SEC-001 combines recall, SSE/log grep, public payload assertions, and docs/web fixture sweeps. |
| Scope/agent-tier shadowing leaks private workspace memory across agents or workspaces | Medium | Critical | TC-INT-002 and TC-SCEN-002 validate agent-workspace, agent-global, workspace, and global precedence. |
| Frozen snapshot semantics regress to current-session mutation | Low | High | TC-SCEN-002 verifies next-session-only reload and sub-agent read-only inheritance. |
| Provider lifecycle is declared but not consumed by daemon/runtime | Medium | High | TC-INT-004 requires active local provider, lifecycle hook evidence, Host API recall, and collision rejection. |
| Extractor/dreaming background workers race shutdown or lose DLQ payloads | Medium | High | TC-INT-005 runs queue, coalescing, drain, DLQ, retry, and idempotency checks. |
| Web UI exposes speculative controls or stale Memory v1 names | Medium | High | TC-UI-001..003 check truthful controls, named route exports, generated contract types, and negative copy. |
| Docs/reference drift after generated CLI/API refresh | Medium | High | TC-REG-001 and automated artifact test guard generated references and narrative docs. |
| Config lifecycle mismatches backend validation or settings UI mutability | Medium | High | TC-UI-002 and TC-REG-001 validate backend defaults, settings payloads, readonly daemon paths, and docs keys. |

## Timeline And Deliverables

| Stage | Owner | Deliverables |
|---|---|---|
| Planning (task_25) | Codex loop | This plan, regression suite, traceability matrix, test cases, QA artifact guard test |
| Bootstrap (task_26) | QA execution | Fresh lab manifest, bootstrap env, scenario workspace, provider home |
| Execution (task_26) | QA execution | Per-case logs, screenshots, SQL evidence, bug reports |
| Fix/rerun (task_26) | QA execution + implementation | Root-cause fixes, focused rerun evidence, final `make verify` |
| Closeout (task_26) | QA execution | `qa/verification-report.md`, residual-risk statement, next phase detector |

## Artifact Layout

| Path | Purpose |
|---|---|
| `.compozy/tasks/mem-v2/qa/test-plans/memory-v2-test-plan.md` | Primary QA plan |
| `.compozy/tasks/mem-v2/qa/test-plans/memory-v2-regression.md` | Smoke, targeted, and full regression lanes |
| `.compozy/tasks/mem-v2/qa/test-plans/memory-v2-traceability.md` | Task/ADR/surface to TC mapping and coverage audit |
| `.compozy/tasks/mem-v2/qa/test-cases/TC-*.md` | Manual/executable scenario definitions for task_26 |
| `.compozy/tasks/mem-v2/qa/issues/BUG-*.md` | Bug reports generated during execution |
| `.compozy/tasks/mem-v2/qa/screenshots/<TC-ID>/` | Browser/visual evidence |
| `.compozy/tasks/mem-v2/qa/logs/<TC-ID>/` | Command, daemon, API, SQL, and build logs |
| `.compozy/tasks/mem-v2/qa/verification-report.md` | Final execution report produced by task_26 |

