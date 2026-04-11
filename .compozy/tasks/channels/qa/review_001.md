# QA Review 001

Date: 2026-04-11
Scope: `.compozy/tasks/channels`
Method: real daemon + CLI flows, adapter runtime markers, targeted integration stress

## Confirmed Issues

### 1. Installing or enabling a channel adapter while the daemon is running leaves the extension in an error state before any channel exists

- Severity: high
- Status: fixed in this run
- Surfaces: extension install/enable flow, channel adapter lifecycle
- Reproduction:
  1. Start an isolated daemon with `AGH_HOME=/tmp/...`.
  2. Run `agh extension install ./sdk/examples/telegram-reference`.
  3. Inspect `agh extension status telegram-reference`.
- Expected:
  - Installing or enabling a channel adapter before any channel instance exists should succeed without surfacing a hard runtime error.
  - The extension may stay idle/registered until a channel is created, but it should not be marked unhealthy.
- Actual:
  - Install/enable tries to initialize the adapter immediately.
  - The extension transitions to `state=error`, `health=unhealthy`, with:
    - `extension: resolve channel runtime for "telegram-reference": daemon: no enabled channel instance configured for extension "telegram-reference"`
  - If the install is executed through the daemon-backed path, the CLI command itself fails.
- Notes:
  - This is a real operator-facing bug in the happy path for installing channel adapters.
  - Creating the channel afterward does recover the extension, so the broken behavior is specifically the premature launch attempt.

### 2. Fast prompt output can race prompt-delivery registration and drop the final projected sequence

- Severity: medium
- Status: fixed in this run
- Surfaces: channel delivery projection, prompt-to-broker handoff
- Reproduction:
  1. Run `go test -tags integration ./internal/extension -run TestChannelDeliveryIntegrationSlowAdapterCoalescesIntermediateDeltas -count=50`.
- Expected:
  - Even with a slow adapter and broker coalescing, the last delivery event should still be the terminal state with `seq=5`.
- Actual:
  - The test flakes with:
    - `last delivery seq = 4, want 5`
  - In the broader integration suite this also shows up as the ordered-delivery test ending on `delta` instead of `final`.
- Notes:
  - Root-cause hypothesis from code inspection: `channels/messages/ingest` starts the prompt, waits to collect seed events, and only then registers the delivery. Very fast ACP events can be emitted before registration and only partially persisted by the time replay runs, so one terminal event is lost.

## Verified Non-Issues In This Run

- Creating an enabled channel after the adapter is already installed now recovers correctly:
  - `agh channel create ... --extension telegram-reference`
  - channel transitions from `starting` to `auth_required`
  - `agh extension status telegram-reference` becomes `active` / `healthy`
- After the fixes above:
  - `agh extension install ./sdk/examples/telegram-reference` with the daemon already running now succeeds and leaves the extension `registered` instead of `error`
  - targeted integration stress passed:
    - `go test -tags integration ./internal/extension -run 'TestChannelDeliveryIntegration(SlowAdapterCoalescesIntermediateDeltas|PromptProducesOrderedDeliveryStream)' -count=50`
    - `go test -tags integration ./internal/daemon -run TestCreateEnabledChannelAfterBootReloadsErroredExtension -count=20`
- The previous local install failure for `sdk/examples/prompt-enhancer` caused by `node_modules/.bin/*` symlinks did not reproduce in this run.
