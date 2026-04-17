# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Write four Diataxis how-to operations pages for AGH Runtime:
  - daemon lifecycle and service management
  - SQLite database administration
  - practical troubleshooting
  - production readiness checklist
- Create `packages/site/content/runtime/operations/meta.json` with daemon -> database -> troubleshooting -> production ordering.
- Verify site build and browser rendering before tracking updates or any commit.

## Important Decisions
- Treat pages as operator how-to guides, not reference pages; use concrete commands and current runtime behavior.
- Reconcile QMD/archived plan details against current code before documenting them.
- Add `operations` to `packages/site/content/runtime/meta.json` so the new section is visible in runtime navigation; keep `operations/meta.json` responsible for in-section ordering.

## Learnings
- Pre-change signal: `packages/site/content/runtime/operations/` does not exist.
- Shared workflow memory says the correct site build selector is `bunx turbo run build --filter=@agh/site`; the task's `--filter=packages/site` selector is stale.
- QMD status is available, but archived/ledger searches for operations-specific material were mostly empty; one old plan mentions detached daemon UX and persistent `~/.agh/logs/agh.log`, which still needs verification against current code.
- Verified daemon facts: `agh daemon start` defaults to detached mode, polls readiness every 100 ms for up to 15s, writes detached child stdout/stderr to `$AGH_HOME/logs/agh.log`, and foreground mode is the right service-manager entrypoint.
- Verified lock/socket facts: `$AGH_HOME/daemon.lock` uses advisory flock and stores a PID; the live UDS socket defaults to `$AGH_HOME/daemon.sock` and is chmodded to `0600`.
- Verified storage facts: `$AGH_HOME/agh.db` is the global catalog; `$AGH_HOME/sessions/<session-id>/events.db` stores the per-session event stream; SQLite runs in WAL mode and close paths checkpoint WAL.
- No current CLI command named `agh config`; the production checklist uses direct config-file review plus foreground startup for validation feedback instead.
- Task-scoped site build passed with `bunx turbo run build --filter=@agh/site` and generated 174 static pages.
- The literal task build selector `bunx turbo run build --filter=packages/site` is stale and fails because no workspace package is named `packages/site`.
- Browser QA passed with `make site-dev` and `agent-browser`: all four operations routes returned 200, the Operations sidebar rendered, a representative `/runtime/reference/config-toml/` internal link resolved, and `agent-browser errors` was empty.
- Full `make verify` still fails before task completion in unrelated `web/src/styles.test.ts` token assertions (`#121212/#1C1C1E/#2C2C2E` expected; `#141312/#1e1c1b/#2e2c2b` present in `packages/ui/src/tokens.css`).

## Files / Surfaces
- Authored docs: `packages/site/content/runtime/operations/daemon.mdx`, `database.mdx`, `troubleshooting.mdx`, `production.mdx`, `meta.json`.
- Runtime navigation touched: `packages/site/content/runtime/meta.json`.
- Source-of-truth code to inspect: `internal/daemon/`, `internal/store/`, `internal/store/globaldb/`, `internal/store/sessiondb/`, `internal/config/paths.go`, `internal/logger/`, `internal/api/udsapi/`, daemon CLI commands.

## Errors / Corrections
- Removed a stale draft command that referenced non-existent `agh config validate/show`.
- Do not mark task tracking complete or create the auto-commit while `make verify` is red from the unrelated web token mismatch.

## Ready for Next Run
- Operations docs are authored and task-scoped verification passed. Next run should either fix the unrelated full-gate blocker with explicit scope, or report the blocker and leave task tracking/commit unchanged.
