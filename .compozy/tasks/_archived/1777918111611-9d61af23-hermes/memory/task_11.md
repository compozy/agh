# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Execute Hermes task 11 as the final QA gate using `qa-output-path=.compozy/tasks/hermes`.
- Required deliverables: real backend/CLI/API/SSE/web/site-doc evidence, fresh verification report, bug issue files/logs/screenshots when applicable, root-cause fixes plus regression coverage for discovered bugs, clean final verification, tracking updates, and one local commit.

## Important Decisions

- Use task 10 QA artifacts under `.compozy/tasks/hermes/qa/test-plans/` and `test-cases/` as the execution matrix seed.
- Keep task 11 evidence under `.compozy/tasks/hermes/qa/` rather than creating a new artifact root.

## Learnings

- TC-SEC-001 found a real bug: TOML config overlays rejected documented remote MCP fields (`transport`, `url`, `auth.*`) even though the canonical model, docs, validator, CLI, and settings API supported them.
- Root cause fix added remote MCP overlay decoding/merge support in `internal/config/merge.go` and regression coverage in `internal/config/config_test.go`.
- Post-fix live MCP redaction evidence passes: `agh config validate`, redacted config output, `agh mcp auth status`, daemon settings API, and a no-secret/token-material check.
- TC-FUNC-001 found a real bug: fresh global DB schema created a legacy `memory_operation_log` without `scope`, `workspace_root`, and `filename`, causing real `agh memory write` to fail with `no such column: scope`.
- Root cause fix added global schema migration v6 `add_memory_operation_scope` and schema assertions in `internal/store/globaldb/global_db_test.go`.
- Post-fix live memory evidence passes: global/workspace writes, CLI health/history, HTTP health/history/list, and body-content redaction check.
- TC-FUNC-002 setup/config evidence passes through focused CLI/config tests and live `config path/list/get/validate/check/set/show`, `update`, and idempotent `uninstall` flows.
- TC-FUNC-003 environment/extension evidence passes through focused tests and live `config validate --repair-env`, extension install/list/status, settings API, and redaction checks.
- TC-REG-001 release-config evidence passes through the GoReleaser config integrity test and inspection for Homebrew cask, deb/rpm, checksums, cosign signing, SBOM, and site-doc mentions.
- TC-REG-002 initially failed because site landing tests asserted stale copy/card count; fixed the test to match the current bento heading and four-card extensibility section, tracked as BUG-003.
- Final integration initially found BUG-004: HTTP prompt request cancellation canceled the prompt context before detached drain, dropping terminal events. Fixed `internal/api/httpapi/prompt.go` so cancellation happens after bounded drain, with updated handler/drain tests.
- Final integration found BUG-005: CLI TOON session-list integration expected the pre-Hermes header. Updated the assertion to include current `failure_kind` diagnostics.
- Final integration found BUG-006: reference prompt-enhancer E2E installed from a local dev source whose SDK dependency symlink escaped the hardened extension source root. Fixed the test fixture to build a packaged temp source with materialized SDK files.
- Final daemon-served web E2E found BUG-007: the app route motion shell keyed from mutable `router.latestLocation`, which could catch up during a dialog open click and remount the Jobs route, losing editor state. Fixed route keying to use reactive `useLocation` and added route-shell regression coverage.
- Final gates pass: `make verify`, `make test-integration`, `make test-e2e-runtime`, `make test-e2e-web`, web lint/typecheck/test, and site test/typecheck/build/browser docs review.

## Files / Surfaces

- `.compozy/tasks/hermes/qa/`
- `.compozy/tasks/hermes/task_11.md`
- `.compozy/tasks/hermes/_tasks.md`
- `.compozy/tasks/hermes/qa/verification-report.md`
- `.compozy/tasks/hermes/qa/issues/BUG-001-remote-mcp-toml-overlay.md`
- `.compozy/tasks/hermes/qa/issues/BUG-002-memory-operation-log-schema.md`
- `.compozy/tasks/hermes/qa/issues/BUG-003-site-landing-test-drift.md`
- `.compozy/tasks/hermes/qa/issues/BUG-004-http-prompt-drain-cancel.md`
- `.compozy/tasks/hermes/qa/issues/BUG-005-cli-session-list-toon-header.md`
- `.compozy/tasks/hermes/qa/issues/BUG-006-reference-extension-sdk-symlink.md`
- `.compozy/tasks/hermes/qa/issues/BUG-007-automation-edit-dialog-route-remount.md`
- `internal/config/merge.go`
- `internal/config/config_test.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/api/httpapi/prompt.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/httpapi/stream_helpers_test.go`
- `internal/cli/cli_integration_test.go`
- `internal/extension/reference_integration_test.go`
- `packages/site/components/landing/__tests__/landing.test.tsx`
- `web/src/routes/_app.tsx`
- `web/src/routes/-_app.test.tsx`
- `.goreleaser.yml` and release docs were inspected only; no Task 11 release changes were required.

## Errors / Corrections

- `agh daemon restart` is not a valid command; restart evidence uses explicit `agh daemon stop` followed by `agh daemon start`.
- A non-interactive `agh install` run cannot open a new TTY; retry setup lifecycle validation through a PTY or record the CLI-mode constraint explicitly.
- Initial MCP redaction check expected `auth_status.state`; actual contract uses `auth_status.status`, so the evidence checker was corrected without changing product code.
- Initial memory live-flow script resolved `bin/agh` relative to the temp workspace; corrected to use an absolute binary path. The subsequent `no such column: scope` failure was a product schema bug and was fixed in production code.

## Ready for Next Run

- Task 11 implementation and QA execution are complete pending tracking-file updates and local commit creation.
