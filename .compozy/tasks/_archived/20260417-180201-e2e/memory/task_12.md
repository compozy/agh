# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Add the browser-lane bridge operator proof on the shipped Bridges route.
- Cover bridge creation or editing, required secret-binding/configuration, real SSE health visibility, and test delivery with browser-visible downstream state change.
- Keep runtime semantics sourced from task_05 and public daemon APIs; do not simulate provider-side delivery inside the browser.

## Important Decisions

- Reuse the shared Playwright harness and extend the existing browser fixture helpers instead of adding a bridge-only runtime launcher.
- Keep assertions on browser-visible bridge route surfaces and use runtime reads only for deterministic seeding or diagnostic confirmation.
- Use an edit-based browser bridge flow seeded through public extension/bridge APIs: one real disabled Telegram bridge is prepared before navigation, then the browser covers edit, secret binding, enablement, health-stream visibility, and test delivery.
- Register the bridge ingress fixture as the bootstrap `general` agent so real inbound bridge routing can reuse the default daemon agent without adding extra browser config wiring.
- Patch the copied `telegram-reference` extension manifest with concrete marker paths and compute the real install checksum in the browser helper so the Playwright lane can drive task_05 runtime truth directly.
- Install the real bridge extension through a launch-mode UDS operator request because `/api/extensions` is not exposed on the browser HTTP surface.
- Seed the bridge as workspace-scoped and resolve the active launch-mode workspace from `runtime.paths.homeDir`, because inbound route creation needs `bridge.WorkspaceID` to create the daemon session.

## Learnings

- The current shared browser helpers cover session, network, and automation only; there is no bridge-specific seeding, selector mapping, artifact state capture, or Playwright scenario yet.
- The current worktree already has unrelated `.compozy/tasks/e2e` tracking edits from earlier tasks; task 12 should avoid disturbing them until its own completion updates are ready.
- The browser bridge seed must match task_05 runtime routing semantics: Telegram inbound updates supply `peer_id` plus `thread_id`, but not `group_id`, so the seeded bridge routing policy cannot require `include_group`.
- A global bridge cannot satisfy the real inbound route-claim path in browser E2E because `createBridgeSession` requires `bridge.WorkspaceID`; workspace scope is part of the product contract here, not a harness workaround.
- Extracting `runtime.resolveWorkspace` from the browser runtime object loses the method `this` binding; the helper must call the bound method to reach `requestJSON`.

## Files / Surfaces

- `web/e2e/fixtures/runtime.ts`
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/fixtures/browser-artifact-session.ts`
- `web/e2e/bridges.spec.ts`
- `web/src/routes/_app/bridges.tsx`
- `web/src/hooks/routes/use-bridges-page.ts`
- `web/src/systems/bridges/components/bridge-create-dialog.tsx`
- `web/src/systems/bridges/components/bridge-detail-panel.tsx`
- `web/src/systems/bridges/components/bridge-test-delivery-dialog.tsx`
- `web/src/systems/bridges/hooks/use-bridge-health-stream.ts`

## Errors / Corrections

- Pre-change gap confirmed: `web/e2e/bridges.spec.ts` does not exist, and the shared browser helpers currently have no bridge coverage.
- The initial browser bridge seed incorrectly required `group_id` in the routing policy. Real Telegram runtime ingress then failed with host-api `Invalid params` because the normalized inbound envelope does not carry `group_id`.
- The first workspace-scoped seed attempt still failed until the helper called the bound `runtime.resolveWorkspace(...)` method; unbound invocation crashed while trying to read `requestJSON` from `undefined`.

## Ready for Next Run

- Task complete. Fresh verification evidence: focused web unit slice (`22` tests), `web/e2e/bridges.spec.ts`, and full `make verify` all passed on April 17, 2026.
