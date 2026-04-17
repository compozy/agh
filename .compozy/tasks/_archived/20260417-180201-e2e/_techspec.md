# TechSpec: Agentic System End-to-End Validation

## Executive Summary

AGH already has substantial subsystem coverage for sessions, ACP, network routing, automation dispatch, task orchestration, bridge delivery, extension subprocesses, and environment providers. What it still lacks is a single end-to-end strategy that proves the shipped product behaves correctly when those parts are composed into real agentic flows. That is the actual goal of this spec.

This techspec defines E2E as two coordinated layers:

1. a deterministic daemon/runtime lane that boots the real daemon, real SQLite stores, real embedded network runtime, real extension subprocesses, and deterministic ACP mock agents
2. a browser lane that drives the daemon-served web app with Playwright for the complete operator workflows the product actually ships today

The runtime lane is the source of truth for protocol correctness, cross-domain orchestration, and persisted artifacts. The browser lane proves that operators can perform meaningful end-to-end work through the embedded SPA, not just reach a page or click through a smoke path. The primary trade-off is additional harness and CI cost in exchange for real confidence that network, channels, tasks, automations, bridges, extensions, environments, and their combinations actually work together.

## System Architecture

### Component Overview

The E2E architecture is intentionally layered instead of trying to force one harness to prove everything:

- `internal/testutil/acpmock/`
  - shared helper for deterministic Go ACP subprocess agents
  - writes temporary agent definitions into the isolated AGH home
  - keeps the mock boundary narrow: exact fixture matching, deterministic streaming, and fault injection only
- `internal/testutil/e2e/`
  - shared daemon/runtime fixture utilities
  - isolated `AGH_HOME`, isolated workspace, seeded config, artifact collection, and runtime boot helpers
- `internal/daemon/*_integration_test.go`
  - cross-system runtime E2E scenarios
  - composition-root location for flows that cut across network, automation, tasks, bridges, extensions, and environments
- `internal/api/httpapi/*_integration_test.go`
  - transport-specific proofs for HTTP-sensitive or transport-asymmetric surfaces such as approval and webhook ingress
- `internal/api/udsapi/*_integration_test.go`
  - transport-specific proofs for UDS operator flows and parity checks
- `internal/extensiontest/`
  - existing bridge adapter harness and provider conformance utilities reused as building blocks for bridge and extension E2E
- `web/e2e/`
  - Playwright browser suite for the daemon-served SPA
  - validates real UI journeys, not mocked route-hook behavior

Two execution lanes are defined:

- **Runtime E2E**
  - boots one real daemon runtime
  - enables network, automation, tasks, bridges, extensions, and environments as required by the scenario
  - registers deterministic ACP mock agents as normal AGH agents
  - drives flows through public APIs, CLI paths, bridge ingress, extension subprocesses, and environment tool hosts
- **Browser E2E**
  - hits the daemon-served UI, not just a Vite proxy
  - drives complete operator journeys through the shipped SPA
  - asserts browser-visible state while the runtime lane remains authoritative for low-level protocol details

### Runtime Data Flow

For a representative system scenario:

1. The harness creates an isolated `AGH_HOME`, workspace, config, and artifact directory.
2. The daemon boots with the required runtime surfaces enabled:
   - embedded network
   - automation manager
   - task runtime
   - bridge manager
   - extension manager
   - environment registry
3. The harness registers one or more deterministic ACP mock agents through temporary agent definitions.
4. The scenario stimulates the system through a real ingress path:
   - session prompt
   - network send
   - automation trigger or webhook
   - bridge ingress
   - extension Host API call
5. The daemon executes the resulting orchestration using the normal runtime paths.
6. Assertions read back persisted and projected state from the public surfaces that correspond to the domain under test.

### Browser Data Flow

For a representative browser scenario:

1. The harness builds the web bundle and starts the daemon serving embedded assets.
2. The daemon is seeded with the runtime state needed for the scenario.
3. Playwright opens the SPA over HTTP.
4. The test performs the user journey through the browser.
5. Assertions verify UI-visible outcomes and, when needed, a second read path from the daemon.

## Implementation Design

### Core Interfaces

The shared runtime harness should stay small and scenario-driven:

```go
package e2e

type RuntimeHarness struct {
    HomePaths    aghconfig.HomePaths
    WorkspaceID  string
    HTTPBaseURL  string
    UDSClient    cli.DaemonClient
    ArtifactsDir string
}

type MockAgentSpec struct {
    FixturePath     string
    FixtureAgent    string
    AgentName       string
    DiagnosticsPath string
}

func StartRuntimeHarness(t *testing.T, opts RuntimeHarnessOptions) *RuntimeHarness
func (h *RuntimeHarness) RegisterMockAgent(t *testing.T, spec MockAgentSpec)
```

`MockAgentSpec` is the required narrow waist for runtime and browser E2E mock-agent wiring. Tests should register fixture-backed agents through `internal/testutil/e2e` instead of calling `acpmock.Register` directly.

The harness is responsible for:

- isolated daemon boot
- isolated database and workspace state
- temp agent definition rendering
- artifact capture
- public-surface clients for HTTP, UDS, and CLI execution

It is not a second runtime framework. Domain logic remains in the daemon and product APIs.

### Data Models

The deterministic ACP fixture model expands from single-session transcript flows to multi-agent system flows. The fixture engine still owns assistant streaming determinism, but it must support scenario primitives for:

- multiple named agents
- channel membership
- inbound network prompt turns
- tool calls and permission requests
- real `agh network send` and other shell/tool-host operations when the scenario requires agent-driven collaboration
- bridge response content
- environment command execution expectations

Fixture schema v2 is the contract for these flows:

- exact matching by `turn_source`, `user_text`, and structured network metadata instead of rendered prompt substring heuristics
- diagnostics that persist both the received prompt metadata and the selected matcher for failed-run forensics
- a constrained `driver_control` step for pathological ACP behaviors such as disconnect, raw invalid JSON-RPC frames, and cancel-blocked sessions

Each E2E scenario must emit an artifact manifest that captures the assertion surfaces required by that domain:

- `transcript.json`
- `events.json`
- `network_messages.json`
- `network_audit.json`
- `automation_runs.json`
- `tasks.json`
- `task_runs.json`
- `bridge_health.json`
- `provider_calls.json`
- `session_environment.json`
- browser trace, screenshots, and console logs for Playwright scenarios

Not every scenario uses every artifact, but the manifest format is stable so failed runs are diagnosable.

### Public Surfaces Exercised

The E2E suites must treat these as first-class product surfaces:

- Sessions
  - `POST /api/sessions`
  - `POST /api/sessions/{id}/prompt`
  - `GET /api/sessions/{id}/transcript`
  - `GET /api/sessions/{id}/events`
  - `POST /api/sessions/{id}/approve`
- Network
  - `GET /api/network/status`
  - `GET /api/network/peers`
  - `GET /api/network/channels`
  - `GET /api/network/channels/{channel}`
  - `GET /api/network/channels/{channel}/messages`
  - `POST /api/network/channels`
  - `POST /api/network/send`
  - `GET /api/network/inbox`
- Automation
  - jobs, triggers, runs, and webhook ingress
- Tasks
  - tasks, task-runs, claim/start/attach/complete/fail/cancel lifecycle
- Bridges
  - provider discovery
  - bridge instance CRUD
  - secret bindings
  - test delivery
  - health stream
- Extensions
  - bridge-related Host API paths
  - automation/task-related Host API paths
  - environment Host API paths that are exercised through a real subprocess
- Web UI
  - workspace onboarding
  - session chat and approval
  - network inspection
  - automation management
  - bridge management
  - task workflows are explicitly excluded from browser E2E until a real web task surface exists
  - skills and knowledge remain out of scope for this E2E initiative because they do not prove the agentic runtime goals targeted by this spec

## Integration Points

- `internal/testutil/acpmock/cmd/acpmock-driver`
  - canonical deterministic ACP mock driver shared by runtime and browser lanes
  - prebuilt once per lane and overridable via `AGH_TEST_ACPMOCK_DRIVER_BIN`
- `cmd/agh`
  - daemon binary reused by runtime and browser lanes
  - prebuilt once per lane and overridable via `AGH_TEST_DAEMON_BIN`
- Embedded NATS network runtime
  - required for true channel and peer collaboration scenarios
- Existing bridge adapter harness in `internal/extensiontest/`
  - reused for provider and extension conformance building blocks
- Environment providers
  - local provider in PR-required suites
  - Daytona provider in nightly or credentialed suites
- Playwright
  - required for browser E2E
  - runs against daemon-served UI
- Vitest
  - remains for unit, component, adapter, and `jsdom` integration tests
  - does not carry the E2E claim

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/testutil/acpmock/` | modified | Current mock ACP helper is too session-centric for system E2E. Moderate test-only risk. | Extend fixtures for multi-agent, network-aware, tool-aware, and environment-aware flows. |
| `internal/testutil/e2e/` | new | Shared runtime/browser fixture utilities. Low production risk because it is test-only. | Add isolated daemon boot, config seeding, artifact capture, and CLI/HTTP/UDS helpers. |
| `internal/daemon/*_integration_test.go` | modified | Cross-system runtime E2E belongs at the composition root. Low production risk. | Add multi-domain runtime scenarios here. |
| `internal/api/httpapi/*_integration_test.go` | modified | HTTP remains important for approval, webhooks, and transport parity. Low risk. | Keep transport-specific scenarios here. |
| `internal/api/udsapi/*_integration_test.go` | modified | UDS remains important for operator flows and CLI parity. Low risk. | Keep UDS-specific scenarios here. |
| `internal/extensiontest/` | modified | Existing harness should be elevated into the formal E2E strategy. Low risk. | Reuse and extend harness for bridge/extension E2E building blocks. |
| `web/e2e/` | new | Browser E2E currently does not exist. Moderate risk because Playwright adds CI weight. | Add Playwright config, helpers, and focused operator journeys. |
| `web/package.json` | modified | Browser E2E needs scripts and Playwright dependencies. Low risk. | Add Playwright scripts alongside existing Vitest scripts. |
| `Makefile` / `magefile.go` | modified | Need explicit E2E entry points. Low risk. | Add runtime, web, combined, and nightly E2E targets. |
| CI jobs | modified | Broader E2E requires tiered execution to protect dev loop. Medium risk if poorly scoped. | Add PR-required runtime and browser lanes plus nightly credentialed lane. |

## Testing Approach

### PR-Required Runtime E2E

These are the minimum daemon/runtime proofs for the shipped agentic system:

1. `TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents`
   - Create a channel with two agents.
   - Send `say` or `direct`.
   - Deliver a real network-origin prompt turn to the receiving agent.
   - Have the receiving agent reply through the real `agh network send` path.
   - Assert RFC-visible correlation across `message_id`, `interaction_id`, `reply_to`, `receipt`, and `trace`.
   - Assert transcript, network message log, network audit log, API projection, and CLI visibility.

2. `TestDaemonE2ENetworkWhoisAndRecipeExchange`
   - Exercise `whois` discovery and `recipe` exchange through the live network runtime.
   - Assert peer visibility, route behavior, and persisted channel history.

3. `TestDaemonE2EAutomationPromptTriggerCreatesCompletedSystemSession`
   - Trigger a real automation path:
     - webhook
     - observer event such as `session.stopped`
     - or manual trigger when appropriate
   - Assert rendered prompt, system session creation, transcript persistence, run status, and stop semantics.

4. `TestDaemonE2EAutomationTaskBackedJobDelegatesTaskRun`
   - Create a task-backed automation job.
   - Trigger it through a real daemon ingress path.
   - Assert `RunDelegated`, `task_id`, `task_run_id`, task-run lifecycle, and session linkage where applicable.

5. `TestDaemonE2EBridgeIngressAgentReplyAndBridgeDelivery`
   - Ingest a bridge event through the real bridge runtime.
   - Assert route creation or reuse, session reuse or creation, agent processing, delivery broker progression, and provider-call side effects.

6. `TestDaemonE2EEnvironmentToolExecutionHonorsSandbox`
   - Run a session in the configured environment.
   - Assert allowed tool execution, blocked operation behavior, and persisted environment metadata.

7. `TestDaemonE2EExtensionHostAPIEnvironmentAndAutomationFlow`
   - Use a real extension subprocess.
   - Prove environment Host API access and one automation or task flow initiated through the extension boundary.

### PR-Required Browser E2E

The browser lane is not just smoke coverage. It should exercise complete operator workflows on the web surfaces that already exist in the product:

1. `session-chat-lifecycle`
   - resolve or select a workspace
   - create a session from the sidebar
   - send a prompt
   - observe streaming in the chat view
   - handle approval when requested
   - stop and resume the session
   - reload the page and verify transcript hydration plus session state continuity

2. `network-channel-creation-and-collaboration-observability`
   - open the Network page
   - create a channel through the UI
   - verify peers materialize
   - verify channel messages and timeline reflect a real runtime collaboration scenario already driven by the daemon lane
   - verify page state survives navigation or reload

3. `automation-job-and-trigger-operator-flow`
   - create or edit a job or trigger in the UI
   - manually trigger a job when that action exists in the UI
   - for trigger-driven flows, stimulate the real ingress outside the browser when needed
   - verify run history, detail panel state, and resulting linked session or transcript surfaces when applicable

4. `bridges-operator-flow`
   - create a bridge
   - manage secret bindings or configuration required by the UI flow
   - observe health stream updates through the real SSE path
   - run test delivery or equivalent bridge action exposed in the UI
   - verify the resulting bridge state and downstream visible effects

Browser E2E should validate what the operator can meaningfully accomplish through the web app today. It should not be reduced to “open route and assert title,” and it should not try to carry low-level protocol truth that belongs in the daemon lane.

### Daemon-Only E2E In Current Product

Some important E2E surfaces remain daemon-only for now because the product does not expose a corresponding web workflow yet or because the proof is inherently runtime-level:

- tasks and task-run lifecycle
- low-level RFC v0 network semantics such as lifecycle correlation, duplicate handling, and queue draining
- extension subprocess behavior and Host API internals
- environment tool-host behavior and sandbox restrictions

These flows must still be covered end to end, but today they belong in the runtime lane rather than browser automation.

### Nightly Or Credentialed E2E

These suites are intentionally excluded from the normal PR loop:

- Daytona-backed environment scenarios
- provider-specific external integrations that need real credentials
- broader restart and recovery matrices
- combined flows that cross bridge, automation, task, and environment boundaries with external dependencies

### Combined-Flow Follow-Up

After the base suites are stable, add combined scenarios such as:

- automation -> agent session -> network reply
- bridge ingress -> agent -> environment tool -> bridge delivery
- automation task-backed delegation -> task run -> resumed session in channel
- web-created automation or bridge configuration -> real runtime execution -> web-observed outcome

These flows are important, but they should be built on top of the base harness instead of replacing it.

## Development Sequencing

### Build Order

1. Create `internal/testutil/e2e/` for isolated daemon boot, public clients, and artifact capture. No dependencies.
2. Extend `internal/testutil/acpmock/` for multi-agent, tool-aware, network-aware fixtures. Depends on step 1.
3. Add network-enabled daemon runtime helpers and the first collaboration scenarios under `internal/daemon`. Depends on steps 1 and 2.
4. Add automation prompt and task-backed runtime scenarios. Depends on step 3.
5. Reuse and extend `internal/extensiontest/` for bridge and extension runtime scenarios. Depends on steps 1 through 4.
6. Add environment and sandbox runtime scenarios, including a nightly Daytona lane. Depends on steps 1 through 5.
7. Add focused transport-specific HTTP and UDS parity scenarios where the composition-root suite is not enough. Depends on steps 3 through 6.
8. Add Playwright under `web/e2e/` and run it against the daemon-served UI. Depends on steps 1 through 7.
9. Add `make test-e2e-runtime`, `make test-e2e-web`, `make test-e2e`, and `make test-e2e-nightly`, plus matching Mage targets and CI jobs. Depends on steps 3 through 8.
10. Add combined-flow scenarios once the base runtime and browser suites are stable. Depends on steps 4 through 9.

### Technical Dependencies

- Node 20 is required for the daemon-served web bundle and Playwright only.
- Runtime E2E builds Go binaries only and may reuse prebuilt `agh` / `acpmock-driver` binaries through environment overrides.
- The web bundle must be built before daemon-served browser E2E runs.
- Embedded network must be enabled in runtime fixtures for collaboration scenarios.
- UDS already exposes `/api/sessions/{id}/approve`, but it currently returns `501 Not Implemented`; approval-sensitive coverage stays in the HTTP and browser lanes until transport parity exists.
- Local environment flows are PR-required; Daytona flows are nightly or credentialed.
- No current lane exercises real LLM providers; any credentialed provider coverage is a separate follow-up lane rather than an implicit promise of the deterministic suite.

## Monitoring and Observability

Every failed E2E run must leave behind enough state to explain the breakage without rerunning immediately:

- transcript payloads
- raw session events when relevant
- network message and audit snapshots
- automation run snapshots
- task and task-run snapshots
- bridge health snapshots
- mocked provider API logs or marker files
- session environment metadata
- browser traces, screenshots, console errors, and network logs for Playwright

Artifact capture is mandatory, not best-effort.

## Technical Considerations

### Key Decisions

- Decision: treat E2E as a system-level concern, not just deterministic ACP integration.
  - Rationale: the product value is in composed agentic behavior across runtime domains.
  - Trade-offs: the scope is broader and CI is heavier.
  - Alternatives rejected: continuing to call a daemon-only ACP slice “E2E”.

- Decision: split the strategy into runtime E2E and browser E2E lanes.
  - Rationale: daemon correctness and browser correctness are both needed, but they are not the same proof.
  - Trade-offs: two harnesses and two CI lanes.
  - Alternatives rejected: daemon-only testing, or pushing protocol assertions into Playwright.

- Decision: make browser E2E cover complete operator journeys on existing web surfaces, not just smoke paths.
  - Rationale: if the web app is part of the shipped product, browser E2E should prove operators can complete real work through it.
  - Trade-offs: Playwright scenarios become a bit heavier and need seeded runtime state.
  - Alternatives rejected: smoke-only browser checks, or pretending non-existent web workflows such as tasks already exist.

- Decision: keep the ACP mock narrow and implement it in Go.
  - Rationale: the shipped suite already shares the ACP SDK with production code, avoids cross-language parity drift, and keeps fixture ownership inside the existing Go test toolchain.
  - Trade-offs: the deterministic driver still needs explicit fault-injection primitives for protocol-path coverage.
  - Alternatives rejected: reviving a separate Node mock driver or treating the mock as a top-level system simulator.

- Decision: route fixture turns by structured prompt metadata and exact user text.
  - Rationale: rendered prompt substrings are not a stable contract once network/system prompts are composed by the runtime.
  - Trade-offs: fixture v2 is intentionally breaking and requires in-repo migration.
  - Alternatives rejected: keeping `contains` / `equals` as a legacy fallback inside the active suite.

- Decision: place cross-system runtime E2E at the composition root under `internal/daemon`.
  - Rationale: daemon boot is where network, automation, tasks, bridges, extensions, and environments are actually wired together.
  - Trade-offs: some scenarios still need transport-specific follow-up coverage elsewhere.
  - Alternatives rejected: scattering every cross-domain flow across package-local tests or inventing a separate top-level Go test tree.

- Decision: use domain-specific assertion surfaces instead of transcript-only goldens.
  - Rationale: network, tasks, automations, bridges, extensions, and environments each have different truth surfaces.
  - Trade-offs: more artifacts and more assertion helpers.
  - Alternatives rejected: transcript-only goldens for the whole system, or raw event dumps as the only source of truth.

- Decision: scope PR-required E2E to shipped, externally reachable surfaces.
  - Rationale: the suite must not claim runtime flows that are only partially wired.
  - Trade-offs: some implemented-but-unwired seams remain out of P0.
  - Alternatives rejected: promising network-to-task handoff E2E before that path is exposed through a real product ingress.

- Decision: make no real-provider claim for the current E2E matrix.
  - Rationale: deterministic ACP fixtures prove runtime integration behavior, not provider quality or provider-specific schema correctness.
  - Trade-offs: product-level provider confidence must come from a separate, credentialed lane if the team wants it.
  - Alternatives rejected: implying that deterministic mock coverage is equivalent to running canonical flows against Claude, OpenAI, or Gemini.

### Known Risks

- Risk: the new runtime harness becomes a second framework.
  - Likelihood: medium.
  - Mitigation: keep helpers focused on daemon boot, fixture seeding, and public-surface reads.

- Risk: browser E2E duplicates daemon assertions and becomes noisy.
  - Likelihood: medium.
  - Mitigation: keep browser assertions strictly UI-visible and leave protocol truth to runtime suites.

- Risk: some apparently implemented surfaces are not yet externally reachable.
  - Likelihood: high.
  - Mitigation: mark those as future work in the spec and do not represent them as shipped E2E scope.

- Risk: credentialed providers and remote environments add CI instability.
  - Likelihood: medium.
  - Mitigation: keep them in nightly or explicitly credentialed lanes.

## Architecture Decision Records

- [ADR-001: Mock ACP Through a Temporary Agent Definition](adrs/adr-001.md) - Historical proposal for a temp-agent mock strategy; superseded by ADR-006.
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) - Treat daemon/runtime proof and browser/operator proof as coordinated but distinct test layers.
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) - Put network, tasks, automation, bridges, extensions, and environments together under `internal/daemon`.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) - Use transcripts, network logs, run records, bridge health, environment metadata, and UI outcomes instead of transcript-only goldens.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) - Cover externally reachable flows in PRs and move heavier credentialed or future surfaces into later tiers.
- [ADR-006: Keep ACP Mock Implemented in Go](adrs/adr-006.md) - Standardize the shipped Go mock driver and the shared-binary runtime/browser harness contract.
- [ADR-007: No Current E2E Lane Uses Real LLM Providers](adrs/adr-007.md) - Document the present confidence boundary and keep provider coverage as a separate future tier.
