# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the production GitHub bridge provider under `extensions/bridges/github` on top of `internal/bridgesdk`.
- Cover GitHub webhook ingress for `issue_comment` and `pull_request_review_comment`, App vs PAT mode config/secret handling, outbound comment delivery, and provider-scoped multi-instance behavior.
- Finish with unit + integration coverage, task tracking updates, a clean `make verify`, and one local commit.

## Important Decisions
- Treat the approved task/spec/design docs as the required design artifact for this execution run instead of reopening a separate brainstorming approval loop.
- Model GitHub as a provider-scoped webhook runtime with repository-scoped bridge instances and a shared webhook path, then disambiguate owned instances using configured repository identity plus installation semantics.
- Keep App-mode delivery installation selection inside the provider by preferring explicit `installation_id` config and otherwise caching installation IDs from inbound webhook payloads or bridge metadata.

## Learnings
- Existing production providers share the same `internal/bridgesdk` runtime pattern: async ownership sync after initialize, per-instance config reconciliation, shared webhook guards, classified delivery failures, and subprocess-backed integration tests under `internal/extension/`.
- The Chat-SDK GitHub adapter reference uses HMAC-SHA256 webhook verification, `issue_comment` plus `pull_request_review_comment` events, review-thread rooting via `in_reply_to_id`, and repository-scoped installation caching for multi-tenant App mode.
- The GitHub provider package now clears the task-local coverage bar at `80.5%`, and the subprocess integration slice `go test -tags integration ./internal/extension -run GitHubProvider -count=1` passes with two owned instances sharing one `/github` endpoint.

## Files / Surfaces
- `extensions/bridges/github/*`
- `internal/extension/github_provider_integration_test.go`
- `.compozy/tasks/bridge-adapters/task_15.md`
- `.compozy/tasks/bridge-adapters/_tasks.md`
- `.compozy/tasks/bridge-adapters/memory/task_15.md`
- `.compozy/tasks/bridge-adapters/memory/MEMORY.md`

## Errors / Corrections
- `provider.serve` and `runServe` exit cleanly on EOF in unit tests; the assertions were corrected to expect success instead of an error.

## Ready for Next Run
- Current phase: complete.
- Production code is committed as `2442cf9` (`feat: add github bridge provider`), post-commit `make verify` is green, and the task plus shared workflow tracking files are updated but intentionally left unstaged.
