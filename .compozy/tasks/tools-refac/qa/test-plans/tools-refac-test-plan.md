# Tool Registry Canonical Surface QA Test Plan

**qa-output-path:** `.compozy/tasks/tools-refac`
**Artifact root:** `.compozy/tasks/tools-refac/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Executive Summary

`tools-refac` finalizes the canonical AGH tool surface on top of the `tools-registry` foundation already shipped on this branch. Tasks 01-11 extend the registry with dynamic policy-input resolution and default discovery overlays (task_01), tools guidance assets and a startup prompt section (task_02), read-only built-ins for coordination/session/workspace (task_03) and memory/observe/bridge (task_04), mutable built-in families for config (task_05), hooks (task_06), automation (task_07), and extensions (task_08), the session-bound autonomy hard cut (task_09), MCP auth status plus hosted MCP projection parity (task_10), and the docs/codegen/example alignment pass (task_11).

This plan converts that work into an execution-ready QA dossier for task_13. It pins the invariants that must be proven from real seams — daemon, SQLite, CLI human/JSON output, HTTP/UDS payloads, hosted MCP projection, generated OpenAPI/TypeScript contracts, web fixtures and tests, and `packages/site` build output. It also pins the negative, concurrency, and redaction matrix that the autonomy hard cut and broadened mutable surface make load-bearing.

Key risks the plan must contain:

- Raw `claim_token` slipping back into any AGH-owned surface (CLI flag, HTTP/UDS DTO, OpenAPI schema, hosted MCP payload, log line, SSE event, observe/memory/web fixture, docs example).
- Default discovery overlay drifting between projection, dispatch, and hosted MCP — for example `agh__bootstrap` showing up in `tools/list` without being callable, or vice versa.
- Mutable tool families bypassing the existing CLI/HTTP writers, validators, or approval gates and creating a parallel mutation lane.
- Cache-as-authority regressions where projection caches return stale callable/denial decisions after agent reload, lineage change, source-health change, hook reload, MCP auth health change, or config overlay change.
- MCP auth status leaking token/PKCE/code material, or hosted MCP approval bridge satisfying client-supplied approval credentials instead of going through ACP.
- Generated OpenAPI/TypeScript drift between `internal/api/spec`, `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, and `web/src/systems/tasks/mocks/fixtures.ts`.
- Site docs and CLI references retaining stale `--claim-token` examples, opt-in-discovery prose, or CLI-first guidance for AGH internals where a tool exists.

## Objectives

- Prove default discovery toolsets (`agh__bootstrap`, `agh__catalog`) reach every agent unless effective policy denies them, across `list`, `search`, `get`, `call`, hosted MCP `tools/list`, and `GET /api/sessions/{id}/tools`.
- Prove the dynamic policy resolver consumes current agent definition, session lineage, source policy, availability, and hook outputs at call time and that operator/session projections diverge correctly for unavailable, denied, or hook-blocked tools.
- Prove the new `agh-tools-guide` plus `HarnessPromptSectionTools` startup section render, that catalog text references `agh__skill_view` first, and that the `agh-agent-setup` examples no longer treat `agh__catalog` as opt-in.
- Prove the expanded built-in surface (`agh__memory`, `agh__sessions`, `agh__workspace`, `agh__config`, `agh__autonomy`, `agh__coordination` extension, `agh__hooks`, `agh__automation`, `agh__extensions`, `agh__mcp_auth`, `agh__observe`, `agh__bridges`) reuses existing domain writers/validators and respects the same approval, source-trust, and redaction rules as their CLI/HTTP/UDS counterparts.
- Prove the autonomy hard cut: every AGH-owned tool/CLI/HTTP/UDS/MCP/contract/log/SSE/observe/memory/web/docs surface no longer accepts or emits raw `claim_token`. Public callers identify leases by `run_id` and the daemon resolves the active lease server-side. `claim_token_hash` survives only as observability metadata.
- Prove the `agh__autonomy` invariants: `AUTONOMY_SESSION_REQUIRED`, `AUTONOMY_NO_ACTIVE_LEASE`, `AUTONOMY_FOREIGN_RUN`, `AUTONOMY_LEASE_EXPIRED`, `AUTONOMY_LEASE_ALREADY_HELD`, single-success heartbeat under cross-session contention, and convergence of tool/CLI/HTTP/UDS on the same `task.Service` lease writers.
- Prove `agh__mcp_auth_status` returns redacted status only — no tokens, codes, PKCE verifiers, callback secrets — and that `agh mcp auth login`/`logout` remain on the management surface, never tool-callable.
- Prove hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools` exactly, that the approval bridge enforces ACP `session/request_permission` (not client-supplied `approval_token`), and that bind-nonce + UDS peer-cred + AGH-binary validation fail closed for foreign processes.
- Prove `agh__network_send` rejects raw token payloads/metadata with `network_raw_token_rejected` and that no AGH-owned message surface reintroduces a token-bearing field.
- Prove generated artifacts (`make codegen`, `make codegen-check`, `make cli-docs`, `packages/site` build, web `bun-typecheck`/`bun-test`) all stay aligned with the runtime contract after the cut.
- Prove the deterministic denial taxonomy is consistent across surfaces for config, hooks, automation, extensions, autonomy, and MCP auth (`CONFIG_PATH_FORBIDDEN`, `CONFIG_SECRET_PATH_FORBIDDEN`, `CONFIG_TRUST_ROOT_FORBIDDEN`, `CONFIG_SCOPE_NOT_ALLOWED`, `CONFIG_VALIDATION_FAILED`, `HOOK_SOURCE_IMMUTABLE`, `HOOK_SECRET_INPUT_FORBIDDEN`, `HOOK_VALIDATION_FAILED`, `HOOK_APPROVAL_REQUIRED`, `AUTOMATION_SCOPE_FORBIDDEN`, `AUTOMATION_SECRET_INPUT_FORBIDDEN`, `AUTOMATION_VALIDATION_FAILED`, `AUTOMATION_APPROVAL_REQUIRED`, `EXTENSION_SOURCE_FORBIDDEN`, `EXTENSION_APPROVAL_REQUIRED`, `EXTENSION_NOT_INSTALLED`, `EXTENSION_VALIDATION_FAILED`, and the canonical tool reason `network_raw_token_rejected`).

## Scope

In scope:

- Backend Go packages changed by tasks 01-11: `internal/tools`, `internal/tools/builtin`, `internal/daemon`, `internal/skills`, `internal/skills/bundled`, `internal/task`, `internal/api/contract`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/api/spec`, `internal/cli`, `internal/automation`, `internal/extension`, `internal/config`, `internal/hooks`, `internal/mcp/auth`, `internal/mcp` (hosted), `internal/observe`, `internal/memory`, `internal/network`, `internal/session`.
- CLI commands touched or expected to converge on the canonical surface: `agh tool list/search/info/invoke`, `agh toolsets list/info`, `agh task list/read/create/cancel/run/run-list`, `agh task next|heartbeat|complete|fail|release` (autonomy hard cut), `agh config show/list/get/set/unset/diff/path`, `agh hooks *`, `agh automation jobs|triggers|runs *`, `agh extension search/list/info/install/update/remove/enable/disable`, `agh mcp auth status` (and management `login/logout` left intact), `agh network status/channels/inbox/peers/send`, `agh memory *`, `agh observe events/metrics/search`, `agh bridge list/status`, `agh session list/status/history/events/describe`, `agh workspace list/info/describe`.
- HTTP/UDS routes: `GET /api/tools`, `POST /api/tools/search`, `GET /api/tools/{id}`, `POST /api/tools/{id}/invoke`, `GET /api/sessions/{id}/tools`, `POST /api/sessions/{id}/tools/search`, `GET /api/toolsets`, `GET /api/toolsets/{id}`, `POST /api/agents/tasks/runs/next`, `POST /api/agents/tasks/runs/{run_id}/heartbeat|complete|fail|release` (post hard cut, `run_id`-keyed), and the hosted MCP exposure under `agh tool mcp` / daemon UDS bind.
- Generated artifacts: `internal/api/spec` OpenAPI output, `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, `web/src/systems/tasks/mocks/fixtures.ts`, and the `packages/site/content/runtime/cli-reference/` tree.
- Web surfaces consuming the autonomy contract or tool DTOs: `web/src/systems/tasks/*`, `web/src/systems/automation/*` (DTO co-ship from task_07), `web/src/systems/settings/*` (MCP auth status from task_10), `web/src/systems/network|session|workspace|bridges/*` for shared DTO drift.
- Site docs: `packages/site/content/runtime/core/{configuration,agents,autonomy,hooks,automation,extensions,memory,network,workspaces,sessions,bridges}/*.mdx`, plus generated CLI references for every command above. The hand-authored `packages/site/content/runtime/cli-reference/index.mdx` and `meta.json` (which survive `make cli-docs` regeneration) are explicitly in scope to confirm new top-level CLI groups stay listed.

Out of scope for task_12 (planning only):

- Executing live runtime, CLI, API, hosted MCP, web, or site flows. That belongs to task_13.
- Fixing production defects discovered during execution. Those belong to task_13 with `systematic-debugging` and `no-workarounds`.
- Driver-specific shell-blocking, ACP-driver Bash changes, remote peer tool execution, lease-per-session relaxation, or self-healing OAuth login/logout — all explicitly deferred by the TechSpec.
- Any compatibility shim. The redesign is a hard cut; the plan does not validate "old + new in parallel" behavior.

## Environment Matrix

| Environment | Purpose | Required Evidence In Task 13 |
|-------------|---------|------------------------------|
| Isolated `AGH_HOME` per worktree (unique daemon ports, unique tmux-bridge socket) | Primary CLI/HTTP/UDS/hosted-MCP/tool runtime verification | `agh-qa-bootstrap` manifest under `qa/logs/<TC-ID>/bootstrap-manifest.json`; isolated coordinates recorded in `qa/verification-report.md` |
| `PROVIDER_HOME` / `PROVIDER_CODEX_HOME` from bootstrap manifest | Provider-backed QA without contaminating `~/.codex` | Provider home path captured in bootstrap manifest and referenced by every TC that touches a real provider |
| Two isolated sessions in the same workspace | Cross-session autonomy and MCP-auth contention | Session IDs recorded in `qa/logs/<TC-ID>/sessions.txt` and used by autonomy negative cases |
| Mock OAuth/MCP server | MCP auth status and hosted-MCP exposure without real provider secrets | Mock server logs under `qa/logs/<TC-ID>/mcp-server.log` |
| SQLite global + session DBs from the isolated `AGH_HOME` | `task_runs` lease state, automation cursor, hook ownership, extension registry, MCP token store | `sqlite3` query output recorded under `qa/logs/<TC-ID>/db-*.txt` |
| Browser desktop 1280px (settings, automation panels) | Optional spot-checks for UI fallout from autonomy/MCP-auth/automation DTO changes | Screenshots under `qa/screenshots/<TC-ID>/` only when a TC explicitly requires UI evidence |
| `packages/site` build (Bun) | Site docs and CLI reference build coverage | `qa/logs/<TC-ID>/site-build.log` |
| Repo gate | `make verify` after every code-bearing change | `qa/logs/<TC-ID>/make-verify.log` |

## Artifact Layout

| Path | Owner | Purpose |
|------|-------|---------|
| `.compozy/tasks/tools-refac/qa/test-plans/tools-refac-test-plan.md` | task_12 | This feature QA plan |
| `.compozy/tasks/tools-refac/qa/test-plans/tools-refac-regression.md` | task_12 | Smoke / targeted / full regression lanes and exit criteria |
| `.compozy/tasks/tools-refac/qa/test-plans/tools-refac-traceability.md` | task_12 | Task → TC mapping + regression hot spots |
| `.compozy/tasks/tools-refac/qa/test-plans/tools-refac-codegen-and-docs.md` | task_12 | Codegen + docs + downstream web verification dossier |
| `.compozy/tasks/tools-refac/qa/test-plans/tools-refac-redaction-suite.md` | task_12 | Redaction sweep procedure for raw `claim_token`, MCP secrets, and PII boundaries |
| `.compozy/tasks/tools-refac/qa/test-cases/TC-*.md` | task_12 | Manual execution cases |
| `.compozy/tasks/tools-refac/qa/issues/BUG-*.md` | task_13 if needed | Structured bug reports tied to a TC ID |
| `.compozy/tasks/tools-refac/qa/screenshots/<TC-ID>/...` | task_13 | Browser/docs screenshots for UI evidence |
| `.compozy/tasks/tools-refac/qa/logs/<TC-ID>/...` | task_13 | Command, daemon, mock, test, build logs |
| `.compozy/tasks/tools-refac/qa/verification-report.md` | task_13 | Final execution evidence |

## Test Strategy

1. **Smoke first.** Run the P0 cases that establish the autonomy hard cut, default discovery overlay, MCP auth status redaction, hosted MCP projection parity, and the codegen/docs alignment gate. Any P0 failure blocks the deeper run.
2. **Targeted lanes next.** Execute domain lanes (discovery/policy, read surfaces, mutable surfaces, autonomy, MCP/hosted MCP, web/docs) using the recommended commands listed in `tools-refac-regression.md`.
3. **Full regression last.** Run `make verify`, the codegen/docs gates, and the entire TC matrix after the final fix set; record the verdict in `qa/verification-report.md`.
4. **Real seams over parser-only checks.** Parser/config tests are acceptable only when paired with surfaced CLI/HTTP/UDS/hosted-MCP/web/docs evidence for the same invariant.
5. **Redaction is non-optional.** Any TC that touches autonomy, MCP auth, hooks, automation, extensions, or config must include the redaction grep step from `tools-refac-redaction-suite.md`.
6. **Concurrency is mandatory for autonomy.** TC-AUT-005 and TC-AUT-006 must run two isolated sessions in parallel and produce one success path plus a deterministic mismatch error.
7. **Provider-home isolation is mandatory.** Never run provider-backed cases against `~/.codex`; always use `PROVIDER_HOME` / `PROVIDER_CODEX_HOME` from the bootstrap manifest.
8. **Worktree isolation when running in parallel.** Concurrent runs use unique `AGH_HOME`, unique daemon ports, unique tmux-bridge socket paths.
9. **Sequential config writes.** `agh config set` and equivalent mutations against the same isolated home must run sequentially per `CLAUDE.md` rule.

## Entry Criteria

- Tasks 01-11 are completed in `_tasks.md` and their implementation commits are present in the local history (`a4601294`, `6640f66a`, `d5316f5b`, `eb2a9253`, `0b879ef1`, `b81143e7`, `06880bab`, `5735b42c`, `1119d6e4`, `5fa9f805`, `72dc927c`).
- The working tree is clean before task_13 execution; any uncommitted edits are deliberate and documented.
- `qa-output-path=.compozy/tasks/tools-refac` is passed unchanged to `qa-execution`.
- Test fixtures use isolated `AGH_HOME` paths and never rely on private local credentials.
- Web and site dependencies are installed (`bun install` at repo root).
- Any task_13 fix starts from a failing reproduction and adds durable regression coverage before the final repository gate.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass; any P1 exception has a `BUG-*.md` issue with severity, impact, workaround, and fix owner.
- No critical defect remains: no raw `claim_token` leak, no hosted MCP projection drift, no missing default discovery, no codegen/docs mismatch.
- `make verify` passes after the last task_13 change.
- `make codegen-check` passes; `make cli-docs` produces no diff against the committed tree (post-`bun run format`).
- `packages/site` build succeeds with the rewritten runtime docs.
- Web `bun-typecheck` and `bun-test` pass against the regenerated `agh-openapi.d.ts` / `web/src/systems/tasks/types.ts` / `web/src/systems/tasks/mocks/fixtures.ts`.
- `qa/verification-report.md` cites the executed commands, evidence paths, pass/fail status, open bugs, and final verdict.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Raw `claim_token` reappears in CLI flag, HTTP/UDS DTO, OpenAPI schema, hosted MCP payload, log, SSE, observe, memory, web fixture, or docs example | Medium | Critical | TC-SEC-001 redaction sweep across every channel; TC-AUT-001 parity test; codegen/docs grep in TC-REG-001 / TC-REG-002 |
| Default discovery overlay drifts between projection, dispatch, and hosted MCP `tools/list` | Medium | Critical | TC-FUNC-001 + TC-INT-003 verify equality of session projection ↔ hosted MCP `tools/list` and that overlay survives empty agent declarations |
| Mutable tool family bypasses CLI/HTTP writer or validator | Medium | Critical | TC-FUNC-004..007 each compare tool, CLI, HTTP/UDS for the same caller scope and assert the same error/decision; reuses existing validators evidenced by source pointer |
| Approval bridge accepts client-supplied approval credentials over hosted MCP | Low | Critical | TC-INT-004 disconnects/cancels/timeouts the ACP approval channel and asserts `approval_canceled`/`approval_timed_out`/`approval_unreachable` |
| Hosted MCP bind succeeds for foreign local process or stale nonce | Medium | Critical | TC-SEC-006 forces foreign UDS peer-credentials and stale/expired nonce paths and expects fail-closed behavior |
| Cache-as-authority regression after agent reload, lineage change, hook reload, MCP health change, or config overlay change | Medium | High | TC-INT-006 mutates each invalidation key and asserts subsequent projection/dispatch reflects new state |
| Autonomy lease bridge silently succeeds for foreign session/run | Low | Critical | TC-AUT-002 / TC-AUT-005 assert `AUTONOMY_FOREIGN_RUN` and single-success-only semantics under contention |
| MCP auth status leaks token / PKCE / code / callback material | Low | Critical | TC-SEC-002 grep checks across CLI, HTTP/UDS, settings JSON, observe payloads, logs |
| `make codegen` / `make cli-docs` drift between runtime and committed artifacts | Medium | High | TC-REG-001 + TC-REG-002 require diff = 0 against tree after regeneration |
| `packages/site` build breaks on rewritten runtime docs | Medium | Medium | TC-REG-003 runs `cd packages/site && bun run build` and the source-test suite |
| Web `tasks` system fixtures/types drift from autonomy hard cut | Medium | High | TC-REG-004 runs Vitest lanes for `web/src/systems/tasks/*` and confirms fixtures contain no `claim_token` |
| Network send accepts raw token payload/metadata | Low | Critical | TC-SEC-003 sends well-formed and `claim_token`-bearing bodies and asserts `network_raw_token_rejected` |
| Catalog text or `agh-agent-setup` regresses to CLI-first | Low | Medium | TC-FUNC-002 compares prompt assembly output and bundled-skill content against expected fixtures |
| Two-touch rule violation in re-fix loop | Medium | Medium | Regression script recommends a redesign TechSpec for any third change to the same package within this workstream |

## Web And Site Verification Requirements

| Task | Required web verification | Required site verification |
|------|---------------------------|----------------------------|
| 01 | Confirm tool projection payload changes do not break `web/src/generated/agh-openapi.d.ts` consumers; no new web UI is required. | Confirm `agent-md.mdx` / `definitions.mdx` describe default discovery semantics and the `tools` / `toolsets` / `deny_tools` grammar without "opt-in" prose. |
| 02 | None — guidance change only. | Confirm `agent-md.mdx` and `definitions.mdx` reflect the canonical tool surface and bundled `agh-tools-guide`; CLI references for `agh tool` / `agh toolsets` describe behavior. |
| 03 | Confirm `web/src/systems/network|session|workspace/*` fixtures and adapters consume the same DTOs the new tools render. | Confirm `core/network/*.mdx`, `core/sessions/*.mdx`, `core/workspaces/*.mdx`, plus CLI references for `network/`, `session/`, `workspace/`. |
| 04 | Confirm `web/src/systems/bridges/*` fixtures continue to type-check; observe/memory consumers (if any) compile. | Confirm `core/memory/*.mdx`, `core/bridges/*.mdx`, plus CLI references for `memory/`, `observe/`, `bridge/`. |
| 05 | Confirm `web/src/systems/settings/*` settings-config consumers compile against any DTO changes. | Confirm `configuration/config-toml.mdx`, `configuration/agent-md.mdx`, plus CLI references for `config/`. |
| 06 | None expected unless the hooks settings UI changed. | Confirm `core/hooks/*.mdx` plus `cli-reference/hooks/`. |
| 07 | Confirm `web/src/systems/automation/*` adapters/components/tests against the regenerated automation DTOs. | Confirm `core/automation/*.mdx` plus `cli-reference/automation/`. |
| 08 | Confirm extension-related shared DTOs compile; no dedicated extension UI is required. | Confirm `core/extensions/*.mdx` plus `cli-reference/extension/`. |
| 09 | Run Vitest lanes for `web/src/systems/tasks/*` and confirm `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, and `web/src/systems/tasks/mocks/fixtures.ts` no longer reference `claim_token`. | Confirm `core/autonomy/task-runs-and-leases.mdx` and `cli-reference/task/{next,heartbeat,complete,fail,release}` describe the `run_id`-keyed contract with no `--claim-token`. |
| 10 | Confirm `web/src/systems/settings/*` reflects redacted MCP auth status. | Confirm `cli-reference/mcp/auth/*` and `core/configuration/mcp-json.mdx` describe the status-only tool plus operator login/logout. |
| 11 | Run `bun-typecheck` and `bun-test` to confirm no consumer drift. | Run `make cli-docs`, `bun run --cwd packages/site build`, and the source-tests including `runtime-tools-canonical-docs.test.ts`. |

## Traceability Matrix

The full mapping lives in `tools-refac-traceability.md`. The summary mapping is:

| Case | Priority | Surface | Proves | Source |
|------|----------|---------|--------|--------|
| TC-FUNC-001 | P0 | Daemon/registry/HTTP/UDS/hosted MCP | Default discovery overlay + per-call policy recompute | task_01, ADR-001, ADR-002 |
| TC-FUNC-002 | P1 | Daemon prompt + skills bundle | Tools prompt section, bundled `agh-tools-guide`, catalog text references `agh__skill_view` first | task_02, ADR-001 |
| TC-FUNC-003 | P1 | Tools/CLI/HTTP/UDS | Read-surface coverage parity for coordination/session/workspace/memory/observe/bridge | tasks 03-04, ADR-001, ADR-002 |
| TC-FUNC-004 | P0 | Tools/CLI/HTTP/UDS | Config mutation tool family + trust-root/secret denial parity | task_05, ADR-002, ADR-006 |
| TC-FUNC-005 | P0 | Tools/CLI/HTTP/UDS | Hook management tool family + source-immutable denial | task_06, ADR-002, ADR-006 |
| TC-FUNC-006 | P0 | Tools/CLI/HTTP/UDS | Automation tool family CRUD/trigger/run inspection parity | task_07, ADR-006 |
| TC-FUNC-007 | P0 | Tools/CLI/HTTP/UDS | Extension lifecycle tool family + trust-source/rollback parity | task_08, ADR-004, ADR-006 |
| TC-FUNC-008 | P0 | Tools/CLI/HTTP/UDS/settings | MCP auth status tool + redaction + management-only login/logout | task_10, ADR-004 |
| TC-INT-001 | P1 | Daemon/registry | Operator vs session projection divergence under deny/unavailable/hook-blocked tools | task_01, ADR-002 |
| TC-INT-002 | P0 | Tools/CLI/HTTP/UDS/hosted MCP | `list/search/info/invoke` parity across surfaces for the expanded built-in catalog | tasks 01-10, ADR-002 |
| TC-INT-003 | P0 | Hosted MCP / sessions API | `tools/list` ≡ `GET /api/sessions/{id}/tools` (set + ordering + reasons) | task_10, ADR-002 |
| TC-INT-004 | P0 | Hosted MCP / approval bridge | Approval bridge timeout/cancel/disconnect + no client-supplied approval credentials | task_10 |
| TC-INT-005 | P1 | Daemon/registry | Hook denial / source-health denial reason codes flow into operator projection | tasks 01, 04, 06 |
| TC-INT-006 | P1 | Daemon/registry | Cache invalidates on agent/lineage/hook/source-health/MCP-auth-health/config-overlay changes | task_01, ADR-002 |
| TC-SEC-001 | P0 | All AGH-owned surfaces | Raw `claim_token` redaction sweep | task_09, ADR-005 |
| TC-SEC-002 | P0 | MCP auth status tool / settings / logs | MCP auth status redaction (no token/PKCE/code/callback) | task_10, ADR-004 |
| TC-SEC-003 | P0 | `agh__network_send` | Raw token payload/metadata rejected with `network_raw_token_rejected` | task_09 |
| TC-SEC-004 | P0 | Tools/CLI/HTTP/UDS | Config trust-root/secret/scope denials | task_05, ADR-006 |
| TC-SEC-005 | P0 | Tools/CLI/HTTP/UDS | Hook secret-input denials + source-immutable hooks | task_06, ADR-006 |
| TC-SEC-006 | P0 | Hosted MCP bind | Bind nonce + UDS peer-creds + AGH binary path validation; foreign processes fail closed | task_10 |
| TC-AUT-001 | P0 | Tools/CLI/HTTP/UDS | Session-bound autonomy claim → heartbeat → complete/fail/release flow | task_09, ADR-003, ADR-005 |
| TC-AUT-002 | P0 | Tools/CLI/HTTP/UDS | `AUTONOMY_FOREIGN_RUN` for cross-session lease attempts | task_09, ADR-005 |
| TC-AUT-003 | P0 | Tools | `AUTONOMY_LEASE_ALREADY_HELD` on second `run_claim_next` from same session | task_09, ADR-005 |
| TC-AUT-004 | P0 | Tools/CLI/HTTP/UDS | `AUTONOMY_NO_ACTIVE_LEASE` and `AUTONOMY_LEASE_EXPIRED` paths | task_09, ADR-005 |
| TC-AUT-005 | P0 | Daemon/task service | Concurrent heartbeats from two sessions yield one success and one deterministic mismatch | task_09, ADR-005 |
| TC-AUT-006 | P0 | Tools/CLI/HTTP/UDS | Tool/CLI/HTTP/UDS converge on the same `task.Service` lease writers | task_09, ADR-003 |
| TC-REG-001 | P0 | Codegen | `make codegen-check` clean + OpenAPI no longer mentions `claim_token` for AGH-owned surfaces | task_09, task_11 |
| TC-REG-002 | P0 | CLI docs | `make cli-docs` regenerates without drift; hand-authored `index.mdx`/`meta.json` keep new groups listed | task_11 |
| TC-REG-003 | P0 | `packages/site` build | `bun run build` + source tests pass for rewritten runtime docs | task_11 |
| TC-REG-004 | P0 | Web tasks system | `bun-typecheck` + Vitest lanes pass; no `claim_token` in `web/src/systems/tasks/*` | task_09, task_11 |
| TC-REG-005 | P1 | Skills/prompt | Catalog and `agh-agent-setup` outputs reference `agh__skill_view` and tool-first discovery, no CLI-first prose | task_02, task_11 |
| TC-UI-001 | P1 | Web settings/automation | Spot-check that automation/settings UI render against post-cut DTOs without runtime errors | tasks 07, 09, 10, 11 |
| TC-AUDIT-001 | P0 plan / P1 exec | Dossier | Self-audit: every task 01-11 maps to ≥1 scenario and ≥1 regression hot spot; required surfaces + negatives covered | task_12 |

## Deliverables

- This feature QA plan.
- `tools-refac-regression.md` smoke / targeted / full lane plan.
- `tools-refac-traceability.md` task → TC mapping with regression hot spots.
- `tools-refac-codegen-and-docs.md` codegen + docs + downstream verification dossier.
- `tools-refac-redaction-suite.md` redaction sweep procedure.
- Manual cases under `qa/test-cases/TC-*.md`.
- Reserved `issues/`, `screenshots/`, and `logs/` evidence locations for task_13.
- Task_13-ready P0/P1 execution order plus the codegen/docs/web verification checklist.
