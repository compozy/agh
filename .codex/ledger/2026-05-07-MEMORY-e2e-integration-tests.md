Goal (incl. success criteria):

- Fix and make all local e2e and integration tests pass properly.
- Success means `make test-integration` and `make test-e2e` pass in the current workspace, with any failures fixed at root cause and without weakening tests.
- Final completion also requires a prompt-to-artifact audit against the objective.

Constraints/Assumptions:

- Conversation in Brazilian Portuguese; code/artifacts in English.
- Prefix shell commands with `rtk`.
- Never run destructive git commands without explicit permission.
- Use `systematic-debugging`, `no-workarounds`, and `testing-anti-patterns`.
- Root-cause fixes only; no timing hacks, lint/test suppression, or mock-driven confidence.
- `make verify` is still the repo completion gate, but it does not prove e2e/integration targets by itself.

Key decisions:

- Treat prior ledger `2026-05-07-MEMORY-fix-issues.md` as historical evidence only because it targeted review issues, not this explicit e2e/integration goal.
- Re-run the actual `make test-integration` and `make test-e2e` targets before claiming completion.

State:

- Active.

Done:

- Loaded RTK instructions and required debugging/testing skills.
- Read prior fix ledger and `internal/CLAUDE.md`.
- Confirmed current `git status --short` only shows untracked ledger/task artifacts so far.
- Ran `make test-integration`; it failed after 10m with concrete failures:
  - `internal/daemon` build errors: `CreateBridgeRequest` still initialized with removed `Status`.
  - `internal/cli` bridge integration tests still pass removed `bridge create --status` flag.
  - `internal/cli` historical network channel integration tests lose historical channels/presence after restart and package times out.
  - `internal/extension` Teams/Telegram provider integration tests report degraded/not-ready bridge runtime.
  - `internal/network` capability fixture/router tests fail because capability `work_id` is now required.
- Fixed bridge contract test drift by removing removed `Status` request fields/CLI flag usage and expecting server-owned initial `starting` status.
- Fixed network capability fixtures/tests to include required `work_id` and directed thread target.
- Fixed historical network channel projections by persisting outbound `greet` timeline entries; added manager regression coverage and confirmed focused CLI historical tests pass.
- Fixed provider integration setup:
  - Telegram launch now binds `webhook_secret` when a webhook listener is enabled.
  - Teams manifest now forwards `AGH_BRIDGE_TEAMS_ALLOW_LOOPBACK_AUTH_FOR_TESTING` into the sanitized subprocess env, and focused Teams/Telegram provider tests pass.
- Fixed remaining daemon integration drift:
  - Updated daemon/network send tests and nightly API send request to include required `surface=thread` and `thread_id`.
  - Updated skills-only prompt assembler test workspace to include the `coder` agent scope required for agent-specific skills.
  - Rebuilt the harness-context prompt input augmenter with the same test workspace resolver used by the harness manager, so current-skills augmentation resolves the test agent.
  - Focused `internal/daemon` integration failures now pass.
- `make test-integration` now passes.
- First `make test-e2e` run passed Go e2e lanes but failed Playwright `web/e2e/network.spec.ts` due stale header expectations.
- Updated network e2e assertions to match the current header contract: channel title text excludes the `#` icon and channel meta exposes `2 agents` plus purpose; focused Playwright network test now passes.
- `make test-e2e` now passes.
- Web gates passed: `make web-lint`, `make web-typecheck`.
- Final repo gate passed: `make verify`.

Now:

- Complete prompt-to-artifact audit and final report.

Next:

- Mark goal complete after audit if no gaps remain.

Open questions (UNCONFIRMED if needed):

- Whether all e2e lanes are runnable locally without external provider credentials is UNCONFIRMED.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-07-MEMORY-e2e-integration-tests.md`
- Failure log: `~/Library/Application Support/rtk/tee/1778130298_make_test-integration.log`
- Focused passing commands:
  - `go test -race -tags integration ./internal/network -run 'TestProtocolFixturesRoundTripWithoutSemanticDrift|TestRoutersExchangeThreadCapabilityTransfers' -count=1`
  - `go test -race -tags integration ./internal/cli -run 'TestBridgeCreateAndGetIntegration|TestBridgeRoutesIntegration' -count=1`
  - `go test -race ./internal/network -run 'TestManagerPersistsConversationsBeforeRuntimeSideEffects/Should_persist_greet_presence_for_historical_channel_projections' -count=1`
  - `go test -race -tags integration ./internal/cli -run 'TestCLIHistoricalChannel(TaskNextAfterDaemonRestartIntegration|TaskRunStartAfterDaemonRestartIntegration|MixedOwnershipAfterDaemonRestartIntegration|TaskRunTerminalAfterDaemonRestartIntegration)' -count=1 -timeout=3m`
  - `go test -race -tags integration ./internal/extension -run 'TestTeamsProviderLaunchNegotiatesBridgeRuntime|TestTeamsProviderIngressAndDeliveryConformance|TestTelegramProviderLaunchNegotiatesBridgeRuntime' -count=1 -timeout=2m`
  - `go test -race -tags integration ./internal/daemon -run 'TestBootNetworkEnabledDeliversInboundAndShutsDownCleanly|TestBootNetworkShutdownTracksInterruptedInFlightDelivery|TestBootLoadsBundledSkillsIntoPromptAssemblerInSkillsOnlyMode|TestDaemonNightlyE2EAutomationTaskResumesIntoNetworkChannel|TestHarnessContextIntegrationStartupAndPromptShareResolverPolicy' -count=1 -timeout=5m`
  - `bun run --cwd web test:e2e:daemon-served:raw --grep "operator verifies thread and direct network surfaces"`
- Commands: `make test-integration`, `make test-e2e`
