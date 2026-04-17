# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add the missing browser E2E proof for the shipped Network page: create a channel via UI, inspect peers, verify visible message/timeline state, and prove reload/navigation continuity.
- Keep runtime RFC/correlation truth in task_03; this task only covers operator-visible browser behavior and the browser/runtime harness plumbing it needs.

## Important Decisions

- Reuse the shared Playwright runtime harness from task_08 instead of adding a route-specific daemon boot path.
- Seed deterministic network collaboration state through public daemon/runtime surfaces so browser assertions observe real runtime-backed data instead of mocked browser-only protocol truth.
- Keep browser-visible message-history assertions on the persisted `say` channel timeline and read `direct`/`trace` progress through peer and status metrics, matching the runtime truth established in task_03.
- Fix shipped-surface regressions uncovered by the scenario instead of weakening the browser flow: restore embedded Tailwind utility output in the daemon-served bundle and refetch network queries when operators return to the Channels tab.

## Learnings

- Baseline before implementation: `web/e2e/network.spec.ts` does not exist, `web/e2e/fixtures/selectors.ts` only exposes session lifecycle helpers, `web/e2e/fixtures/runtime-seed.ts` only seeds workspace/session state, and browser route-state capture is tailored to the session page.
- The shipped Network route already exposes most stable test IDs for tabs, create-channel dialog controls, channel items, peer items, detail panels, and message cards; only metric-card selectors had to be added.
- The runtime network timeline persists `say` messages for the channel view, while `direct` and `trace` outcomes are exposed through runtime status and peer metrics.
- The daemon-served embedded web bundle needs `@import "tailwindcss";` plus an explicit `@source "../../packages/ui/src/**/*.{ts,tsx}"` entry in `web/src/styles.css`; without them, browser E2E sees an unstyled shipped UI and the dialog becomes unusable.
- Returning from the Peers tab to the Channels tab was serving stale cached channel history/status after runtime-side activity until `use-network-page` explicitly refetched the relevant queries on tab return.

## Files / Surfaces

- `web/e2e/fixtures/runtime.ts`
- `web/e2e/fixtures/runtime-helpers.ts`
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/fixtures/runtime-seed.test.ts`
- `web/e2e/fixtures/runtime.test.ts`
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/fixtures/selectors.test.ts`
- `web/e2e/fixtures/browser-artifact-session.ts`
- `web/e2e/fixtures/browser-artifact-session.test.ts`
- `web/e2e/fixtures/artifacts.ts`
- `web/e2e/fixtures/artifacts.test.ts`
- `web/e2e/network.spec.ts`
- `web/src/routes/_app/network.tsx`
- `web/src/routes/_app/-network.test.tsx`
- `web/src/hooks/routes/use-network-page.ts`
- `web/src/systems/network/components/network-peer-detail-panel.tsx`
- `web/src/styles.css`
- `web/src/styles.test.ts`

## Errors / Corrections

- Corrected a shipped CSS packaging gap in `web/src/styles.css` that left the daemon-served Network dialog unstyled and non-interactive in browser E2E.
- Corrected stale-query behavior in `use-network-page` so channel details and metrics are refreshed when operators switch back to the Channels tab after peer inspection.

## Ready for Next Run

- Task 10 is complete after clean focused browser/unit coverage plus repository verification (`make web-lint`, `make web-typecheck`, `make verify`).
- Later browser/operator tasks can reuse the network-enabled harness seeding, route-state capture, and metric-card selectors added here.
