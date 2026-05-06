# Memory v2 Slice 1 Regression Suite

**qa-output-path:** `.compozy/tasks/mem-v2`
**Status:** Planning complete, not executed
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Execution Rules

- Task_26 must activate `agh-qa-bootstrap`, `real-scenario-qa`, `qa-execution`, and `agh-worktree-isolation`.
- Create a fresh isolated lab by default. Reuse `qa/bootstrap-manifest.json` only when continuing the same active QA pass.
- Export `AGH_WEB_API_PROXY_TARGET` from the manifest or `bootstrap.env` before web QA; do not hardcode `localhost:2123`.
- Use `PROVIDER_HOME` and `PROVIDER_CODEX_HOME` from the manifest for bound-secret or brokered provider QA. Native CLI providers with `home_policy = operator` keep operator login state unless a scenario explicitly tests isolation.
- Capture all command output under `.compozy/tasks/mem-v2/qa/logs/<TC-ID>/`.
- File `qa/issues/BUG-NNN.md` for every reproduced defect before applying a fix.
- No waiver path: if a P0 invariant fails, fix production code/docs/tests and rerun the failed scenario plus affected gates.
- Run config mutations sequentially against the same QA home.
- Run `make verify` after the final fix set and before marking task_26 complete.

## Smoke Lane

Estimated duration: 45-60 minutes.

| Order | Case | Priority | Stop Condition | Minimum Evidence |
|---:|---|---|---|---|
| 1 | TC-REG-001 | P0 | Codegen/docs truth gate fails | `make codegen-check`, `make cli-docs`, focused site tests |
| 2 | TC-SCEN-001 | P0 | Controller-backed write is not searchable without undocumented reindex | CLI/UDS/API write and search payloads, DB event rows |
| 3 | TC-INT-003 | P0 | CLI/HTTP/UDS/native output diverges for same selector | Side-by-side JSON and diff |
| 4 | TC-SEC-001 | P0 | `_system`, prompt-only memory, raw trace, or secret leaks | Recall output, SSE/log grep, public payload grep |
| 5 | TC-UI-002 | P0 | Settings UI/backend config lifecycle drifts | Browser/settings evidence plus backend payload |
| 6 | TC-SCEN-002 | P0 | Snapshot reload mutates current session or ledger is missing after stop | Session transcripts, prompt evidence, `ledger.jsonl`, UI inspector |

## Targeted Runtime Lanes

### Controller, WAL, Replay, And Search Visibility

| Order | Case | Priority | Scope |
|---:|---|---|---|
| 1 | TC-SCEN-001 | P0 | Write through CLI/UDS/API, then search immediately across surfaces |
| 2 | TC-INT-001 | P0 | Decision WAL, replay, revert, idempotency, atomic writes |
| 3 | TC-INT-003 | P0 | Transport/native parity for writes, edits, deletes, decisions |

Recommended commands:

- `go test ./internal/memory ./internal/memory/controller ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli -count=1`
- `go test -race ./internal/memory ./internal/memory/controller ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1`
- `agh memory write ... -o json`, `agh memory search ... -o json`, UDS equivalents through `curl --unix-socket`, and HTTP equivalents against the isolated daemon.

### Recall, Signals, Scope, And Workspace Identity

| Order | Case | Priority | Scope |
|---:|---|---|---|
| 1 | TC-INT-002 | P0 | Deterministic recall, live signals, shadowing, stale banner, `_system` exclusion |
| 2 | TC-SCEN-002 | P0 | Snapshot, reload, sub-agent read-only, session ledger |
| 3 | TC-SEC-001 | P0 | Redaction and prompt-only memory exclusion |

Recommended commands:

- `go test ./internal/memory/recall ./internal/memory ./internal/workspace ./internal/situation -count=1`
- `go test -race ./internal/memory/recall ./internal/memory -count=1`
- Move the scenario workspace after creating `.agh/workspace.toml`; verify the same `workspace_id` and search results.

### Provider, Extension Host, Extractor, And Dreaming

| Order | Case | Priority | Scope |
|---:|---|---|---|
| 1 | TC-INT-004 | P1 | MemoryProvider registry, bundled local provider, Host API recall |
| 2 | TC-INT-005 | P1 | Extractor persisted-message hook, `_inbox`, queue, DLQ, dreaming gates/retry |

Recommended commands:

- `go test ./internal/memory/provider/... ./internal/extension ./internal/daemon -count=1`
- `go test ./internal/memory/extractor ./internal/memory/consolidation ./internal/memory ./internal/daemon -count=1`
- `go test -race -parallel=4 ./internal/extension ./internal/daemon -count=1`

## Targeted UI And Docs Lanes

| Order | Case | Priority | Scope |
|---:|---|---|---|
| 1 | TC-UI-001 | P0 | Web Knowledge list/show/edit/delete/search/decision history |
| 2 | TC-UI-002 | P0 | Web Memory Settings payload, validation, readonly daemon-managed fields |
| 3 | TC-UI-003 | P1 | Web Session Inspector forensic ledger and read-only behavior |
| 4 | TC-REG-001 | P0 | Codegen, CLI docs, API docs, runtime docs truth/discovery |

Recommended commands:

- `cd web && bunx vitest run src/routes/_app/-knowledge.test.tsx src/routes/_app/settings/-memory.test.tsx src/routes/_app/-agents.\\$name.sessions.\\$id.test.tsx`
- `cd web && bunx vitest run src/systems/knowledge src/systems/session src/hooks/routes/use-settings-memory-page.test.tsx`
- `cd packages/site && bun run test -- runtime-docs-truth runtime-docs-discovery runtime-manual-cli-examples runtime-manual-api-routes memory-v2-qa-artifacts`
- `cd packages/site && bun run build`
- Browser check through `browser-use:browser` against the isolated web dev server.

## Full Regression Lane

Estimated duration: 4-6 hours.

1. Execute the smoke lane in order.
2. Execute all remaining P0 and P1 cases in this order: TC-INT-001, TC-INT-002, TC-INT-004, TC-INT-005, TC-UI-001, TC-UI-003, TC-SEC-001, TC-REG-001.
3. Run `make test-integration` for Memory v2 integration proofs if credentials/environment permit.
4. Run `make test-e2e-runtime`.
5. Run `make test-e2e-web` with `AGH_WEB_API_PROXY_TARGET` derived from the manifest.
6. Run `make verify`.
7. Populate `.compozy/tasks/mem-v2/qa/verification-report.md` with command summaries, evidence paths, bug status, rerun outcomes, and final verdict.

## Pass, Fail, And Conditional Criteria

PASS:

- Every P0 passes with persisted evidence.
- At least 90 percent of P1 cases pass.
- `make verify`, `make codegen-check`, site build, and relevant web tests pass after the final fix set.
- No critical bug remains around data loss, redaction, search visibility, cross-surface parity, workspace identity, or prompt injection.

FAIL:

- Any P0 fails.
- Any curated memory mutation bypasses the controller.
- Any CLI/HTTP/UDS/native tool surface diverges for the same operation.
- Any `_system` artifact, raw decision body, raw LLM trace, token, or prompt-only memory leaks into public payloads, recall, SSE, web, or docs.
- Search requires an undocumented manual reindex after a supported controller-backed write.
- `make verify` fails after the final fix set.

CONDITIONAL:

- A P1 UI/docs issue remains with a `BUG-NNN.md`, owner, operator impact note, and no impact on P0 correctness.

## Regression Maintenance

- Any task_26 fix must add the narrowest durable automated regression available.
- If a new high-risk invariant appears, add a new TC and update `memory-v2-traceability.md`.
- If the same package or behavior is patched twice in this QA stream, a third change requires a structural redesign TechSpec.
