# Network Threads QA Test Plan

## Executive Summary

This plan validates the AGH Network hard cut from flat channel / peer timelines to explicit conversation containers:

- `surface:"thread"` with `thread_id` for public N-to-N conversations.
- `surface:"direct"` with deterministic `direct_id` for restricted two-party rooms.
- `work_id` for lifecycle-bearing work bound to one conversation container.

The QA objective is behavior-first evidence that an operator and agents can coordinate real work through public threads, direct rooms, summarize-back flows, and cross-surface inspection without any active `interaction_id`, `kind:"direct"`, or peer-room timeline behavior leaking back into current surfaces.

Key risks:

| Risk | Why it matters | Primary coverage |
| --- | --- | --- |
| Conversation isolation breaks | Direct room messages leaking into public threads would violate the product model. | TC-SCEN-002, TC-UI-001, TC-REG-001 |
| `work_id` becomes a queue or conversation alias | This would reintroduce the ambiguity the hard cut removed. | TC-SCEN-001, TC-SCEN-003, TC-REG-001 |
| CLI/API/Web disagree | Agents rely on structured control-plane truth, not only web UI state. | TC-SCEN-001, TC-SCEN-002, TC-UI-001 |
| Direct-room resolution races fragment history | Two peers opening the same direct room must converge on one `direct_id`. | TC-INT-001 |
| Generated docs/contracts drift | Operators and agents would learn old flags or old protocol fields. | SMOKE-001, TC-REG-001 |
| Provider-backed agent behavior is unavailable | Local checks alone do not prove live agent cooperation. | All P0 scenario cases record live-provider evidence or exact blocked boundary. |

## Scope Definition

In scope:

- Runtime thread/direct/work persistence and query surfaces.
- CLI thread/direct/work/send commands and structured output.
- HTTP and UDS parity for thread/direct/work endpoints.
- Native tools and hosted/MCP tool schema behavior used by agents.
- Extension Host API read/write capability behavior at the scenario level.
- Web `/network` route tree for channel threads, thread detail, directs, direct detail, and activity.
- Browser artifact fields `network_selected_thread` and `network_selected_direct`.
- Site/runtime docs and CLI reference guardrails for current protocol vocabulary.
- E2E harness commands: `make test-e2e-runtime`, `make test-e2e-web`, and `make verify`.

Out of scope:

- Private group threads or direct rooms with more than two peers.
- Cryptographic privacy claims for direct rooms.
- Unread synchronization, notification preferences, retention controls, transcript export, and analytics dashboards.
- Compatibility aliases for `interaction_id`, `kind:"direct"`, `--interaction-id`, `--thread-id`, `--direct-id`, or `--work-id`.
- Archived `.compozy/tasks/_archived/*` historical terminology.

## Behavioral Scenario Charter

Startup situation:

- A fresh QA lab starts AGH with an isolated `AGH_HOME`, isolated provider homes where provider policy requires it, unique daemon ports, and a scenario workspace.
- Network fixtures include a `builders` channel, at least two agent peers, and seeded public thread / direct room examples.
- The web dev server is launched with `AGH_WEB_API_PROXY_TARGET` from the QA bootstrap manifest when the daemon is not on the default port.

Operator intent:

- Coordinate a launch/review conversation publicly.
- Move sensitive or detailed review work into a restricted direct room.
- Bring a concise result back to the public thread.
- Inspect the same state through CLI, API, Web, native tools, and persisted runtime evidence.

Expected business outcome:

- The operator can understand which conversation is public, which room is restricted, which work item is active or terminal, and which agent produced the result.
- Agents keep lifecycle work scoped to one conversation container and use `reply_to`, `trace_id`, and `causation_id` to connect handoffs and summaries.

Agent roles:

| Actor/Agent | Role | Expected behavior | Evidence source |
| --- | --- | --- | --- |
| Operator | Scenario driver | Starts daemon/lab, opens thread, requests review, inspects state. | CLI transcript, API response, browser screenshots, verification report. |
| Founder / requester agent | Public work initiator | Opens or continues the public thread and asks for review. | Thread messages, prompt wrapper metadata, persisted audit rows. |
| Reviewer agent | Restricted work owner | Resolves direct room, advances `work_id`, produces a useful review artifact or summary. | Direct-room messages, work lookup, session transcript or blocked provider boundary. |
| Observer / QA agent | Cross-surface verifier | Compares CLI/API/Web/runtime views and records mismatches as bugs. | QA report, bug files, screenshots. |

Live provider / LLM expectations:

- Release-grade execution should use a provider-backed AGH session when credentials and local prerequisites are reachable.
- If live provider execution is blocked, QA execution must name the exact provider, credential, binary, or account boundary and still validate all local runtime, CLI, API, UDS, Web, and E2E harness surfaces.
- Mock/acpmock evidence remains readiness or regression evidence only; it is not counted as live provider proof.

Expected artifacts:

- `.compozy/tasks/network-threads/qa/verification-report.md`
- CLI/API/Web transcripts and screenshots under `.compozy/tasks/network-threads/qa/`
- Any bug reports under `.compozy/tasks/network-threads/qa/issues/BUG-*.md`
- Browser screenshots proving thread/direct route state under `.compozy/tasks/network-threads/qa/screenshots/`
- QA bootstrap block with manifest path, lab root, runtime home, base URL, and verification evidence if a reusable lab remains healthy.

Disruption probes:

- Stop/restart daemon or rerun runtime harness and confirm persisted thread/direct/work state remains coherent.
- Resolve the same direct room concurrently from both peers and confirm one deterministic `direct_id`.
- Submit a legacy field or flag and confirm deterministic rejection rather than fallback.
- Navigate directly to a missing thread/direct route and confirm operator-readable error state.

## Test Strategy and Approach

Smoke readiness checks:

- Confirm required commands exist and broad gates can run.
- Confirm generated docs/contracts do not contain active legacy vocabulary.
- Confirm web and runtime E2E harnesses are runnable.
- Smoke checks are entry criteria only and must not be reported as release-grade proof.

Release-grade behavioral evidence:

- Execute P0 real-scenario journeys first: public thread coordination, restricted direct handoff, summarize-back, and direct resolve race.
- For each P0 case, collect CLI command/output, API response, persisted runtime state, browser screenshot when the Web surface is involved, and live provider evidence or an explicit blocked-provider boundary.
- Compare at least one persisted object across CLI/API/Web/runtime state in every P0 scenario.
- Run a disruption probe in every P0 journey and record operator-visible behavior.

Regression evidence:

- Run `make verify` after the last code or fixture change.
- Run `make test-e2e-runtime` and `make test-e2e-web` as targeted behavior harnesses.
- Re-run the highest-risk scenario after the full gate passes.
- Keep failing scenarios as bug reports, not silent notes.

## Environment Requirements

| Requirement | Expected value |
| --- | --- |
| OS | macOS developer workstation or CI-equivalent Linux runner with AGH prerequisites. |
| Runtime | Go toolchain, Bun workspace dependencies, SQLite with race-test support. |
| Browser | Browser plugin / browser-use for local web validation; approved fallback is `agent-browser`. |
| Daemon isolation | Fresh QA lab by default; unique `AGH_HOME`, daemon ports, and tmux bridge socket paths. |
| Provider homes | Use `PROVIDER_HOME` / `PROVIDER_CODEX_HOME` for bound-secret or brokered providers; preserve operator home for `native_cli` provider policy. |
| Web proxy | Export `AGH_WEB_API_PROXY_TARGET` from bootstrap manifest before `make web-dev` when daemon port is isolated. |
| Output root | `.compozy/tasks/network-threads/qa/` |

## Entry Criteria

- `task_01` through `task_17` are completed in task frontmatter and mirrored in `state.yaml`.
- `task_18` QA planning artifacts exist and pass structural checks.
- A fresh or explicitly reusable QA bootstrap manifest is available before live QA execution, or `qa-execution` records why bootstrap could not be created.
- No known unrelated dirty worktree changes are modified or reverted by QA.
- The QA execution task has access to this plan, the test cases, and the regression suite.

## Exit Criteria

- All P0 real-scenario journeys either pass or produce bug reports with exact reproduction and evidence.
- 90% or more P1 test cases pass, with no critical or high bug left unresolved.
- CLI, API/UDS, runtime store, and Web UI agree for the same thread/direct/work objects.
- Live provider-backed behavior is exercised, or the exact blocked provider/tool/credential boundary is documented.
- `make verify` passes after the last fix or artifact change.
- `make test-e2e-runtime` and `make test-e2e-web` pass or any blocker is documented with the exact failing command and issue file.
- `.compozy/tasks/network-threads/qa/verification-report.md` includes the required QA bootstrap block when a healthy reusable lab remains.

## Execution Matrix

| ID | Priority | Class | Primary surfaces | Must run before |
| --- | --- | --- | --- | --- |
| SMOKE-001 | P0 | Smoke readiness | Make targets, docs scan, generated artifacts | Any P0 journey |
| TC-SCEN-001 | P0 | E2E / Real Scenario | CLI, API, Web, runtime store, provider-backed session | TC-SCEN-002 |
| TC-SCEN-002 | P0 | E2E / Real Scenario | CLI, API, native tools, runtime store, provider-backed session | TC-SCEN-003 |
| TC-SCEN-003 | P0 | E2E / Real Scenario | CLI, API, Web, runtime store | Final verification |
| TC-INT-001 | P0 | Integration / E2E | Runtime harness, API, store | Final verification |
| TC-UI-001 | P1 | Browser E2E | Web, API proxy, browser artifacts | Final verification |
| TC-REG-001 | P1 | Regression | CLI, API, docs, generated contracts, native tools | Final verification |

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Live provider credentials unavailable | Medium | High | Record exact boundary, continue all local surfaces, do not claim live provider proof. |
| Browser dev server points at default daemon | Medium | High | Derive `AGH_WEB_API_PROXY_TARGET` from bootstrap manifest. |
| QA lab reuses stale state | Medium | Medium | Fresh lab by default; reuse only same-session healthy manifest. |
| E2E runtime harness fails from unrelated race | Medium | High | Reproduce narrowly, file BUG, fix root cause only if it is in scope and confirmed. |
| Legacy vocabulary appears in archived docs | High | Low | Limit failure to active docs/contracts; archived artifacts are provenance. |
| Direct-room privacy overclaim in copy | Low | High | Assert "restricted, not encrypted" in docs/tool/UI test coverage. |

## Timeline and Deliverables

| Phase | Deliverable | Output |
| --- | --- | --- |
| Planning | Feature test plan | `qa/test-plans/network-threads-test-plan.md` |
| Planning | Regression suite | `qa/test-plans/network-threads-regression.md` |
| Planning | Execution-ready cases | `qa/test-cases/SMOKE-001.md`, `TC-*.md` |
| Execution | Baseline verification and behavioral evidence | `qa/verification-report.md` |
| Execution | Bug reports, screenshots, transcripts | `qa/issues/`, `qa/screenshots/` |

