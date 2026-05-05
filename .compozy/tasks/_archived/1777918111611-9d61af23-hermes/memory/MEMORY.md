# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 01 has introduced reusable persistence/retry foundations in implementation: `internal/store.RunMigrations` and `internal/retry`.
- Task 02 has introduced config-driven observability retention and typed health base fields in implementation.
- Task 03 has introduced typed lifecycle failure diagnostics, redacted crash bundles, and downstream ACP agent probes across store/session/acp/observe/API/CLI/web/docs surfaces.
- Task 04 has introduced durable automation scheduler cursors, scheduled-run fire identity, `skip_missed` boot reconciliation, and separate delivery-error diagnostics across store/automation/API/CLI/web/docs surfaces.
- Task 05 has introduced MCP remote OAuth 2.1 PKCE auth config/status, durable token storage, redacted API/CLI/settings surfaces, and symlink escape hardening for skills and managed extension runtime dependencies.
- Task 06 has introduced shared `internal/toolruntime` process checkpoints, global `tool_processes` persistence, PID/start-time boot reconciliation, and scoped process interrupts across ACP agents/terminals, environment terminals, hooks, extensions, and shared subprocess helpers.
- Task 07 has introduced typed memory visibility through `GET /api/memory/health`, `GET /api/memory/history`, `agh memory health`, and `agh memory history`, backed by the durable memory catalog operation log.
- Task 08 has introduced local CLI config inspection/mutation (`agh config show/list/get/set/path/validate/check/edit`) plus managed-aware install/update/uninstall diagnostics without changing web settings/API contracts.
- Task 09 has introduced explicit workspace `.env` inspection/repair through `agh config validate/check --repair-env`, extension `requires_env` and `missing_env` diagnostics across CLI/API/settings/web contracts, and GoReleaser Homebrew cask plus nFPM `deb`/`rpm` package targets with checksum signing and SBOM coverage preserved.
- Task 10 has introduced QA planning artifacts under `.compozy/tasks/hermes/qa/`: feature test plan, regression suite, 15 manual P0/P1 test cases, and reserved `issues/`, `screenshots/`, and `logs/` evidence paths. Task 11 must keep `qa-output-path=.compozy/tasks/hermes`.
- Task 11 has executed the Hermes final QA pass across backend/runtime, CLI, API/SSE, web, and site docs; final evidence lives in `.compozy/tasks/hermes/qa/verification-report.md` with bug reports under `.compozy/tasks/hermes/qa/issues/`.

## Shared Decisions

- Future Hermes schema work should add ordered `store.Migration` entries near the owning store package schema list and let `RunMigrations` persist/check `schema_migrations` integrity.
- Future retry loops for transient Hermes work should prefer `internal/retry` policy/delay/wait helpers so cancellation and jitter behavior stay centralized.
- `observability.retention_days = 0` means keep observability history; negative values are invalid.
- Future lifecycle/memory health work should extend or consume `health.persistence` and `health.retention` rather than adding unrelated top-level health fields.
- Task 03 lifecycle diagnostics extend health as `health.failures` and downstream ACP availability as `health.agent_probes`; future health consumers should preserve those typed fields rather than flattening them.
- Future remote MCP auth work should use `internal/mcp/auth` and global DB token storage; never add access/refresh tokens, OAuth codes, PKCE verifiers, or client secrets to static config, logs, API payloads, CLI output, or generated settings fixtures.

## Shared Learnings

- `web/` and `packages/site` have no direct follow-up for Task 01 foundations because the migration table and retry helper are internal Go surfaces with no API/OpenAPI/settings/client payload changes.
- Task 02 retention is scoped to global observe rows only: `event_summaries`, `token_stats`, and `permission_log`; it does not delete session catalog rows or per-session event DBs.
- API health contract changes require `make codegen`, web generated type/fixture updates, and TypeScript SDK generated contract updates.
- Session lifecycle failures are persisted as typed `failure.kind` plus redacted `summary` and optional `crash_bundle_path`; CLI/API/SSE/web consumers should surface the stored object and avoid parsing stop detail text.
- Future automation consumers should treat scheduler cursor fields (`scheduler.next_run_at`, `last_scheduled_at`, `last_fire_id`, `catch_up_policy`, `misfire_count`) as schedule state, and `run.delivery_error` as delivery diagnostics separate from run execution `error`.
- Settings MCP server contracts now include `transport`, optional `url`, token-free `auth`, and redacted `auth_status`; web settings currently edits stdio servers only and treats remote rows as read-only config/CLI-auth managed entries.
- OAuth loopback login must bind only localhost or loopback IP redirect hosts; explicit `:0` redirect URLs should be replaced with the actual bound listener port before building the authorization URL.
- Skill and managed-extension path loading must canonicalize symlink targets and reject paths resolving outside the approved skill or extension source root.
- Tool process recovery must validate PID start-time evidence before signaling recovered processes; records without local PID evidence, including remote sandbox terminals, should be marked stale after restart rather than killed by PID.
- Memory operation history should remain bounded and pass summaries through `diagnostics.RedactAndBound` before API/CLI exposure. Runtime prompt assembly must remain independent from future context-ref/provider-hook interfaces until a later task explicitly wires that behavior.
- Task 06 did not change API/SSE/generated-client contracts, so no `web/` implementation follow-up was required. Site docs now cover operator-visible restart reconciliation and scoped interrupt behavior.
- Task 07 changed OpenAPI/web generated contracts and site docs, but did not surface a settings UI or runtime prompt behavior change. Future web consumers can use the generated `getMemoryHealth` and `listMemoryHistory` operations.
- Task 08 reused Task 05 config redaction boundaries for CLI config output: secret-bearing `env` maps are redacted, while token material remains outside static config.
- Task 09 extension environment diagnostics expose only variable names. API/web consumers should continue treating `requires_env` and `missing_env` as non-secret requirement identifiers and must not add environment values to payloads, logs, fixtures, or UI.
- Local GoReleaser OSS cannot validate this repository's Pro release config; Task 09 added a Go YAML integrity test for local checks, while CI's GoReleaser Pro dry-run remains the full package-build validation path.
- Hermes QA artifacts are planning-only until task 11. Task 11 owns live execution and must write its final evidence to `.compozy/tasks/hermes/qa/verification-report.md`.
- Remote MCP TOML overlays must support the same remote auth shape as the canonical config model: `transport`, `url`, and token-free `auth` fields.
- Fresh global DB schemas must include memory operation scope columns (`scope`, `workspace_root`, `filename`); migration v6 `add_memory_operation_scope` preserves that invariant.
- HTTP prompt handlers must drain terminal agent events after client disconnect before canceling the prompt context, matching UDS behavior and preserving durable terminal/error evidence.
- Web route animation keys should come from the reactive TanStack location, not mutable `router.latestLocation`, so local dialog/editor state is not discarded by delayed route-key catch-up.
- Reference managed-extension E2E installs should use packaged temp sources with materialized dependencies; source-root symlink escape rejection remains the correct production behavior.

## Open Risks

## Handoffs

- Task 03 completed in commit `b01f4963 feat: harden acp session lifecycle`; post-commit `make verify` passed.
- Task 04 completed in commit `e8a17a4b feat: add durable automation scheduler`; post-commit `make verify` passed.
- Task 05 completed in commit `156b8e8b feat: add mcp oauth auth security`; pre-commit `make verify` passed.
- Task 06 completed in commit `0f3e1893 feat: add tool process registry`; pre-commit `make verify` passed.
- Task 07 completed in commit `26f6ab1d feat: add memory visibility surfaces`; post-commit `make verify` passed.
- Task 08 completed in commit `c96077ff feat: add cli config lifecycle`; post-commit `make verify` passed.
- Task 09 completed in commit `a799dea3 feat: harden env extension release`; pre-commit `make verify` passed.
- Task 10 completed in commit `92adb526 test: add hermes hardening qa artifacts`; post-commit `make verify` passed.
