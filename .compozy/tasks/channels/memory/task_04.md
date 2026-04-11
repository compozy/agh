# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Extended the extension protocol, Go contract, and TypeScript SDK so channel-capable extensions explicitly negotiate `channels/deliver` and receive only instance-scoped channel launch data.

## Important Decisions

- `channel.adapter` is the canonical `provides` value for channel-capable extensions, and `channels/deliver` is the only capability-specific extension service required by this task.
- Channel launch data is injected through `subprocess.InitializeRuntime.Channel` via a manager-side `ChannelRuntimeResolver`, not ambient environment variables.
- Bound secret exposure stays limited to the selected channel instance; no arbitrary vault or secret Host API surface was introduced.
- Channel Host API methods added for later tasks are `channels/messages/ingest`, `channels/instances/get`, and `channels/instances/report_state`, with `channel.read` and `channel.write` enforcement.

## Learnings

- JSON initialize markers must be parsed as newline-delimited records; `strings.Fields` corrupts recorded payloads when JSON strings contain spaces.
- Test helpers that emulate extension initialize responses should derive capability service methods from the requested `provides` set to stay aligned with protocol validation.

## Files / Surfaces

- `internal/extension/protocol/host_api.go`
- `internal/subprocess/handshake.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/contract/sdk.go`
- `internal/extension/capability.go`
- `internal/extension/manager.go`
- `internal/extension/manager_test.go`
- `internal/extension/manager_integration_test.go`
- `internal/daemon/daemon_test.go`
- `sdk/typescript/src/{extension.ts,extension.test.ts,host-api.ts,host-api.test.ts,index.ts,generated/contracts.ts}`
- `.compozy/tasks/_archived/20260411-014454-ext-architecture/_protocol.md`

## Errors / Corrections

- Fixed integration marker parsing after the first run truncated recorded initialize payloads at spaces inside JSON string values.
- Fixed the daemon extension helper initialize response so it advertises capability service methods required by stricter initialize validation, which unblocked `make verify`.

## Ready for Next Run

- Task 05 can build on the negotiated channel method identifiers and `runtime.channel` payload; the delivery/ingest behavior itself is still intentionally unimplemented.
