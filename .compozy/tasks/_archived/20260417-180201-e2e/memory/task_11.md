# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the browser E2E automation operator flow on the shipped Automation page using the shared Playwright harness from task_08 and the runtime automation truth from task_04.
- Prove an operator can manage automation through the UI, cause a real execution path, and observe browser-visible run history plus linked session/transcript context.

## Important Decisions
- Reuse existing automation UI actions and route-level query flow instead of inventing test-only transport seams.
- Favor a job-based manual-trigger path because the shipped UI already exposes `Run now`; keep trigger coverage limited to the same management surface if it stays within task scope.
- Seed one deterministic automation job, one trigger, and one completed baseline run through public daemon APIs, then use that seeded run as the stable session-link/transcript proof inside the browser flow.
- Keep the browser-visible operator proof centered on editing a job and manually triggering a second run, while still asserting trigger visibility from the shared Automation surface.

## Learnings
- Current browser harness seeding only covers workspace/session/network flows; automation fixtures need their own deterministic seed path.
- Existing automation UI already exposes stable test IDs for route tabs, list items, detail panel, create dialog launch, run button, and form fields, but there is no shared Playwright selector map for the Automation page yet.
- The automation route already prepends a queued manual run into visible history client-side after `triggerAutomationJob`, so browser E2E can assert immediate run visibility while still relying on real backend mutation/invalidation.
- Browser artifact capture needs automation-specific route context and must count run cards separately from the run-history wrapper and session-link anchors to keep failed screenshots/debug state readable.
- The shipped run-history UI did not expose direct session navigation; adding an explicit `View Session` link keeps transcript assertions on a product surface instead of forcing the browser lane to reconstruct session URLs out-of-band.

## Files / Surfaces
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/fixtures/runtime-seed.test.ts`
- `web/e2e/fixtures/runtime.ts`
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/fixtures/selectors.test.ts`
- `web/e2e/fixtures/browser-artifact-session.ts`
- `web/e2e/fixtures/browser-artifact-session.test.ts`
- `web/e2e/fixtures/artifacts.ts`
- `web/e2e/automation.spec.ts`
- `web/src/routes/_app/-automation.integration.test.tsx`
- `web/src/systems/automation/components/automation-detail-panel.test.tsx`
- `web/src/systems/automation/components/automation-run-history.tsx`

## Errors / Corrections
- Corrected the Playwright flow to treat workspace onboarding as conditional; the shipped app can land directly on the main shell when a workspace is already selected.

## Ready for Next Run
- Task 11 is complete after focused helper/unit coverage, the dedicated Playwright automation scenario, `make web-test`, `make web-lint`, `make web-typecheck`, `cd web && bun run test:e2e -- e2e/automation.spec.ts`, and full `make verify`.
- Later browser/operator tasks can reuse the automation seed path, selector map, artifact capture fields, and run-history session-link surface added here.
