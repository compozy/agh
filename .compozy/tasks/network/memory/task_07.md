# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the daemon-owned network manager and boot/shutdown integration required by task 07.
- Keep session integration late-bound via post-construction setters/callbacks instead of constructor coupling.
- Surface network diagnostics through daemon status/info without exposing broker credentials.
- Finish with required unit/integration tests, verification evidence, task tracking, and one local commit.

## Important Decisions
- The PRD/techspec/ADRs are the approved design baseline for this task, so no separate design round is needed.
- Reuse the existing turn-end notifier seam from task 06 and add any new session-facing network lifecycle seam as a post-construction setter.
- Keep task 07 scoped to composition-root boot/runtime ownership and diagnostics; API/CLI command surfaces stay in later tasks unless required by tests.
- Model the daemon-owned runtime as a single `internal/network.Manager` that composes transport, router, peer presence, delivery, and audit behavior behind one boot-owned lifecycle.
- Expose the runtime to daemon/API layers through the `core.NetworkService` read/send surface plus late-bound session lifecycle setters instead of constructor-time package coupling.

## Learnings
- Pre-change gap confirmed: there is no `internal/network/manager.go`, no `bootNetwork` phase, no session network lifecycle setter, and no network fields in current daemon info/status payloads.
- `session.Manager` already exposes `SetTurnEndNotifier` and `IsPrompting`, which should be consumed by the network manager instead of re-hooking prompt internals.
- `session.Manager` fires `Notifier.OnSessionCreated` / `OnSessionStopped` after session activation/finalization, which is the natural late-bound join/leave integration point from daemon composition code.
- `make verify` initially surfaced task-local cleanup work rather than architectural issues: rollback paths in `network.Manager.JoinSpace()` needed explicit error handling, default-test-only helpers had to be split from integration helpers, and the daemon package needed one more diagnostics-focused unit test to restore coverage above 80%.
- The final touched-package coverage snapshot is `internal/session` 81.9%, `internal/network` 80.4%, `internal/daemon` 80.5%, and `internal/api/core` 80.1%.

## Files / Surfaces
- `internal/network/{manager.go,delivery.go,transport.go,peer.go,router.go,lifecycle.go}`
- `internal/session/{interfaces.go,manager.go,manager_helpers.go,manager_lifecycle.go,manager_test.go}`
- `internal/daemon/{boot.go,daemon.go,info.go,daemon_test.go,daemon_integration_test.go}`
- `internal/api/{core,contract,httpapi,udsapi,testutil}`
- `internal/cli/daemon.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Fixed `make verify` lint failures by handling rollback errors in `internal/network/manager.go`, removing dead test aliases/helpers, and replacing direct `nil` context calls in manager tests with a typed helper.
- Fixed the daemon coverage regression by adding targeted tests for `daemonNetworkInfo()` and `NetworkInfo.Validate()`.
- `make verify` hit one transient `internal/session` concurrent-stop failure once; the package passed the direct `go test -race -parallel=4 ./internal/session` repro immediately afterward, and the subsequent full `make verify` run passed cleanly.

## Ready for Next Run
- None. Task 07 implementation, verification, and tracking are complete.
