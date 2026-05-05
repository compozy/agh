# Goal (incl. success criteria):

- Implement network-threads task_08: expose conversation containers through public API contracts, HTTP/UDS/agent-native UDS routes, OpenAPI, generated web TypeScript, and parity/legacy-rejection tests.
- Success requires strict rejection of `interaction_id` and `kind:"direct"` at public ingress, HTTP/UDS route parity for thread/direct/work surfaces, agent-native UDS semantics updated, `make codegen`/`make codegen-check`, clean `make verify`, tracking updates, and one local commit only after verification.

# Constraints/Assumptions:

- Repo root: `/Users/pedronauck/Dev/compozy/agh2`.
- No destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit user permission.
- Existing dirty task-tracking/QA deletions and untracked ledgers/memory are pre-existing; do not revert or overwrite unrelated changes.
- Scope is task_08 only: contracts, shared API handlers, HTTP/UDS route registration, agent-native UDS channel behavior, codegen, and tests. CLI/native tools/extensions/web UI implementation are later tasks unless generated contracts require minimal consumer alignment.
- Required skills loaded: `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-contract-codegen-coship`, `agh-code-guidelines`, `agh-test-conventions`, `golang-pro`, `testing-anti-patterns`, `no-workarounds`, `systematic-debugging`.

# Key decisions:

- Use `_techspec.md`, ADR-002, and ADR-003 as source of truth for `surface`, `thread_id`, `direct_id`, and `work_id`.
- Use Task 05/06 store/runtime conversation APIs; do not add new persistence primitives.
- Keep generated files generated via `make codegen`, not manual edits.

# State:

- Implementation in progress; DTOs, validation, shared conversation handlers, route replacement, generated artifacts, focused tests, and minimal web/daemon compile fallout are addressed.

# Done:

- Read workflow shared memory and task_08 memory.
- Scanned other ledgers for relevant network-thread context.
- Read repo/root/internal/web guidance, required skills, task_08, `_tasks.md`, `_techspec.md`, `_design.md`, and ADR-001/002/003.
- Confirmed shared memory says tasks 01-07 are complete and task_08 should consume store/runtime conversation APIs.
- Observed pre-existing dirty worktree entries in task tracking/QA artifacts and untracked ledgers/memory.
- Captured pre-change signal: public contract/codegen still expose `interaction_id`, `NetworkSendRequestFromPayload` maps `interaction_id` into runtime `WorkID`, generated OpenAPI/web types contain `interaction_id`, and HTTP/UDS only register old flat message paths instead of thread/direct/work routes.
- Updated contract DTOs away from `interaction_id`, added hard-cut send JSON rejection for `interaction_id` and `kind:"direct"`, and mapped send/envelope/message payloads to `surface`, container ids, and `work_id`.
- Added shared core handlers for thread list/show/messages, direct list/resolve/show/messages, and work lookup using the Task 05 store API.
- Replaced public HTTP/UDS flat message route registration with the TechSpec thread/direct/work route map; old flat routes remain only in the core unit-test helper for legacy internal handler coverage.
- Added focused tests for contract legacy rejection, send conversation validation, shared conversation read paths, UDS network route parity against documented HTTP/UDS spec routes, and HTTP/UDS direct-room resolve.
- Ran `make codegen` and `make codegen-check`; generated OpenAPI/web contract artifacts no longer include `interaction_id` or old flat message operations.
- First full `make verify` failed during `agh-web:typecheck` because existing web network consumers referenced removed generated operations. Added minimal system adapter/type alignment to fetch messages via the new thread/direct operations while preserving current component contracts; `make web-typecheck` passed afterward.
- A later full `make verify` reached lint/typecheck and exposed daemon composition fallout: `daemon.Registry` did not include `store.NetworkConversationStore`, daemon test doubles lacked the new store methods, and native network send still built `InteractionID`. Updated the daemon registry/test double and native send input to use `surface`, container ids, and `work_id`.
- The next lint/typecheck pass exposed CLI compile fallout: `network send` still registered `--interaction-id` and rendered `interaction_id`. Updated the command and default tests to use `--work-id`, `surface`, `thread_id`, and `direct_id`; default CLI/daemon packages compile.
- Lint then found mechanical issues from the new surfaces: CLI send function length, long testutil callback declarations, and an over-parameterized core limit parser. Split CLI flag registration, wrapped callback types, and made the shared limit parser fixed to the `limit` query key; focused package lint is clean.
- Fixed stale native builtin `network_send` JSON schema so it allows `surface`, `thread_id`, `direct_id`, and `work_id` and no longer allows `interaction_id`; focused daemon native send, builtin descriptor, and bundled network skill tests pass.
- First post-fix `make verify` failed only on unrelated `sdk/typescript/src/integration.test.ts` hitting its fixed 30s timeout under full-suite load; the same focused test passed immediately afterward (`2 passed`, 292ms). No code changes made to that unrelated test.
- Final verification evidence: `make codegen-check` passed; focused changed-package Go tests passed; isolated timeout-sensitive frontend tests passed; after all code changes, `VITEST_MAX_WORKERS=4 make verify` exited 0 with frontend format/lint/typecheck/tests/build, Go lint `0 issues`, Go tests `DONE 8272 tests`, and package boundaries `OK: all package boundaries respected`.
- Self-review correction after that gate: updated build-tagged daemon network collaboration E2E and web MSW/story fixtures to remove stale `--interaction-id`/`kind:"direct"` usage; targeted `make web-typecheck`, `go test -tags integration ./internal/daemon -run '^$' -count=0`, and daemon assertion tests passed.
- Local commit created: `eb30d28d feat: expose network conversation contracts`. Post-commit status has no code/generated files left unstaged; remaining changes are `.compozy` tracking/QA/memory and `.codex` ledger artifacts.

# Now:

- Prepare final response with verification evidence, commit hash, and remaining unstaged tracking note.

# Next:

- None for task_08 implementation.

# Open questions (UNCONFIRMED if needed):

- None.

# Working set (files/ids/commands):

- Task file: `.compozy/tasks/network-threads/task_08.md`
- PRD directory: `.compozy/tasks/network-threads/`
- Workflow memory: `.compozy/tasks/network-threads/memory/MEMORY.md`
- Task memory: `.compozy/tasks/network-threads/memory/task_08.md`
- Expected code surfaces: `internal/api/contract/contract.go`, `internal/api/core/{interfaces.go,network.go,network_details.go,agent_channels.go}`, `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, `internal/api/spec/spec.go`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.
