# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is complete: `internal/frontmatter` now owns shared frontmatter splitting and decode flow for `config`, `memory`, and `skills`.
- Task 02 is complete: `internal/api/contract` now owns the shared daemon DTO contract consumed by `apicore`, `httpapi`, and `udsapi`.
- Task 04 is complete: shared server-side API code now lives in `internal/api/core`, and the old `internal/apicore` plus `internal/apisupport` package boundaries are retired.
- Task 05 is complete: HTTP and UDS transports now live in `internal/api/httpapi` and `internal/api/udsapi`, and shared API test helpers now live in `internal/api/testutil`.
- Task 06 is complete: per-session SQLite ownership now lives in `internal/store/sessiondb`, global registry/workspace/observe persistence now lives in `internal/store/globaldb`, and `internal/store` is reduced to shared helpers, types, validation, and narrow interfaces.
- Task 07 is complete: canonical replay assembly now lives in `internal/transcript`, `session` delegates transcript queries there, and transcript endpoint behavior remains stable through the API.
- Task 08 is complete: dream orchestration now lives in `internal/memory/consolidation`, while `internal/daemon` only wires the consolidation runtime into boot, session-stop notifications, and shutdown lifecycle.
- Task 09 is complete: the last refac-v2 transport and daemon bridge layers are removed, remaining runtime consumers use the narrowed final interfaces, and the deleted compatibility files no longer sit between `api/httpapi`/`api/udsapi` and `api/core`.

## Shared Decisions
- Shared frontmatter behavior is centralized behind `frontmatter.Split` for delimiter/body extraction and `frontmatter.Decode` for caller-provided metadata decoding.
- Shared daemon request/response DTOs moved out of `internal/apicore/payloads.go` into `internal/api/contract`; `apicore` now keeps only transport helpers such as SSE plumbing and cursors.
- CLI now consumes shared daemon DTOs from `internal/api/contract` via aliases in `internal/cli/client.go`; only CLI-local aggregate/view types such as `WorkspaceDetailRecord`, `HealthStatus`, `IdentityRecord`, and the `memory.MemoryHeader` alias remain outside the shared contract.
- HTTP prompt/AI SDK stream payloads remain transport-local in `internal/httpapi/prompt.go` and should not be moved into `internal/api/contract`.
- The shared API core boundary is now `internal/api/core`; HTTP and UDS transports import that package directly, while transport-local prompt-stream payload shaping remains in `internal/httpapi/prompt.go`.
- Import-boundary enforcement now treats `internal/api/httpapi` and `internal/api/udsapi` as the transport-only roots; future tasks should update boundary rules against those paths rather than the retired top-level transport packages.
- Persistence ownership is explicit by database scope: `internal/store/sessiondb` owns per-session event persistence and writer-loop lifecycle, `internal/store/globaldb` owns the global registry/workspace/observe surfaces, and `internal/store` should stay limited to shared primitives rather than regrowing concrete database responsibilities.
- Transcript ownership is explicit: `internal/transcript` owns canonical replay message types, assembly, and canonical event-envelope marshaling, while `session`, `api/core`, and `daemon` consume `transcript.Message` instead of session-local transcript DTOs.
- Dream runtime ownership is explicit: `internal/memory/consolidation` owns the background check loop, trigger behavior, workspace selection, and dream session spawning; `internal/memory` still owns gate evaluation and lock semantics via `memory.Service` and `memory.ConsolidationLock`.
- Final transport/runtime convergence is direct: `internal/api/httpapi` and `internal/api/udsapi` now call `api/core` and `api/contract` directly without local forwarding wrappers, `internal/daemon` uses the shared `api/core` transport interfaces plus an interface-typed workspace service in `RuntimeDeps`, and `observe` owns an explicit narrowed registry contract instead of embedding the broader `store.SessionRegistry`.

## Shared Learnings
- `config` keeps its historical missing/unterminated frontmatter error strings by mapping shared frontmatter sentinel errors at the call site, while `memory` and `skills` rely directly on shared sentinel categories plus their existing higher-level wrappers.
- For refac-v2 tasks with explicit coverage thresholds, repo gates alone are insufficient evidence because `make verify` and `make test-integration` do not print Go package coverage; direct `go test -cover` runs on the touched packages are required in task closeout.

## Open Risks

## Handoffs
