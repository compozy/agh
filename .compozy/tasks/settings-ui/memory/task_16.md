# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add durable daemon-served Playwright coverage for the critical settings operator flows under `web/e2e/`.
- Execute the task_15 QA matrix against the shipped settings surface, capture evidence under `.compozy/tasks/settings-ui/qa/`, and fix any root-cause regressions.
- Finish with fresh passing `make test-e2e-web` and `make verify`.

## Important Decisions

- Reuse the existing `web/e2e/fixtures/test.ts` runtime harness instead of introducing a settings-only browser runner.
- Seed deterministic settings prerequisites through public daemon APIs in `web/e2e/fixtures/runtime-seed.ts`, not through ad hoc file mutations inside specs.
- Keep shared additions tight: only stable settings selectors plus reusable seed/cleanup helpers needed by the committed settings specs.
- Treat the missing `scripts/discover-project-contract.py` path from the task text as a repo mismatch and document it in QA evidence while using the actual Make/Mage verification contract.
- Keep non-loopback ADR-004 validation in the normal Playwright lane by extending the shared runtime with a configurable bind host instead of building a second transport harness.

## Learnings

- The repo’s broad gate is `make verify`, but browser settings coverage must also be exercised explicitly through `make test-e2e-web`; `Verify()` in `magefile.go` does not include Playwright.
- The settings surface already exposes stable test ids for shell navigation, restart banners, providers, MCP scope switching, skills, and hooks/extensions actions.
- Restart polling only survives a real page refresh if the active operation id is persisted; the fix stores the minimal restart state in `sessionStorage` and rehydrates it on reload.
- Settings transport parity must be derived from the daemon bind host; a zero-value parity surface breaks hooks/extensions affordances even on loopback binds.
- Nested TOML overlay writes for skills required fixing the shared persistence renderer rather than weakening the browser scenario.

## Files / Surfaces

- `.compozy/tasks/settings-ui/qa/issues/BUG-001-restart-refresh-continuity.md`
- `.compozy/tasks/settings-ui/qa/issues/BUG-002-skills-overlay-persistence.md`
- `.compozy/tasks/settings-ui/qa/issues/BUG-003-settings-transport-parity.md`
- `.compozy/tasks/settings-ui/qa/verification-report.md`
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/fixtures/selectors.test.ts`
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/fixtures/runtime-seed.test.ts`
- `web/e2e/fixtures/runtime.ts`
- `web/e2e/settings-transport.spec.ts`
- `web/e2e/settings.spec.ts`
- `.compozy/tasks/settings-ui/qa/screenshots/`
- `internal/config/persistence.go`
- `internal/config/persistence_test.go`
- `internal/daemon/settings.go`
- `internal/daemon/settings_test.go`
- `web/src/systems/settings/stores/use-settings-restart-store.ts`
- `web/src/systems/settings/hooks/use-settings-restart.test.tsx`
- `web/src/test-setup.ts`

## Errors / Corrections

- `python3 scripts/discover-project-contract.py --root .` failed because `scripts/discover-project-contract.py` is not present in this checkout.
- QA exposed three blocking regressions that were fixed at the source:
  - restart polling lost continuity on full refresh during daemon relaunch
  - nested skills overlay writes produced invalid TOML
  - settings transport parity returned a zero-value status

## Ready for Next Run

- Fresh settings QA evidence lives under `.compozy/tasks/settings-ui/qa/`; if a follow-up run changes behavior, rerun `make test-e2e-web` and `make verify`, then refresh the screenshots and verification report instead of creating a parallel artifact path.
