# TC-FUNC-020: provide_tools no longer advertised

**Priority:** P1
**Type:** Functional
**Package:** internal/extension
**Related Tasks:** 08

## Objective

Validate that after the tool resource migration, the extension manager no longer negotiates or advertises the `provide_tools` capability during extension handshake. Extensions must now publish tools as resource records through the resource runtime rather than through the legacy provide_tools capability. Any extension that still expects provide_tools should receive a capabilities response that omits it, forcing migration to the resource-based approach.

## Preconditions

- Extension manager is initialized with the resource runtime enabled.
- The tool resource migration (Task 08) is complete in the codebase.
- A test extension is available that previously relied on `provide_tools` in its capability negotiation.
- A second test extension is available that does NOT request `provide_tools`.

## Test Steps

1. Start the daemon with the resource runtime and extension manager active.
   **Expected:** Daemon starts without error. Extension manager initializes.

2. Inspect the extension manager's advertised server capabilities (the set of capabilities the daemon offers to extensions during handshake).
   **Expected:** The capabilities set does NOT include `provide_tools`. Capabilities like `provide_resources` or equivalent resource-based capabilities are present.

3. Connect the test extension that requests `provide_tools` during its initialize handshake.
   **Expected:** The extension connects successfully. The daemon's capabilities response omits `provide_tools`. The extension does not receive an error, but the capability is simply absent from the negotiated set.

4. Attempt to send a `tools/list` notification or `provide_tools` RPC from the extension.
   **Expected:** The daemon either ignores the message, returns a method-not-found error, or returns an empty acknowledgment. No tool records are created from this legacy path.

5. Connect the second test extension that does not request `provide_tools`.
   **Expected:** The extension connects and negotiates successfully. No regression for extensions that never used `provide_tools`.

6. Have the first extension publish a tool via the resource-based path (e.g., resource write for kind=tool).
   **Expected:** The tool resource record is created in the store. The tool appears in the projector's next Build cycle. This confirms the migration path works.

## Edge Cases

- Extension manifest that statically declares `provide_tools` as a required capability: daemon should either reject the extension with a clear error or degrade gracefully (verify which behavior is specified).
- Legacy configuration files that reference `provide_tools` in extension settings: loading config should emit a deprecation warning or ignore the setting without error.
- Extension that sends both `provide_tools` and resource-based tool writes: only the resource-based writes take effect.
- Grepping the codebase for `provide_tools` string: it should appear only in test code and migration documentation, not in production capability negotiation paths.
