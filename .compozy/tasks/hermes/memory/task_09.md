# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 09 Environment, Extension, and Release Hardening: safe `.env` sanitization/repair, extension `requires_env` manifest support with missing-env diagnostics, GoReleaser package targets with signing/checksum/SBOM guarantees preserved, tests/dry-run evidence, web/site impact assessment, clean verification, tracking updates, and one local commit.

## Important Decisions

- `.env` writes must be explicit, so repair will be exposed through config validation/check flow instead of automatic mutation during normal config loads.
- Extension env requirement status will expose only variable names in `requires_env` and `missing_env`; secret values must never be surfaced in CLI/API/web output.
- Registry schema changes are avoidable for Task 09 because installed extension status can derive requirement metadata from the manifest path already persisted in registry rows.
- Homebrew packaging uses GoReleaser `homebrew_casks` because current GoReleaser v2 docs mark casks as the active Homebrew path; Linux packages use nFPM `deb` and `rpm` targets.

## Learnings

- Baseline audit: `.env` loading currently uses `godotenv.Read` directly; malformed multi-key lines are not repaired and secret-like values are not sanitized before lookup.
- Baseline audit: extension manifests currently have no `requires_env` field and extension payloads/settings rows have no missing-env diagnostics.
- Baseline audit: `.goreleaser.yml` currently builds archives with checksum signing and SBOM, but has no Homebrew or nFPM package targets.
- Task 08 completed local CLI config lifecycle in commit `c96077ff` and deliberately left `.env` repair and extension env handshakes to Task 09.
- Focused backend validation after implementing `.env`, extension, API/settings, and release-config tests passed for `go test ./internal/config ./internal/extension ./internal/cli ./internal/api/contract ./internal/api/core ./internal/daemon ./internal/settings`.
- Release hardening now preserves checksum signing and archive/source SBOMs, adds package SBOMs, and mechanically asserts Homebrew cask plus `deb`/`rpm` targets in `internal/config/release_config_test.go`.
- Full `make verify` passed after lint corrections: web lint/typecheck/test/build, Go fmt/lint (`0 issues.`), race-enabled Go unit tests (`DONE 5851 tests in 55.156s`), Go build, and package boundaries.
- Fresh pre-commit `make verify` passed after staging the scoped Task 09 files: web lint/typecheck/test/build, Go fmt/lint (`0 issues.`), race-enabled Go unit tests (`DONE 5851 tests in 6.118s`), Go build, and package boundaries.
- Local `go run github.com/goreleaser/goreleaser/v2@v2.15.3 check` is not usable for this repository because GoReleaser OSS rejects the existing Pro config; release config integrity is covered locally by the Go YAML test and by CI's GoReleaser Pro dry-run.

## Files / Surfaces

- Touched: `internal/config`, `internal/cli/config.go`, `internal/extension`, `internal/cli/extension*.go`, `internal/api/contract`, daemon/settings conversions, `web/src/systems/settings`, `web/src/routes/_app/settings/hooks-extensions.tsx`, `.goreleaser.yml`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, release config tests, and clean `packages/site` docs.

## Errors / Corrections

- First focused Go test run failed because the new manifest test used `strings` without importing it; fixed the import and reran the changed backend package set successfully.
- First full `make verify` run failed in Go lint on `sprintfQuotedString` and `hugeParam`; fixed `.env` formatting to use `%q` and changed manifest conversion to a pointer receiver, then reran the full gate successfully.

## Ready for Next Run

- Task tracking updated: task_09 status/checklists completed, master task row completed, and task_10 QA plan now calls out `.env`, extension env, web/API, docs, and release package checks.
- Local commit created: `a799dea3 feat: harden env extension release`.
