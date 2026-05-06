Goal (incl. success criteria):

- Execute `.compozy/tasks/mem-v2/task_02.md`: add numbered Slice 1 memory schema migrations, stable `.agh/workspace.toml` workspace_id resolution, idempotent path-keyed catalog backfill to workspace_id, focused tests for fresh/migrated/reopen/idempotent behavior, clean verification, tracking updates, and one local commit.

Constraints/Assumptions:

- Work in `/Users/pedronauck/dev/compozy/agh3`; prompt cwd `/Users/pedronauck/dev/compozy/looper` is not the implementation repo for this task.
- Required workflow memory paths are `.compozy/tasks/mem-v2/memory/MEMORY.md` and `memory/task_02.md`; read before code edits.
- Required skills used/read: `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-schema-migration`, `agh-code-guidelines`, `agh-test-conventions`, `golang-pro`, `testing-anti-patterns`, `systematic-debugging`, `no-workarounds`.
- Do not run destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- No `EnsureSchema` fallback, no dual durable authority between `workspace_root` and `workspace_id`; greenfield hard cut.
- Automatic local commit is enabled only after clean verification, self-review, memory/tracking updates.

Key decisions:

- Task 02 scope is narrower than the full Slice 1 TechSpec: schema DDL, workspace resolver identity, and path-keyed backfill foundation only.
- `workspace_id` format is ULID (26-char Crockford base32), per TechSpec and ADR-004 post-review refinement.
- Permission-denied or invalid workspace identity resolution must fail closed with deterministic errors.
- Path-keyed `workspace_root` may be used only as legacy input to a bounded migration/backfill; after migration, `workspace_id` is authoritative.

State:

- Implementation is verified pre-commit and committed locally as `96bb1d3f` (`feat: add memory v2 workspace identity schema`). Current full `make verify` passes after schema/resolver/backfill fixes and SDK integration timeout hardening.
- First commit attempt was rejected by commitlint because the message was not Conventional Commit formatted; staged changes remain code-only.
- Second commit attempt was rejected because repo commitlint requires an empty scope; third attempt succeeded with `feat: ...`.
- Post-commit `make verify` passed: frontend tests `329 passed (329)`, Go tests `DONE 8090 tests`, lint `0 issues`, package boundaries `OK`.
- First full `make verify` retry failed only in `sdk/typescript/src/integration.test.ts` with a 30s Vitest timeout; isolated rerun of that exact test file passed in 166ms, indicating a full-suite load-sensitive failure rather than a Task 02 regression.
- Second full `make verify` reproduced the same SDK integration timeout. Applied a narrow verification-hardening change to `sdk/typescript/vitest.config.ts`: `fileParallelism: false` for the extension SDK project, because its integration tests spawn subprocesses and share built SDK artifacts.
- Root Vitest gate now passes after the SDK scheduling fix: `bun run test` reported 329 test files and 2088 tests passing.
- Full `make verify` then reached Go race tests and failed on two Task 02 identity contract mismatches: API memory history still expected the path-style workspace query string instead of the durable workspace ID, and one daemon native-tools test wrote workspace memory before creating the workspace root directory.
- Corrected those tests and validated affected Go packages under race: `go test -race -parallel=4 ./internal/api/core ./internal/daemon ./internal/memory ./internal/store/globaldb` passed.
- Full `make verify` still reproduced the SDK integration timeout under full verification load. Raised only `sdk/typescript/src/integration.test.ts` real-stdio integration timeout from 30s to 120s, matching the existing SDK build timeout while keeping assertions unchanged.
- `bunx vitest run --project extension-sdk --reporter verbose` passed after the SDK timeout adjustment.

Done:

- Read root/internal `AGENTS.md` and `CLAUDE.md` for AGH.
- Read workflow memory files and Task 02.
- Read `_techspec.md`, `_tasks.md`, and ADR-001 through ADR-012.
- Read related mem-v2 ledgers for task generation, Task 01 contract completion, and spec review context.
- Loaded AGH schema/code/test skill references.
- Added `.agh/workspace.toml` identity creation/loading with ULID validation and fail-closed invalid/permission-denied errors.
- Wired resolver results and cache hits to expose stable `ResolvedWorkspace.WorkspaceID`.
- Added memory catalog v3-v7 migrations for workspace identity, events, chunks/FTS, decisions, recall signals, and consolidations.
- Migrated runtime catalog/history/search/health paths from durable `workspace_root` ownership to `workspace_id`.
- Added global DB v17 `memory_events` migration and legacy operation-log conversion.
- Added focused resolver/schema/backfill/reopen tests; fixed migration bugs found by those tests.

Now:

- Apply final verification checklist and report completion evidence.

Next:

- Final response with commit hash, verification report, and remaining uncommitted tracking/memory files.

Open questions (UNCONFIRMED if needed):

- None blocking.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-05-MEMORY-memory-schema-workspace.md`
- `.compozy/tasks/mem-v2/task_02.md`
- `.compozy/tasks/mem-v2/_techspec.md`
- `.compozy/tasks/mem-v2/_tasks.md`
- `.compozy/tasks/mem-v2/memory/MEMORY.md`
- `.compozy/tasks/mem-v2/memory/task_02.md`
- `.compozy/tasks/mem-v2/adrs/adr-001.md` through `adr-012.md`
- `internal/workspace/identity.go`
- `internal/workspace/resolver.go`
- `internal/memory/catalog.go`
- `internal/memory/store.go`
- `internal/store/globaldb/global_db.go`
- `go test ./internal/workspace ./internal/memory ./internal/store/globaldb`
- `bunx vitest run --config vitest.config.ts src/integration.test.ts --reporter verbose` from `sdk/typescript` passed.
- `bun run test` passed after setting SDK Vitest `fileParallelism: false`.
- `go test -race -parallel=4 ./internal/api/core ./internal/daemon ./internal/memory ./internal/store/globaldb` passed.
- `bunx vitest run --project extension-sdk --reporter verbose` passed.
- `make verify` passed pre-commit.
