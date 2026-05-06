# BUG-002: Bridge Ingest Lost Network Prompt Semantics

**Severity:** High  
**Priority:** P1  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** daemon bridge ingest E2E path
- **Live provider/LLM:** acpmock-backed daemon E2E; external bridge provider not required

## Summary

The Telegram bridge ingress E2E intermittently failed on the second inbound message with a JSON-RPC internal error because the Host API session adapter hid network-prompt capabilities from the extension host.

## Behavioral Impact

- **Operator/User Goal:** Bridge messages cannot reliably reuse an existing routed AGH session.
- **Agent Behavior:** The bridge extension sees a generic prompt failure instead of a network-aware in-flight prompt state.
- **Business Outcome:** External-channel collaboration appears unreliable under realistic repeated ingress.
- **Cross-Surface State:** Daemon route reuse and extension Host API behavior diverged.

## Reproduction

```bash
go test -race -parallel=4 -count=1 -tags integration \
  -run 'TestDaemonE2EBridgeIngressCreatesAndReusesRouteThroughTelegramExtension' ./internal/daemon
```

Observed before the fix:

- The second bridge ingest failed through the Host API with a JSON-RPC internal error.

## Expected

Bridge ingress must use the network prompt path and expose prompt-busy state to the Host API adapter so repeated inbound messages route deterministically.

## Root Cause

`hostAPISessionManagerAdapter` only exposed the base `session.Manager` methods. It dropped optional bridge/network prompt methods (`PromptNetwork` and `IsPrompting`), so extension Host API code used the normal prompt path and could not coordinate around network prompt activity.

## Fix

`internal/daemon/daemon.go` now returns a network-capable Host API adapter when the source session manager supports it. `internal/daemon/daemon_test.go` asserts `PromptNetwork` and `IsPrompting` are exposed and that the adapter does not fall back to normal `Prompt`.

## Verification

- `go test ./internal/daemon -run TestNewHostAPISessionManagerAdapter -count=1`
- `go test ./internal/testutil/acpmock ./internal/testutil/acpmock/cmd/acpmock-driver -count=1`
- `go test -race -parallel=4 -count=1 -tags integration -run 'TestDaemonE2EBridgeIngressCreatesAndReusesRouteThroughTelegramExtension' ./internal/daemon`
- `make test-e2e-runtime`

## Impact

- **Users Affected:** Operators using bridge-backed collaboration sessions.
- **Frequency:** Reproducible in repeated bridge ingress flows.
- **Workaround:** None.

## Related

- Test Case: TC-SCEN-001

