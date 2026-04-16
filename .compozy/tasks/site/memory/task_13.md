# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create automation runtime docs for jobs, triggers, webhooks, and sidebar metadata under `packages/site/content/runtime/automation/`.
- Ground the docs in current `internal/automation/`, API contracts/routes, CLI behavior, and site MDX conventions.
- Required verification includes site build, browser QA on all touched routes, full `make verify` if possible, self-review, tracking updates, and one local commit only after clean verification.

## Important Decisions
- Current implementation is source of truth; archived automation QMD material is context only and may be stale.
- Prior docs tasks found `turbo run build --filter=packages/site` stale; verify the current package selector and prefer `bunx turbo run build --filter=@agh/site` if needed.
- Document actual trigger behavior: triggers do not bind to jobs by ID today; they define their own agent, prompt template, filters, retry, and fire limit, then use the shared dispatcher path.

## Learnings
- Shared workflow memory records a known full-repo `make verify` blocker in `web/src/styles.test.ts` caused by a design-token expectation mismatch; this task must rerun and report current evidence.
- Automation defaults: timezone `UTC`, max concurrent jobs `5`, retry `none`, fire limit `12/1h`, webhook freshness `5m`.
- Schedule modes are `cron`, `every`, `at`; cron is 5-field minute/hour/day/month/dow with no seconds field. Past `at` schedules are not backfilled and one-shot jobs unregister after firing.
- Run statuses are `scheduled`, `running`, `delegated`, `completed`, `failed`, and `canceled`.
- Webhook delivery routes are HTTP-only under `/api/webhooks/global/:endpoint` and `/api/webhooks/workspaces/:workspace_id/:endpoint`; management routes exist over HTTP and UDS.
- Config-backed jobs/triggers are read-only except for enabled overlays; dynamic jobs/triggers can be created, edited, and deleted through API/CLI.

## Files / Surfaces
- Authored docs: `packages/site/content/runtime/automation/{jobs,triggers,webhooks}.mdx`, `packages/site/content/runtime/automation/meta.json`.
- Updated nav surface: `packages/site/content/runtime/meta.json`.
- Runtime/API/CLI surfaces to inspect: `internal/automation/`, `internal/api/contract/`, `internal/api/core/`, `internal/api/httpapi/`, `internal/api/udsapi/`, `internal/cli/`.
- Source exploration covered `internal/automation/model`, `dispatch.go`, `schedule.go`, `trigger.go`, `manager.go`, `internal/config/automation.go`, `internal/api/core/automation.go`, `internal/api/contract/automation.go`, and `internal/cli/automation.go`.

## Errors / Corrections
- Corrected trigger filter docs: `kind` matches the activation event name, not dispatcher kind.
- Corrected webhook docs: `X-AGH-Webhook-Delivery-ID` is required; missing headers map to 400 validation errors, while stale/future timestamps and bad signatures map to 401.
- Corrected webhook payload docs: JSON object fields are exposed under `.Data`, and `.Data.payload` is also added when the JSON object does not already define `payload`.

## Ready for Next Run
- Site docs build passed with `bunx turbo run build --filter=@agh/site`; task-provided `bunx turbo run build --filter=packages/site` still fails because no package has that workspace name.
- Browser QA passed via `make site-dev` and `agent-browser` for `/runtime/automation/jobs`, `/runtime/automation/triggers`, and `/runtime/automation/webhooks`; sidebar navigation from webhooks to jobs also worked.
- Full `make verify` failed in pre-existing `web/src/styles.test.ts` token assertions: expected `#121212/#1C1C1E/#2C2C2E`, current CSS contains `#141312/#1e1c1b/#2e2c2b`.
- Because the full gate is not clean, task tracking was not marked complete and no commit was created.
