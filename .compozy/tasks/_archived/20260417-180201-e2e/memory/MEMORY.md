# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 01 is complete: `internal/testutil/e2e/` now provides the shared runtime harness and artifact plumbing that later daemon and browser E2E tasks should build on.
- Task 02 is complete: `internal/testutil/acpmock/` now provides fixture-backed named ACP mock agents, launch-compatible temporary agent definitions, and diagnostics that plug into the shared runtime harness.
- Task 03 is complete: `internal/daemon` now has composition-root runtime network collaboration scenarios for direct reply lifecycle plus whois/recipe exchange, backed by the shared harness and fixture-driven mock agents.
- Task 04 is complete: `internal/daemon` now has composition-root runtime automation coverage for webhook-created system sessions and task-backed automation delegation, backed by shared automation/task harness helpers and fixture-driven mock agents.
- Task 05 is complete: `internal/daemon` now has composition-root runtime bridge ingress coverage for route creation, session reuse, delivery progression, secret bindings, and real extension subprocess Host API behavior through `telegram-reference`.
- Task 06 is complete: `internal/daemon` now has local-provider runtime E2E coverage for allowed tool execution, blocked sandbox operations, persisted sandbox metadata, and explicit tool-host diagnostics, backed by the shared harness.
- Task 07 is complete: HTTP and UDS transport parity coverage now reuses the shared runtime harness for HTTP approval flow, HTTP webhook ingress, UDS/CLI projection parity, and the documented UDS approval `501 Not Implemented` gap.
- Task 08 is complete: `web/e2e/` now provides the shared Playwright browser harness, daemon launch/attach runtime helpers, stable browser artifact capture, and smoke coverage against the daemon-served onboarding shell.
- Task 11 is complete: the browser lane now covers Automation-page operator flow with seeded jobs/triggers, real manual job execution, visible run history, and linked session/transcript navigation on the shipped UI.
- Task 12 is complete: the browser lane now covers Bridges-page operator flow with seeded real `telegram-reference` runtime state, edit/enable/test-delivery actions, live health visibility, and inbound route creation through the shipped UI.
- Task 13 is complete: explicit repo-local E2E lanes now exist across `Makefile`, Mage, and package scripts for `runtime`, `web`, combined PR-required coverage, and nightly credentialed coverage without falling back to the broad `test-integration` sweep.
- Task 14 is complete: the nightly lane now adds combined runtime flows for automation task resume plus bridge ingress/environment delivery, a browser-observed `@nightly` bridge-to-session operator flow on the shared Playwright harness, and richer multi-domain artifact capture without changing the default PR-required lanes.

## Shared Decisions

- The shared E2E runtime harness boots AGH as an external `agh daemon start --foreground` subprocess instead of importing `internal/daemon`, so HTTP/UDS integration suites can reuse it without creating package cycles.
- Runtime assertions and diagnostics should stay on public product surfaces: HTTP, UDS, CLI, persisted artifacts, and daemon-owned files rather than in-process hooks into daemon internals.
- Artifact capture is manifest-driven: only captured surfaces are written into the manifest, using stable relative paths under a per-run artifact root.
- Fixture-backed mock agents must be registered before the runtime harness boots the daemon, via `RuntimeHarnessOptions.MockAgents`, so the real agent catalog sees the generated `AGENT.md` files during startup.
- Network collaboration scenarios should drive real `agh network ...` sends from the runtime lane and use mock agents as deterministic recipients, instead of re-encoding RFC semantics in browser automation or daemon-internal hooks.
- Automation and task runtime scenarios should reuse the shared subprocess harness helpers for fixture seeding, webhook/manual trigger delivery, automation run listing, task/task-run reads, and artifact capture rather than creating daemon-only setup seams.
- Bridge runtime scenarios should reuse the shared UDS helper layer in `internal/testutil/e2e` for extension install/enable and bridge lifecycle/secrets/routes reads, while keeping final pass/fail assertions in `internal/daemon`.
- Browser runtime scenarios should reuse `web/e2e/fixtures/test.ts` and its `createBrowserRuntime(...)` helpers instead of starting Vite preview servers or re-implementing daemon boot logic inside each browser spec.
- Browser scenarios that need daemon-operator-only endpoints should go through the shared launch-mode `requestOperatorJSON(...)` UDS helper instead of assuming the daemon HTTP surface exposes those routes.
- When runtime tests need a real `telegram-reference` binary, they should build from the repo root and write the output into a per-test temp extension copy rather than the shared `sdk/examples/telegram-reference/bin/` path; package-level integration suites run in parallel.
- The lane matrix is now centralized in `internal/e2elane`, so future tasks should update one shared runtime/web/nightly plan instead of duplicating selectors across Mage targets, Makefile targets, and package scripts.
- Later-tier daemon combined-flow coverage should keep the `TestDaemonNightlyE2E...` prefix so `internal/e2elane.NightlyRuntimeE2EPattern` can select those scenarios centrally without leaking them into the default runtime lane.
- Multi-domain nightly diagnostics should capture `combined_flow.json` and `tool_host_diagnostics.json` separately from provider-call artifacts, so one failed run retains both the cross-domain identifiers and the concrete tool-host outcomes.

## Shared Learnings

- A subprocess-backed harness is the cycle-safe seam for cross-package runtime E2E coverage in this repository because `internal/daemon` already composes `internal/api/httpapi` and `internal/api/udsapi`.
- Gosec needs explicit path-containment validation for artifact writes in the shared collector even when artifact paths come from a fixed internal contract.
- The ACP mock layer stays maintainable when it only owns deterministic ACP session behavior and fixture diagnostics, leaving daemon boot, config validation, permissions, and public-surface assertions to the real runtime.
- Network enablement has to be persisted into the seeded config file, not only toggled in-memory, or the real daemon boots with the embedded network runtime disabled.
- The shared harness now installs a real `agh` shim into the seeded home `PATH`, which lets daemon-owned runtime code and mock-agent `sandbox_exec` commands reach the product CLI deterministically.
- Stable network audit assertions are easier to maintain when artifact capture decodes the JSONL audit log into ordered JSON arrays before writing snapshots.
- Automation webhook fixtures must be signed with a live current timestamp during test execution because the daemon validates request freshness.
- Integration-inclusive coverage is the meaningful threshold for these runtime-heavy packages; task 04 leaves `internal/daemon` at 80.0% and `internal/testutil/e2e` at 80.1% with the shipped runtime lane enabled.
- The shared E2E helper layer now includes bridge/operator helpers plus bridge-specific artifact capture for routes, delivery state, and secret bindings, which later transport-parity and browser bridge tasks should reuse instead of open-coding UDS calls.
- Bridge secret bindings are exposed on the daemon operator surface under `/api/bridges/:id/secret-bindings`, not a `/secrets` alias.
- Runtime tasks can seed named sandbox profiles plus `defaults.sandbox` through `internal/testutil/e2e.ConfigSeedOptions`, which keeps local-provider coverage on the same subprocess harness instead of adding an environment-specific boot path.
- Session environment diagnostics are most useful when captured from both surfaces at once: the public session payload and the persisted `session.json` metadata under the seeded session home. This keeps environment assertions readable after stop or failure.
- Real prompted user sessions do not stop automatically after an ACP turn; runtime E2E that needs stable stopped-state assertions should issue an explicit public-surface stop and then wait for `store.SessionMetaFile` plus `/api/sessions/:id` to converge.
- Tool-host diagnostics for sandbox scenarios should be recorded as explicit allowed/blocked operation outcomes in the shared artifact model, paired with host side effects or their absence, rather than relying on transcript text alone.
- Task 06 keeps the runtime-heavy coverage bars green with integration-inclusive package coverage: `internal/testutil/e2e` reached 80.7% and `internal/daemon` remained at 80.0%.
- Helpers that are imported by `internal/api/httpapi` or `internal/api/udsapi` integration suites must not depend on `internal/cli`, because `internal/cli` already depends on `internal/daemon`, which imports those transport packages and creates an import cycle in tests. Use the shared harness's shell-backed `CLI.RunJSON(...)` surface when transport-parity checks need real CLI reads.
- Browser E2E must build the `web` bundle before compiling `cmd/agh`, otherwise the daemon binary serves stale embedded assets and the browser lane is no longer testing the shipped surface.
- Web Vitest `setupFiles` execute for node-environment unit tests too, so browser-global shims in `web/src/test-setup.ts` need runtime guards instead of assuming jsdom.
- Browser-visible session streaming that goes through the AI SDK must emit `finishReason` on HTTP `finish` chunks; sending `stopReason` causes client-side validation failures and breaks the shipped session UI even when the daemon turn completed correctly.
- Mock ACP agents used in browser/runtime resume flows must advertise `loadSession` consistently with their implemented `session/load` behavior; if the initialize capabilities say resume is unsupported, the daemon's public `/api/sessions/:id/resume` path fails with HTTP 500 even when the mock driver can actually load the session.
- The daemon-served embedded web bundle must import Tailwind (`@import "tailwindcss";`) and register shared UI sources explicitly (currently `@source "../../packages/ui/src/**/*.{ts,tsx}"`) or browser E2E will exercise an unstyled shipped surface instead of the real operator UI.
- Browser network operator checks should treat the channel timeline as `say`-message history only; `direct` and `trace` collaboration progress is surfaced on status and peer metrics, and later browser tasks should assert those product surfaces instead of expecting every message kind in the timeline.
- Browser automation operator checks can seed deterministic jobs, triggers, and one completed baseline run through the public automation APIs, then use the shipped `Run now` action for the browser-driven execution and a run-history session link for transcript assertions without recreating task_04 runtime semantics in Playwright.
- Browser bridge operator checks should seed workspace-scoped bridges, not global bridges, when the flow needs inbound route creation; the daemon bridge-ingress session path requires `bridge.WorkspaceID`.
- The default browser lane is now explicitly daemon-served via `web/package.json` (`test:e2e:daemon-served`), excludes `@nightly` specs by default, and the nightly web selector uses `--pass-with-no-tests` so credentialed/nightly expansion can land before nightly browser specs exist.
- Bridge health SSE snapshots can advance `route_count` and `last_success_at` before the cached routes query refreshes; the shipped Bridges UI needs route-query invalidation when the live `route_count` diverges from cached routes or ingress-driven browser flows will show updated health with an empty routes panel.

## Open Risks

- Credentialed Daytona and other live-provider nightly scenarios still depend on external secrets/providers in the execution sandbox; default PR-required lanes must remain secret-free.

## Handoffs

- Reuse `internal/testutil/e2e` for new runtime/browser lanes instead of adding more package-local `newIntegrationRuntime` boot logic.
- Reuse `internal/extensiontest` marker helpers and conformance checks when asserting provider-side bridge effects, but keep daemon-owned route/session/delivery truth in the composition-root tests.
