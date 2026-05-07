# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 04 live provider discovery sources in `internal/modelcatalog`.
- Success requires side-effect-free provider_live rows/status for configured/core providers, fail-closed status for unsafe providers without configured discovery, timeout/redaction/env-home policy coverage, same-provider refresh coalescing, no ACP session calls, focused tests, >=80% provider source coverage, `go test ./internal/modelcatalog/...`, and full `make verify`.

## Important Decisions
- Current worktree already contains uncommitted Task 03 `internal/modelcatalog` service/source code. Treat it as branch state and build on it without reverting.
- Same-provider refresh work is coalesced in `modelcatalog.Service` before source execution so concurrent refreshes cannot double-touch provider HOME/subprocess work for the same provider.
- Live provider adapters emit `provider_live` rows/status only; discovery never uses ACP session creation/load/mutation paths.
- Self-review correction: same-provider refreshes with different `SourceID` scopes are serialized but not coalesced into the wrong status result; only identical refresh scopes share in-flight status/error results.

## Learnings
- Pre-change signal: `rtk go test ./internal/modelcatalog ./internal/modelcatalog/... -count=1 -cover` passes with 32 tests, but live provider discovery adapters for Task 04 are absent.
- Implemented live adapters for Codex/OpenAI, Claude/Anthropic, Gemini, OpenRouter, Vercel AI Gateway, Ollama, OpenCode, and configured/fail-closed OpenClaw/Hermes/Pi.
- Focused verification passed:
  - `rtk go test ./internal/modelcatalog -count=1`
  - `rtk proxy go test ./internal/modelcatalog/... -count=1 -cover` with 83.4% coverage after the self-review coalescing correction.
  - `rtk go test ./internal/modelcatalog -run 'TestLiveProvider|TestLiveDiscovery|TestLiveProviderParsing' -count=1 -race`
  - `rtk go test ./internal/modelcatalog/... -count=1 -race`
  - `rtk go test ./internal/store/globaldb -run 'TestGlobalDBModelCatalog|TestOpenGlobalDBFailsOnSchemaMigrationIntegrityMismatch' -count=1`
  - `rtk make fmt`
  - `rtk make lint`
- Full verification passed three times with `rtk make verify`: once before the self-review correction, once after it, and once after commit `ca4f350e`.

## Files / Surfaces
- Changed: `internal/modelcatalog/service.go`
- Added: `internal/modelcatalog/live_sources.go`
- Added: `internal/modelcatalog/live_sources_test.go`
- Referenced but not changed: `internal/config/provider.go`, `internal/providerenv`, `internal/session/provider_runtime.go`.

## Errors / Corrections
- `scripts/check-test-conventions.py` is absent in this checkout, so the explicit skill script check could not be run.
- Initial in-flight refresh coalescing shared results by provider only; self-review found it could return the wrong source status for concurrent same-provider refreshes with different `SourceID`s. Fixed by adding a refresh scope key and a regression test.

## Ready for Next Run
- Task 04 implementation, tests, tracking updates, local commit `ca4f350e feat: add live provider discovery sources`, and post-commit `rtk make verify` are complete.
- Tracking/memory artifacts remain untracked by design; code commit staged only Go implementation surfaces.
