# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/acp` end-to-end: subprocess spawn, initialize/session lifecycle, prompt streaming, request handlers, permission sandboxing, crash reporting, and required test coverage.

## Important Decisions
- Use `github.com/coder/acp-go-sdk` for protocol types/constants and the low-level `Connection` transport instead of the generated `ClientSideConnection` so raw prompt usage and unstable `usage_update` payloads can still be decoded.
- Have `Driver.Start` own `initialize` plus `session/new` or `session/load` based on `StartOpts.ResumeSessionID`, returning the resolved ACP session id and captured modes/models on `AgentProcess`.
- Drain a brief prompt-stream quiescence window before sending the terminal `done` event because the SDK runs inbound notification handlers concurrently.

## Learnings
- `acp-go-sdk` v0.6.3 stable types do not include `PromptResponse.usage` or `usage_update`, so usage capture must be best-effort from raw JSON.
- The ACP helper subprocess pattern using the current `go test` binary is fast enough to cover start/prompt/crash/integration flows without a dedicated mock server binary in the repo.

## Files / Surfaces
- `internal/acp/client.go`
- `internal/acp/handlers.go`
- `internal/acp/permission.go`
- `internal/acp/types.go`
- `internal/acp/client_test.go`
- `internal/acp/handlers_test.go`
- `internal/acp/client_integration_test.go`
- `go.mod`
- `go.sum`

## Errors / Corrections
- Fixed a prompt-stream race by synchronizing event sends and prompt closure under the same lock.
- Moved the final `done` event to after the prompt drain window so it is consistently the last stream item.
- Added direct handler tests to push regular package coverage over the 80% requirement.

## Ready for Next Run
- Verified with `go test -race ./internal/acp`, `go test -race -tags integration ./internal/acp`, `go test -cover ./internal/acp` (80.1%), and `make verify`.
- Next task can consume `internal/acp` directly when defining `session.AgentDriver`, including `StartOpts.ResumeSessionID`, `AgentProcess.SessionID`, `AgentProcess.Caps`, and prompt-stream `AgentEvent` values.
