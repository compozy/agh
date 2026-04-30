# Tool Registry Canonical Surface Regression Suite

**qa-output-path:** `.compozy/tasks/tools-refac`
**Artifact root:** `.compozy/tasks/tools-refac/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Execution Rules

- Task_13 must activate `qa-execution` with `qa-output-path=.compozy/tasks/tools-refac`.
- Bootstrap a fresh isolated lab with `agh-qa-bootstrap`. Capture `bootstrap-manifest.json` before running any case. Reuse the manifest only across consecutive cases of the same active QA pass.
- Use `PROVIDER_HOME` and `PROVIDER_CODEX_HOME` from the manifest for any provider-backed case. Never point at `~/.codex`.
- For Web QA against the isolated daemon, export `AGH_WEB_API_PROXY_TARGET` derived from the manifest. Do not hardcode `localhost:2123`.
- Smoke-first execution. Any P0 failure stops the run, files `BUG-*.md`, requires a root-cause fix, then restarts smoke.
- Targeted lanes follow smoke; full regression follows targeted. Do not skip lanes after a fix touches the relevant domain.
- Do not weaken tests to make them pass. Failing invariants require code/config/docs fixes plus narrow durable regression coverage.
- Capture command output under `.compozy/tasks/tools-refac/qa/logs/<TC-ID>/`.
- Capture browser/docs screenshots (when a TC asks for them) under `.compozy/tasks/tools-refac/qa/screenshots/<TC-ID>/`.
- Record final evidence and residual risk in `.compozy/tasks/tools-refac/qa/verification-report.md`.
- After the final fix set, run `make verify` and the codegen/docs/web gates as the global gate.

## Smoke Lane

Estimated duration: 30-45 minutes.

| Order | Case | Priority | Stop Condition | Minimum Evidence |
|-------|------|----------|----------------|------------------|
| 1 | TC-FUNC-001 | P0 | Default discovery overlay missing for empty agents OR projection ≠ dispatch policy decision | Tool list payload, hosted MCP `tools/list` snapshot, `GET /api/sessions/{id}/tools` JSON |
| 2 | TC-INT-003 | P0 | Hosted MCP `tools/list` ≠ session projection | Side-by-side hosted MCP and HTTP payloads, with diff |
| 3 | TC-AUT-001 | P0 | Autonomy flow leaks raw `claim_token` OR fails to route to existing writers | CLI/HTTP/UDS/tool transcripts plus daemon log |
| 4 | TC-SEC-001 | P0 | Any AGH-owned surface emits `claim_token` text | Grep evidence across CLI human/JSON, HTTP/UDS/SSE/observe/memory/log/web fixtures |
| 5 | TC-SEC-002 | P0 | MCP auth status emits token/code/PKCE/secret material | Status payload across CLI/HTTP/UDS/settings + log grep |
| 6 | TC-FUNC-004 | P0 | Config tool mutates a forbidden trust-root/secret path OR diverges from CLI | Tool/CLI/HTTP/UDS allow + deny matrix |
| 7 | TC-REG-001 | P0 | `make codegen-check` reports drift | `make codegen-check` log |

## Targeted Lanes

Run targeted lanes after smoke passes or after a fix touches the relevant domain.

### Discovery, Policy, And Prompt

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-FUNC-001 | P0 | Default discovery overlay + per-call policy recomputation |
| 2 | TC-INT-001 | P1 | Operator vs session projection divergence |
| 3 | TC-INT-006 | P1 | Cache invalidates on agent/lineage/hook/source-health/MCP-auth-health/config-overlay changes |
| 4 | TC-FUNC-002 | P1 | Tools prompt section + `agh-tools-guide` bundling |
| 5 | TC-REG-005 | P1 | Catalog and `agh-agent-setup` reference `agh__skill_view` first |

Recommended commands for task_13 evidence:

- `go test ./internal/tools ./internal/tools/builtin ./internal/daemon`
- `go test ./internal/skills ./internal/skills/bundled`
- `agh tool list -o json` against the isolated daemon, with and without an empty-`tools` agent definition.
- `curl -s --unix-socket "$AGH_HOME/run/sock/uds.sock" http://localhost/api/sessions/$SID/tools | jq`
- `agh tool mcp --session $SID --bind-nonce $NONCE` then capture `tools/list` from the hosted MCP transport.

### Read Surfaces (Coordination, Session, Workspace, Memory, Observe, Bridges)

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-FUNC-003 | P1 | Tool/CLI/HTTP/UDS parity for the read families |
| 2 | TC-INT-002 | P0 | `list/search/info/invoke` parity across surfaces |
| 3 | TC-INT-005 | P1 | Hook denial / source-health denial reason codes |

Recommended commands:

- `go test ./internal/tools/builtin -run "TestNetwork|TestSession|TestWorkspace|TestMemory|TestObserve|TestBridges"`
- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi`
- CLI parity: `agh network status -o json`, `agh session list -o json`, `agh workspace list -o json`, `agh memory list -o json`, `agh observe events -o json`, `agh bridge list -o json` against the same isolated daemon, then compare to `tool invoke` of the equivalent `agh__*_*` ID.

### Mutable Surfaces (Config, Hooks, Automation, Extensions)

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-FUNC-004 | P0 | Config tool family + trust-root/secret denials |
| 2 | TC-FUNC-005 | P0 | Hook tool family + source-immutable denials |
| 3 | TC-FUNC-006 | P0 | Automation tool family CRUD + run inspection |
| 4 | TC-FUNC-007 | P0 | Extension lifecycle + trust-source + rollback |
| 5 | TC-SEC-004 | P0 | Config trust-root/secret/scope denial parity |
| 6 | TC-SEC-005 | P0 | Hook secret-input + source-immutable denial parity |

Recommended commands:

- `go test ./internal/config ./internal/hooks ./internal/automation ./internal/extension`
- `go test ./internal/tools/builtin -run "TestConfig|TestHook|TestAutomation|TestExtension"`
- `go test ./internal/daemon -run "TestNativeConfig|TestNativeHook|TestNativeAutomation|TestNativeExtension"`
- `agh config set <path> <value>` and the equivalent `tool invoke` for both allowed and forbidden paths.
- `agh hooks create|update|delete` for config-backed and source-owned declarations and the equivalent `tool invoke`.
- `agh automation jobs create|update|delete|trigger` and `agh automation runs list|get` plus tool equivalents.
- `agh extension search|install|update|remove|enable|disable` and tool equivalents using a fixture marketplace and a forbidden source.

### Autonomy Hard Cut

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUT-001 | P0 | Claim → heartbeat → complete flow |
| 2 | TC-AUT-002 | P0 | `AUTONOMY_FOREIGN_RUN` cross-session denial |
| 3 | TC-AUT-003 | P0 | `AUTONOMY_LEASE_ALREADY_HELD` second-claim denial |
| 4 | TC-AUT-004 | P0 | `AUTONOMY_NO_ACTIVE_LEASE` and `AUTONOMY_LEASE_EXPIRED` |
| 5 | TC-AUT-005 | P0 | Concurrent heartbeats — single success path |
| 6 | TC-AUT-006 | P0 | Tool/CLI/HTTP/UDS converge on the same writers |
| 7 | TC-SEC-001 | P0 | Cross-channel `claim_token` redaction |
| 8 | TC-SEC-003 | P0 | `agh__network_send` raw-token rejection |

Recommended commands:

- `go test ./internal/task ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- `go test ./internal/tools/builtin -run "TestAutonomy"`
- `make test-e2e-runtime` (gates the autonomy E2E lane plus full HTTP/UDS/tool/MCP path)
- `agh task next` → `agh task heartbeat <run_id>` → `agh task complete <run_id> --result …` (no `--claim-token` on any command).
- Two-session contention: open two isolated sessions, claim a single run from session A, attempt heartbeat from session B, expect `AUTONOMY_FOREIGN_RUN`.

### Hosted MCP And Approval Bridge

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-INT-003 | P0 | `tools/list` parity |
| 2 | TC-INT-004 | P0 | Approval bridge timeout/cancel/disconnect |
| 3 | TC-FUNC-008 | P0 | MCP auth status tool diagnostics |
| 4 | TC-SEC-002 | P0 | MCP auth status redaction |
| 5 | TC-SEC-006 | P0 | Hosted MCP bind nonce + UDS peer-credentials + binary validation |

Recommended commands:

- `go test ./internal/mcp ./internal/mcp/auth ./internal/daemon -run "TestHostedMCP|TestApproval|TestMCPAuth"`
- `agh mcp auth status -o json` and tool-invoke `agh__mcp_auth_status`.
- Force `approval_required=true` and exercise: ACP grant, deny, timeout, hosted MCP disconnect, ACP unavailable.
- Foreign-process bind attempts: launch a foreign local process invoking `agh tool mcp --session $SID --bind-nonce $NONCE` with mismatching peer-credentials or stale nonce; expect deterministic permission failure and no projection.

### Codegen, Docs, And Web

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-REG-001 | P0 | `make codegen-check` clean |
| 2 | TC-REG-002 | P0 | `make cli-docs` no drift |
| 3 | TC-REG-003 | P0 | `packages/site` build |
| 4 | TC-REG-004 | P0 | Web `tasks` system regression |
| 5 | TC-UI-001 | P1 | Spot-check automation/settings UI |

Recommended commands:

- `make codegen` then `make codegen-check`
- `make cli-docs` and `git status` against `packages/site/content/runtime/cli-reference/`; followed by `bun run --cwd packages/site format` to ensure tables stay aligned.
- `bun run --cwd packages/site typecheck`
- `bun run --cwd packages/site build`
- `bun-typecheck` and `bun-test` (root) — these are the gates the Verify pipeline runs.
- Vitest lanes: `bunx vitest run web/src/systems/tasks` and `bunx vitest run packages/site/lib/runtime-tools-canonical-docs.test.ts`.
- Optional UI check: `make web-dev` against the isolated daemon (with `AGH_WEB_API_PROXY_TARGET` set) and exercise automation/settings panels.

## Full Regression Lane

Estimated duration: 3-5 hours.

Execute after all smoke and targeted lanes pass or after the final task_13 fix set.

1. Run all P0 cases in smoke order.
2. Run remaining P0 / P1 in this sequence: TC-INT-001, TC-INT-002, TC-INT-005, TC-INT-006, TC-FUNC-002, TC-FUNC-003, TC-FUNC-005, TC-FUNC-006, TC-FUNC-007, TC-FUNC-008, TC-SEC-002, TC-SEC-003, TC-SEC-004, TC-SEC-005, TC-SEC-006, TC-AUT-002, TC-AUT-003, TC-AUT-004, TC-AUT-005, TC-AUT-006, TC-REG-002, TC-REG-003, TC-REG-004, TC-REG-005, TC-UI-001.
3. Run repository gate: `make verify`.
4. Run `make test-e2e-runtime` to exercise the autonomy / approval bridge / hosted MCP lane end to end.
5. Run any extra web/site commands required by files changed during task_13.
6. Populate `.compozy/tasks/tools-refac/qa/verification-report.md` with command output summaries, evidence paths, unresolved issues, and final verdict.
7. Run TC-AUDIT-001 to confirm dossier completeness invariants still hold post-execution (every task → scenario + hot spot; every required negative covered; no new TC was added to compensate for missing scope without a linked `BUG-*.md`).

## Pass, Fail, And Conditional Criteria

PASS:

- All P0 pass.
- At least 90% of P1 pass.
- `make verify`, `make codegen-check`, and the `packages/site` build pass after the last change.
- No critical bug, raw `claim_token` leak, hosted MCP projection drift, mutable-surface bypass, MCP auth secret leak, or codegen/docs mismatch remains open.

FAIL:

- Any P0 fails.
- Any AGH-owned surface emits or accepts raw `claim_token`.
- Hosted MCP `tools/list` ≠ session projection.
- Mutable tool family bypasses CLI/HTTP/UDS validators or writers.
- MCP auth status leaks token/code/PKCE/callback secret material.
- `make verify`, `make codegen-check`, or the `packages/site` build fails after the final fix set.

CONDITIONAL:

- A P1 docs or UI issue remains with a documented workaround, `BUG-*.md`, and explicit owner, while all P0 and final repository gates pass.

## Regression Maintenance

After task_13:

- Promote any reproducer into the narrowest durable Go, web, or site regression test.
- Add a new TC only if the discovered gap represents a reusable invariant not already covered by this dossier.
- Keep evidence paths stable under `.compozy/tasks/tools-refac/qa/` so future runs can diff reports across releases.
- If the same package is patched twice in one workstream, the third change must be opened as a new TechSpec per the two-touch rule.
