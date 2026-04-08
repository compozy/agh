# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the marketplace interface/types and ClawHub HTTP client required by task 08.
- Keep scope limited to `internal/skills/marketplace` and `internal/skills/marketplace/clawhub` plus tests unless validation proves another file is required.

## Important Decisions
- Treat missing `SkillsConfig.Marketplace`/`MarketplaceConfig` from task 02 as an upstream gap, not scope for task 08, because the current deliverables are isolated to marketplace packages.
- Normalize ClawHub base URLs so callers can supply either a registry root or a full `/api/v1` base without duplicating API paths in task 10.
- Expose download payloads as `io.ReadCloser` on `SkillArchive.Data` so the install flow can stream archives and close the HTTP body explicitly.

## Learnings
- Current repo already carries marketplace-related runtime groundwork (`SourceMarketplace`, provenance helpers, MCP consent list), so task 08 can target isolated packages without touching existing registry logic.
- `httptest.NewServer` exposes escaped slash segments in `request.URL.Path` for slugs like `@agh/review`, so client tests need to assert `@agh%2Freview`.

## Files / Surfaces
- `.codex/ledger/2026-04-07-MEMORY-marketplace-client.md`
- `internal/config/config.go`
- `internal/skills/registry.go`
- `internal/skills/types.go`
- `.compozy/tasks/skills-v2/_techspec.md`
- `.compozy/tasks/skills-v2/adrs/adr-003.md`
- `internal/skills/marketplace/registry.go`
- `internal/skills/marketplace/types.go`
- `internal/skills/marketplace/clawhub/client.go`
- `internal/skills/marketplace/clawhub/client_test.go`

## Errors / Corrections
- Task references `internal/config/config.go` as if `MarketplaceConfig` from task 02 already exists, but current repository does not include those fields. No correction applied in code yet; treating as non-blocking for task 08.
- Initial client tests failed because override base URLs were not normalized to `/api/v1`; fixed by normalizing empty-path base URLs in `NewClient`.
- `make lint` flagged unchecked `Close()` calls on response bodies; fixed by explicitly reading/closing JSON responses and checking close errors in tests/helpers.

## Ready for Next Run
- Task 08 implementation and verification are complete. `go test ./internal/skills/marketplace/... -cover` passed at 80.0%, `make lint` passed, and `make verify` passed after the final code changes.
