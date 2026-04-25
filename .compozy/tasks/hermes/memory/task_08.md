# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 08 CLI Config and Setup Lifecycle: `agh config` inspection/mutation/validation with redacted output, shell completion, managed-aware install/update/uninstall behavior, tests, docs/web impact assessment, clean verification, tracking updates, and one local commit.

## Important Decisions

- Task 08 must satisfy both the task-local command set (`get`, `set`, `list`, `path`, `validate`) and the broader TechSpec naming (`show`, `edit`, `check`) where repository scope allows; aliases or thin commands are preferred over divergent behavior.
- Config output and diagnostics must reuse the Task 05 redaction boundary: never expose MCP tokens, OAuth codes, PKCE verifiers, client secrets, or auth-sensitive environment values in CLI/setup output.
- `agh completion` is already provided by Cobra default completion generation; Task 08 adds focused coverage instead of replacing it with a duplicate custom command.
- `agh update` is diagnostic/idempotent in this task: it defers to `AGH_MANAGED` package managers or points unmanaged installs at manual release/source update paths without mutating binary files.
- `agh uninstall` removes runtime artifacts and preserves AGH home by default; destructive home removal requires explicit `--purge --force`.
- No `web/` API/settings contract change is required because Task 08 adds local CLI commands and docs without changing settings payloads, generated OpenAPI, or typed clients.

## Learnings

- Shared memory confirms Task 05 already added remote MCP auth config/status, durable token storage, redacted settings/API/CLI surfaces, and generated web contract updates; Task 08 can consume those types instead of inventing new redaction rules.
- The Hermes CLI/setup analysis maps Task 08 to issues 36, 37, 39, 40, 42, and 57; issue 43 `.env` repair and issue 59 extension env handshakes are broader Track 6 follow-ups unless current code inventory shows they are directly required for install/config behavior.
- Baseline signals: `go run ./cmd/agh config --help`, `go run ./cmd/agh update --help`, and `go run ./cmd/agh uninstall --help` all fail as unknown commands; `go run ./cmd/agh completion bash` already emits Cobra bash completion output.
- Current CLI has `internal/cli/install.go` for interactive bootstrap only; no config/update/uninstall command files exist yet.
- Existing persistence API has `ResolveConfigWriteTarget` and `EditConfigOverlay`, which should be used for CLI config mutation instead of rewriting TOML manually.
- Added `WriteTarget.Path()` to expose resolved write targets without hardcoding config paths outside `internal/config`.
- Affected package tests `go test ./internal/cli ./internal/config` pass after adding config/lifecycle coverage.
- Fresh `make verify` passed after lint cleanup: web format/type/test/build completed, Go lint reported `0 issues.`, Go tests reported `DONE 5839 tests`, and package boundaries passed.
- Post-commit `make verify` also passed on commit `c96077ff`: Go lint reported `0 issues.`, Go tests reported `DONE 5839 tests in 7.427s`, and package boundaries passed.
- A separate exploratory `bun --cwd packages/site test` failed in existing dirty landing-page tests expecting old bento/extensibility copy; the failure is outside Task 08 CLI docs/config changes and was not folded into this task.

## Files / Surfaces

- Planned inventory surfaces: `internal/cli`, `internal/config`, `internal/api/contract/settings.go`, `internal/api/core/settings.go`, `web/src/routes/_app/settings/`, `packages/site/`, install/update scripts or release metadata if present.
- Implemented surfaces: `internal/cli/config.go`, `internal/cli/lifecycle.go`, `internal/cli/root.go`, `internal/cli/install.go`, `internal/config/persistence.go`, focused CLI tests, generated CLI reference docs for config/update/uninstall, and setup/configuration site docs.
- Assessed but not changed: `web/src/routes/_app/settings/`, `internal/api/contract/settings.go`, and `internal/api/core/settings.go`.

## Errors / Corrections

- First full `make verify` runs exposed Go lint issues in the new config CLI code (`errcheck`, `funlen`, `gocritic`, `gocyclo`, `gosec`, then `unparam`); fixed by splitting reflection helpers, moving the mutation path table to package scope, passing config by pointer, using duration integer conversion, adding an explicit editor-command security justification, and simplifying provider path classification.

## Ready for Next Run

- Task 08 implementation, verification, tracking, scoped local commit, and post-commit verification are complete.
- Local commit: `c96077ff feat: add cli config lifecycle`.
