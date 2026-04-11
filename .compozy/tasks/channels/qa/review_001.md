# QA Review 001

Date: 2026-04-11
Scope: `.compozy/tasks/channels`
Mode: operator-style QA using real CLI/daemon flows plus integration verification

## Confirmed Issues

### 1. Enabled channel creation does not start or recover the adapter runtime

- Severity: high
- Scope: channels runtime / operator flow
- Reproduction:
  1. Build and install `sdk/examples/telegram-reference`.
  2. Start an isolated daemon with its own `AGH_HOME`.
  3. Run `agh channel create --scope global --platform telegram --extension telegram-reference --display-name 'QA Telegram' --include-peer`.
  4. Run `agh channel get <id>`, `agh extension status telegram-reference`, and `agh observe health`.
- Expected:
  - The enabled channel should trigger extension reload/start automatically.
  - The channel should transition out of `starting` once the adapter reports state.
  - The extension should recover from the initial "no enabled channel instance" condition on its own.
- Actual:
  - The channel remains stuck in `starting`.
  - The extension remains `state=error`, `health=unhealthy`, with:
    - `extension: resolve channel runtime for "telegram-reference": daemon: no enabled channel instance configured for extension "telegram-reference"`
  - A manual `agh channel restart <id>` immediately recovers the flow and moves the channel to `auth_required`.
- Notes:
  - This is a real user-facing lifecycle bug.
  - Root-cause hypothesis from code inspection: create persists the channel instance but does not execute the daemon runtime reload/start path used by `enable` / `restart`.

### 2. Installing a local extension directory fails when common package-manager symlinks exist

- Severity: medium
- Scope: extension install path discovered during wide QA run
- Reproduction:
  1. Use a fresh `AGH_HOME`.
  2. Run `agh extension install sdk/examples/prompt-enhancer`.
- Expected:
  - Local extension installation should succeed for a standard checked-out example directory, or at least ignore transient developer-managed payloads like `node_modules/.bin/*` symlinks.
- Actual:
  - Install fails with:
    - `extension: symlinks are not allowed in extension payload "/.../sdk/examples/prompt-enhancer/node_modules/.bin/tsc"`
- Notes:
  - This is also reproducible via `go test -tags integration ./internal/extension -run TestReferenceExtensionsEndToEnd`.
  - This issue is broader than channels, but it surfaced during the required wide QA pass.

### 3. Fast prompt output can race delivery registration and drop the final projected sequence

- Severity: medium
- Scope: channel delivery projection / channel ingest lifecycle
- Reproduction:
  1. Run `go test -tags integration ./internal/extension -run TestChannelDeliveryIntegrationSlowAdapterCoalescesIntermediateDeltas -count=50`.
- Expected:
  - Even when adapter delivery is slow and deltas are coalesced, the final delivery event should still represent the latest projected state and keep `seq=5`.
- Actual:
  - The test flakes with:
    - `last delivery seq = 4, want 5`
- Notes:
  - Root-cause hypothesis from debugging: `channels/messages/ingest` starts the prompt before registering the prompt delivery, then only seeds from a partial event snapshot. Fast prompt events can land between the initial seed read and the later `RegisterPromptDelivery`, causing one projected event to be lost before the broker is attached.
