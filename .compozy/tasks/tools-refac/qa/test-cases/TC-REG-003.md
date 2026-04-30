# TC-REG-003: `packages/site` Build And Source Tests

**Priority:** P0 (Critical)
**Type:** Regression / Docs
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the rewritten runtime documentation builds, the source tests pass, and the new `runtime-tools-canonical-docs.test.ts` regression covers the updated tool-first guidance and CLI references.

## Traceability

- Task: task_11.
- TechSpec: "Docs And Generated Surfaces", "Delete Targets".
- ADRs: ADR-001..ADR-006.
- Surfaces: `packages/site/`, `packages/site/lib/runtime-tools-canonical-docs.test.ts`.

## Preconditions

- Bun deps installed (`bun install` at repo root).
- Working tree clean for `packages/site` after running TC-REG-001 and TC-REG-002.

## Test Steps

1. Generate sources:
   ```bash
   cd packages/site && bun run source:generate | tee ../../.compozy/tasks/tools-refac/qa/logs/TC-REG-003/source-generate.log
   ```

2. Workspace typecheck (workspace-only smoke; the canonical gate is `make bun-typecheck` from root):
   ```bash
   bun run --cwd packages/site typecheck | tee qa/logs/TC-REG-003/site-typecheck.log
   ```

3. Site source tests:
   ```bash
   bun run --cwd packages/site test | tee qa/logs/TC-REG-003/site-test.log
   ```
   - **Expected:** All tests pass; `runtime-tools-canonical-docs.test.ts` succeeds; no broken-link or missing-source diagnostics.

4. Build:
   ```bash
   bun run --cwd packages/site build | tee qa/logs/TC-REG-003/site-build.log
   ```
   - **Expected:** Build succeeds.

5. Spot-check rewritten pages:
   ```bash
   grep -RIn "claim_token[^_]" packages/site/content/runtime | tee qa/logs/TC-REG-003/runtime-grep.txt
   grep -RIn "claim_token_hash" packages/site/content/runtime | tee qa/logs/TC-REG-003/runtime-hash-grep.txt
   ```
   - **Expected:** First grep returns zero matches. Second grep returns observability mentions only.

6. Confirm tool-first language replaced opt-in / CLI-first prose. Manually inspect:
   - `core/configuration/agent-md.mdx`
   - `core/configuration/config-toml.mdx`
   - `core/agents/definitions.mdx`
   - `core/autonomy/task-runs-and-leases.mdx`
   - `core/hooks/event-catalog.mdx`
   - `core/automation/*.mdx`
   - `core/extensions/*.mdx`
   - `core/memory/*.mdx`
   - `core/network/*.mdx`
   - `cli-reference/mcp/auth/*.mdx`

7. Confirm new pages exist for the canonical tool families: search the `core/` tree for documentation referencing `agh__memory_*`, `agh__sessions_*`, `agh__workspace_*`, `agh__config_*`, `agh__hooks_*`, `agh__automation_*`, `agh__extensions_*`, `agh__autonomy_*`, `agh__mcp_auth_status`, `agh__observe_*`, `agh__bridges_*`.

## Evidence To Capture

- All logs above.
- `qa/logs/TC-REG-003/runtime-grep.txt`, `runtime-hash-grep.txt`.
- Manual inspection notes recorded in `qa/logs/TC-REG-003/manual-spot-checks.md`.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Broken link to deleted target | autonomy doc refers to removed path | Source test fails; fix link before passing |
| Leftover prose mentioning `--claim-token` | generated CLI ref already clean but `core/` page stale | Manually update `core/` page; re-run source tests |
| Build failure due to missing image | broken asset reference | Fix asset; re-run build |

## Channels Exercised

- `packages/site` Bun pipeline.

## Related Test Cases

- TC-REG-001 (codegen drift).
- TC-REG-002 (cli-docs drift).
- TC-REG-005 (catalog text + agh-agent-setup regression).
