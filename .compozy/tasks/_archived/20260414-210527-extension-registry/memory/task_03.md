# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_03 by adding ClawHub and GitHub `RegistrySource` adapters under `internal/registry/`, plus unit tests for the required edge cases.
- Finish only after targeted adapter verification and repository `make verify` both pass.

## Important Decisions
- Use the approved task spec + techspec + ADRs as the execution design baseline for this run.
- Rely on the existing installer manifest-root traversal for GitHub source-archive fallback instead of adding adapter-specific extraction behavior.
- Keep the legacy marketplace package untouched for now except where refactoring is necessary to share behavior safely; full migration/removal stays in later tasks.
- Keep the new ClawHub adapter behavior aligned with the existing marketplace client semantics, including retry/backoff behavior and the current `/skills` search path contract.
- Use `GITHUB_TOKEN` from environment in the GitHub adapter, with release-metadata API calls resolving assets and the installer performing the final extraction safety checks.

## Learnings
- `internal/registry/installer.go` already walks into a single extracted top-level directory until it finds `SKILL.md` or `extension.toml`, which covers GitHub's `<repo>-<tag>/` source archive layout.
- The current ClawHub client already has the retry/backoff and error-shaping logic this task needs; the missing piece is adapting it to `registry.RegistrySource` and `DownloadOpts`.
- Package coverage reached the task bar after adding targeted negative-path tests: `internal/registry/clawhub` at 82.5% and `internal/registry/github` at 81.0%.
- Full repository verification passed via `make verify` after fixing explicit `response.Body.Close()` handling required by `errcheck`, including a final post-commit rerun on committed HEAD.

## Files / Surfaces
- `internal/registry/` shared source/types/installer surfaces
- `internal/skills/marketplace/clawhub/client.go`
- `internal/skills/marketplace/clawhub/client_test.go`
- `internal/registry/clawhub/client.go`
- `internal/registry/clawhub/client_test.go`
- `internal/registry/github/client.go`
- `internal/registry/github/client_test.go`

## Errors / Corrections
- Fixed a test-harness issue where `t.Setenv` was combined with `t.Parallel()` in the GitHub token test.
- Adjusted one GitHub download assertion to use the actual streamed response content length instead of stale asset metadata.
- Fixed unchecked response-body close paths in the GitHub adapter and server test helper after `make verify` surfaced `errcheck` failures.

## Ready for Next Run
- Task 03 is complete. Code landed in local commit `c170cc1` (`feat: add registry source adapters`). Tracking and memory files remain intentionally unstaged.
