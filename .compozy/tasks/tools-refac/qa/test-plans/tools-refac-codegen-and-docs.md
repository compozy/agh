# Codegen, Docs, And Downstream Web Verification Dossier

**qa-output-path:** `.compozy/tasks/tools-refac`
**Status:** Planning complete, not executed
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

This document is the authoritative checklist task_13 follows when verifying generated artifacts, site documentation, and the downstream web consumers affected by the autonomy hard cut and the broadened built-in tool surface. It is referenced by TC-REG-001..005 and TC-UI-001.

## Why this exists

`tools-refac` co-ships contract changes (autonomy hard cut), CLI surface changes (new `tool/` and `toolsets/` groups, autonomy command flag deletions), and site documentation changes that delete stale `claim_token` and CLI-first prose. The verification surfaces are different per artifact and follow strict ordering — codegen → docs regeneration → format → site build → web typecheck/test — because each step's input is the previous step's output.

## Step 0 — Prerequisites

| Requirement | Evidence |
|-------------|----------|
| Working tree clean before regeneration | `git status` snapshot under `qa/logs/<TC-ID>/git-status-pre.txt` |
| Bun deps installed | `bun install` log under `qa/logs/<TC-ID>/bun-install.log` |
| Go deps verified | `make deps` log under `qa/logs/<TC-ID>/make-deps.log` |
| Isolated `AGH_HOME` from `agh-qa-bootstrap` | `bootstrap-manifest.json` under `qa/logs/<TC-ID>/` |

## Step 1 — Backend / OpenAPI Codegen (TC-REG-001)

1. Run `make codegen`.
2. Capture the diff: `git status --porcelain` and `git diff --stat`.
3. Capture explicit diffs against the autonomy contract: `git diff openapi/agh.json web/src/generated/agh-openapi.d.ts`.
4. Run `make codegen-check` and assert it exits 0.
5. Grep the OpenAPI for legacy fields: `grep -n "claim_token" openapi/agh.json` and `grep -n "claim_token" web/src/generated/agh-openapi.d.ts`. Both must return zero matches **except** for the observability-only `claim_token_hash` field, which remains intentional.
6. Confirm new autonomy contract shapes use `run_id` and the agent task request DTOs no longer contain `ClaimToken`.

Pass criteria:
- Diff after `make codegen` is empty.
- `make codegen-check` exits 0.
- Only `claim_token_hash` survives in the generated artifacts; raw `claim_token` is absent.
- AGH-owned autonomy routes (`/api/agents/tasks/runs/...`) accept `run_id` in path/body and return responses without raw token material.

Evidence files:
- `qa/logs/TC-REG-001/make-codegen.log`
- `qa/logs/TC-REG-001/make-codegen-check.log`
- `qa/logs/TC-REG-001/openapi-claim-token-grep.txt`

## Step 2 — CLI Reference Regeneration (TC-REG-002)

1. Run `make cli-docs`.
2. Capture the diff: `git status --porcelain packages/site/content/runtime/cli-reference/` and `git diff --stat packages/site/content/runtime/cli-reference/`.
3. Run `bun run --cwd packages/site format` (or `make bun-lint`) to apply oxfmt to regenerated tables.
4. Capture the diff again — it must be empty (i.e., committed tree already matched the formatted regeneration).
5. Confirm hand-authored landing pages survive: `packages/site/content/runtime/cli-reference/index.mdx` still lists `tool/` and `toolsets/` groups; `packages/site/content/runtime/cli-reference/meta.json` still includes those groups.
6. Grep for legacy autonomy guidance:
   - `grep -RIn "--claim-token" packages/site/content/runtime/cli-reference/task` must return zero matches.
   - `grep -RIn "claim_token" packages/site/content/runtime/cli-reference/task` returns either zero matches or only `claim_token_hash` references.
7. Confirm new tool-callable command groups have docs:
   - `tool/` (list, search, info, invoke, mcp) — present and post-format consistent.
   - `toolsets/` (list, info) — present.
   - `mcp/auth/status` — present (login/logout remain documented as operator-only).
   - Mutable command families (`config/set|unset|diff`, `hooks/{create,update,delete,enable,disable}`, `automation/{jobs,triggers,runs}`, `extension/{install,update,remove,enable,disable}`).

Pass criteria:
- `make cli-docs` produces no diff after `bun run format`.
- Hand-authored landing pages keep new groups listed.
- No `--claim-token` flag references in autonomy command docs.

Evidence files:
- `qa/logs/TC-REG-002/make-cli-docs.log`
- `qa/logs/TC-REG-002/cli-reference-diff.txt`
- `qa/logs/TC-REG-002/cli-reference-claim-token-grep.txt`

## Step 3 — Site Build (TC-REG-003)

1. `cd packages/site && bun run source:generate` and capture the log.
2. `cd packages/site && bun run typecheck` (workspace-only smoke).
3. `bun run --cwd packages/site test` to run site source tests, including `packages/site/lib/runtime-tools-canonical-docs.test.ts`.
4. `cd packages/site && bun run build` and capture the log.
5. Spot-check rewritten runtime pages from the `Post-Implementation Residual Checks` section — they must no longer contain stale opt-in/CLI-first prose:
   - `core/configuration/agent-md.mdx`
   - `core/configuration/config-toml.mdx`
   - `core/agents/definitions.mdx`
   - `core/autonomy/task-runs-and-leases.mdx`
   - `core/hooks/event-catalog.mdx`
   - `cli-reference/task/{next,heartbeat,complete,fail,release}.mdx`
   - `cli-reference/mcp/auth/*.mdx`
6. Confirm runtime core pages reference the canonical surface families (`agh__memory`, `agh__sessions`, `agh__workspace`, `agh__config`, `agh__hooks`, `agh__automation`, `agh__extensions`, `agh__autonomy`, `agh__mcp_auth`, `agh__observe`, `agh__bridges`).

Pass criteria:
- `bun run build` succeeds with no broken-link or source-test failure.
- `runtime-tools-canonical-docs.test.ts` passes.
- Rewritten pages no longer mention raw `claim_token` flow or CLI-first AGH-internal patterns.

Evidence files:
- `qa/logs/TC-REG-003/site-source-generate.log`
- `qa/logs/TC-REG-003/site-typecheck.log`
- `qa/logs/TC-REG-003/site-test.log`
- `qa/logs/TC-REG-003/site-build.log`

## Step 4 — Web `tasks` System Regression (TC-REG-004)

1. Confirm regenerated artifacts: `web/src/generated/agh-openapi.d.ts` from Step 1.
2. Grep web tree:
   - `grep -RIn "claim_token" web/src/systems/tasks` — must return zero matches (or only `claim_token_hash` for observability).
   - `grep -RIn "claim_token" web/src/systems/tasks/mocks/fixtures.ts` — zero matches except `claim_token_hash`.
   - `grep -RIn "ClaimToken" web/src` — zero matches.
3. Run `make bun-typecheck` (root) — gates the entire monorepo turbo `typecheck`. This is the canonical gate the Verify pipeline runs.
4. Run focused Vitest lanes against the affected systems:
   - `bunx vitest run web/src/systems/tasks --reporter=basic`
   - `bunx vitest run web/src/systems/automation --reporter=basic`
   - `bunx vitest run web/src/systems/settings --reporter=basic`
5. Optionally run `make bun-test` for the full monorepo test set.

Pass criteria:
- `make bun-typecheck` succeeds.
- Vitest lanes for tasks/automation/settings systems pass.
- `web/src/systems/tasks/{types.ts,mocks/fixtures.ts}` has no raw `claim_token` references.

Evidence files:
- `qa/logs/TC-REG-004/bun-typecheck.log`
- `qa/logs/TC-REG-004/vitest-tasks.log`
- `qa/logs/TC-REG-004/vitest-automation.log`
- `qa/logs/TC-REG-004/vitest-settings.log`
- `qa/logs/TC-REG-004/web-claim-token-grep.txt`

## Step 5 — Skills / Prompt Catalog (TC-REG-005)

1. Inspect `internal/skills/catalog.go` and confirm the catalog text branches: when `agh__skill_view` is callable, the catalog must teach `agh__skill_view`; when denied, the conditional CLI fallback may appear.
2. Inspect `internal/skills/bundled/skills/agh-agent-setup/SKILL.md` and confirm it does not describe `agh__catalog` as opt-in-only.
3. Inspect `internal/skills/bundled/skills/agh-tools-guide/SKILL.md` (or directory contents) and confirm it teaches `agh__tool_search → agh__tool_info → invoke`.
4. Run `go test ./internal/skills ./internal/skills/bundled` to exercise catalog text and bundled-content tests.
5. Run `go test ./internal/daemon -run "Prompt|Section"` to confirm `HarnessPromptSectionTools` orders correctly relative to `skills` and `network`.

Pass criteria:
- Catalog text references `agh__skill_view` first.
- `agh-tools-guide` exists and teaches the canonical loop.
- `HarnessPromptSectionTools` rendering order is enforced by tests.
- Source tests for skills/bundled and daemon prompt assembly pass.

Evidence files:
- `qa/logs/TC-REG-005/skills-catalog-tests.log`
- `qa/logs/TC-REG-005/daemon-prompt-tests.log`
- `qa/logs/TC-REG-005/agh-tools-guide-snapshot.md` (snapshot of the skill content)

## Step 6 — Spot-Check Web UI (TC-UI-001)

This step is conditional. Only execute when QA evidence requires manual UI verification (e.g., automation/settings DTO change suspected).

1. Start the daemon from the isolated `AGH_HOME`.
2. Export `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest.
3. `make web-dev` (or `bun run --cwd web dev`).
4. Open the web UI at the dev URL.
5. Spot-check:
   - Automation panel renders job/trigger/run lists without console errors.
   - Settings → MCP servers shows `auth_status` with redacted values (`token_present: true|false`).
   - Tasks views render without referencing `claim_token`.
6. Capture screenshots under `qa/screenshots/TC-UI-001/`.

Pass criteria:
- No console errors related to missing/extra DTO fields.
- Settings MCP page reflects redacted auth status.
- Tasks views render the `run_id`-keyed contract.

## Aggregated Verification Checklist

Use this checklist as the final pass before publishing the verification report.

- [ ] `make codegen` produces no diff (TC-REG-001).
- [ ] `make codegen-check` exits 0 (TC-REG-001).
- [ ] OpenAPI + `web/src/generated/agh-openapi.d.ts` contain no raw `claim_token` field for AGH-owned autonomy routes (TC-REG-001).
- [ ] `make cli-docs` + `bun run format` produces no diff (TC-REG-002).
- [ ] `cli-reference/index.mdx` and `meta.json` keep `tool/` and `toolsets/` groups listed (TC-REG-002).
- [ ] No `--claim-token` flag examples remain in `cli-reference/task/*` (TC-REG-002).
- [ ] `bun run --cwd packages/site build` succeeds (TC-REG-003).
- [ ] `runtime-tools-canonical-docs.test.ts` passes (TC-REG-003).
- [ ] Site `core/{configuration,agents,autonomy,hooks,automation,extensions,memory,network,workspaces,sessions,bridges}/*.mdx` describe the canonical surface (TC-REG-003).
- [ ] `make bun-typecheck` passes against the regenerated types (TC-REG-004).
- [ ] Vitest lanes for `web/src/systems/{tasks,automation,settings}` pass (TC-REG-004).
- [ ] Web tree contains no raw `claim_token` references (TC-REG-004).
- [ ] Catalog text + `agh-agent-setup` no longer teach CLI-first / opt-in discovery (TC-REG-005).
- [ ] `agh-tools-guide` exists with the canonical loop guidance (TC-REG-005).
- [ ] Optional UI spot-check completes without DTO console errors (TC-UI-001).

Document the result inside `qa/verification-report.md` with:

- The list of executed commands and their evidence paths.
- Any deviation from the checklist with the linked `BUG-*.md`.
- The final verdict (PASS / FAIL / CONDITIONAL).
