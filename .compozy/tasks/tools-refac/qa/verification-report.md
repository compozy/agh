# Verification Report (placeholder)

**qa-output-path:** `.compozy/tasks/tools-refac`
**Status:** Pending — task_13 has not yet executed the dossier.
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

This file is a scaffold. Task_13 (Real-Scenario QA Execution) replaces it with execution evidence.

## Lab Coordinates

- `AGH_HOME`: _populated by task_13_
- Daemon ports: _populated by task_13_
- `tmux-bridge` socket: _populated by task_13_
- `PROVIDER_HOME` / `PROVIDER_CODEX_HOME`: _populated by task_13_
- `AGH_WEB_API_PROXY_TARGET`: _populated by task_13_
- Bootstrap manifest path: `qa/logs/<TC-ID>/bootstrap-manifest.json` _produced by `agh-qa-bootstrap`_

## Smoke Lane

| Order | Case | Result | Evidence |
|-------|------|--------|----------|
| 1 | TC-FUNC-001 | _pending_ | _logs path_ |
| 2 | TC-INT-003 | _pending_ | _logs path_ |
| 3 | TC-AUT-001 | _pending_ | _logs path_ |
| 4 | TC-SEC-001 | _pending_ | _logs path_ |
| 5 | TC-SEC-002 | _pending_ | _logs path_ |
| 6 | TC-FUNC-004 | _pending_ | _logs path_ |
| 7 | TC-REG-001 | _pending_ | _logs path_ |

## Targeted Lanes

| Lane | Cases | Result Summary |
|------|-------|----------------|
| Discovery / Policy / Prompt | TC-FUNC-001, TC-INT-001, TC-INT-006, TC-FUNC-002, TC-REG-005 | _pending_ |
| Read Surfaces | TC-FUNC-003, TC-INT-002, TC-INT-005 | _pending_ |
| Mutable Surfaces | TC-FUNC-004..007, TC-SEC-004, TC-SEC-005 | _pending_ |
| Autonomy | TC-AUT-001..006, TC-SEC-001, TC-SEC-003 | _pending_ |
| Hosted MCP / Approval | TC-INT-003, TC-INT-004, TC-FUNC-008, TC-SEC-002, TC-SEC-006 | _pending_ |
| Codegen / Docs / Web | TC-REG-001..005, TC-UI-001 | _pending_ |

## Repository Gates

- `make verify`: _pending_
- `make codegen-check`: _pending_
- `make cli-docs` + format clean: _pending_
- `bun run --cwd packages/site build`: _pending_
- `make bun-typecheck` / `make bun-test`: _pending_
- `make test-e2e-runtime`: _pending_

## Open Issues

_Populated from `qa/issues/BUG-*.md` once task_13 records defects._

## Final Verdict

_Populated by task_13 after the full regression lane. Acceptable values: PASS / FAIL / CONDITIONAL._
