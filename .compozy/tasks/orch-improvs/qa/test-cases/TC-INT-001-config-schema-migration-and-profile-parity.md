# TC-INT-001: Config, Schema Migration, And Execution Profile Parity

**Priority:** P0

**Objective:** Prove config defaults, validation, GlobalDB schema, and execution profile management
are consistent across fresh and migrated runtimes and across CLI, HTTP, UDS, native tools, OpenAPI,
generated TypeScript, web data hooks, and docs.

**Requirements Covered:** tasks 01-08, 11-13, 16, 18, 24, 28, 30; ADR-002, ADR-004, ADR-005,
ADR-010.

## Preconditions

- One fresh isolated QA home.
- One migrated QA home created from the schema version immediately before orchestration profile
  migrations.
- Generated OpenAPI and TypeScript artifacts are present.
- Site docs have been generated from the current source.

## Test Steps

1. Boot the fresh QA home.
   **Expected:** GlobalDB contains orchestration columns, profile selector tables, review fields,
   notification cursor tables, bridge subscription tables, and all migration records.

2. Boot the migrated QA home.
   **Expected:** Numbered migrations create the same schema shape as fresh boot; no boot-time
   compatibility repair path is used.

3. Load config with default `[task.orchestration]` values omitted.
   **Expected:** Runtime defaults match site docs and generated examples.

4. Load config with invalid provider override, sandbox ref, and review bounds.
   **Expected:** Config validation rejects invalid values with deterministic errors before runtime
   execution.

5. Create or update an execution profile through HTTP.
   **Expected:** UDS, CLI JSON, native `task_execution_profile_get`, and web data hooks read the
   same normalized profile.

6. Delete the profile through CLI and inspect through HTTP/UDS/native/web.
   **Expected:** All surfaces report the default inherited profile shape, not stale deleted state.

7. Start a run and attempt to mutate profile fields that would change an active worker.
   **Expected:** Mutation is rejected while `tasks.current_run_id` is populated; web edit/delete
   controls mirror the lock.

8. Run codegen checks and generated TypeScript consumer tests.
   **Expected:** `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and web query adapters
   are in sync with the transport contract.

## Behavioral Evidence

- Fresh and migrated DB file paths.
- Schema inspection output or test logs proving table/index/column parity.
- Config files used for valid and invalid runs.
- CLI/HTTP/UDS/native/web outputs for the same profile id.
- Codegen and docs generation command evidence.

## Disruption Probes

- Run profile update concurrently with a claim attempt.
- Try a task provider override when config disallows task provider overrides.
- Try sandbox `none` when config disallows no-sandbox task profiles.

