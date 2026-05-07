# Fix Global SQLite Migration Order Drift And Add Guardrails

## Summary

- Root cause confirmed: the daemon correctly refuses to start because `globalSchemaMigrations` changed the identity of an already-applied migration version.
- The observed `/Users/pedronauck/.agh/agh.db` records `17=add_task_orchestration_profile_schema`, `18=add_task_review_gate_schema`, `19=add_notification_cursors`, `20=add_bridge_task_subscriptions`.
- Current code incorrectly expects `17=rebuild_network_conversation_containers` and shifted the previously recorded task/bridge migrations to `18..21`.
- The fix restores append-only migration identity, keeps strict integrity mismatch failures, adds regression coverage for this exact history, and documents the rule in durable project memory plus active agent instructions.

## Key Changes

- Restore the canonical global migration order in `internal/store/globaldb/global_db.go`:
  - `17 add_task_orchestration_profile_schema`
  - `18 add_task_review_gate_schema`
  - `19 add_notification_cursors`
  - `20 add_bridge_task_subscriptions`
  - `21 rebuild_network_conversation_containers`
  - `22 memv2_memory_events`
- Update network conversation migration tests to use `networkConversationMigrationVersion = 21` and seed legacy network DBs from the corrected pre-network history.
- Add a regression test that seeds a DB matching the observed local history through migration `20`, with legacy `network_timeline_log.interaction_id`, then opens it through `OpenGlobalDB` and asserts no integrity mismatch, network migration v21, memory migration v22, intact task/bridge schema, and idempotent reopen.
- Add an append-only registry contract test for the known global migration sequence, emphasizing versions `17..22`.
- Preserve strict integrity behavior in `store.RunMigrations`; do not accept arbitrary mismatches or edit `schema_migrations` in place.
- Do not add one-pass repair unless real DBs are found with the broken inverse sequence.

## Documentation Guardrails

- Add `docs/_memory/lessons/L-021-schema-migration-identity-is-append-only.md`.
- Update `docs/_memory/lessons/README.md` with `L-021`.
- Update root `AGENTS.md` and `CLAUDE.md` under `### Schema Migrations` with the append-only registry rule.
- Update `internal/AGENTS.md` and `internal/CLAUDE.md` with an `internal/store` migration invariant.

## Public Interfaces / Data Contract

- No HTTP, UDS, CLI, OpenAPI, web, or config contract changes.
- The internal data contract is made explicit: global SQLite migration numbers, names, and checksums are immutable once applied anywhere meaningful.
- Fresh DB final schema remains the same. Existing DBs with the observed `17..20` history upgrade by applying only missing migrations `21` and `22`.

## Test Plan

- Run `go test ./internal/store ./internal/store/globaldb -count=1 -race`.
- Run an isolated daemon upgrade proof using a temp copy of `/Users/pedronauck/.agh/agh.db`.
- Verify lesson and instruction guardrails landed in the intended files.
- Run `make verify`.

## Assumptions

- The selected implementation scope is registry and tests, not generic migration-runner redesign.
- The observed local DB history is valid and must be preserved.
- The live `/Users/pedronauck/.agh/agh.db` will not be manually mutated during validation.
- Persistent artifacts are written in English.
