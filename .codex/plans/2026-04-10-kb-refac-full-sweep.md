# KB Refactor Full Sweep

## Summary

- Execute the entire `kb-refac` scope in phased checkpoints, not a single big-bang batch.
- Treat the techspec as the scope authority, but treat the live codebase as the implementation authority for exact file sizes, helper names, and stale counts.
- Fix root causes: composition-root concentration, duplicated session-start flow, repetitive ACP dispatch, oversized hook and transport surfaces, dead exported API surface, and repeated workspace primitives.
- Keep runtime behavior and wire formats stable throughout; each phase closes only after its own verification gate and without temporary bridge code left behind.

## Key Changes

1. Phase 1: safe cleanup and task artifact sync
   - Remove confirmed dead exports and unused shadcn UI components listed in the techspec.
   - Unexport test-only production helpers and replace callers with direct constructors or same-package helpers instead of keeping test-facing production API.
   - Create the missing `kb-refac` ADR files referenced by the techspec and update the techspec only when implementation proves a claim stale.
2. Phase 2: session lifecycle dedup
   - Extract one private `startSession` pipeline shared by create and resume.
   - Keep `Create` and `Resume` limited to source-specific preamble logic that prepares a `sessionStartSpec`.
3. Phase 3: ACP and daemon orchestration
   - Replace ACP inbound switch dispatch with a typed registry plus a small decode/execute helper.
   - Split ACP driver startup into subprocess spawn, ACP connection initialization, and session negotiation helpers.
   - Refactor daemon construction into `applyDefaults()` plus boot phases backed by a `bootState` and cleanup stack.
4. Phase 4: hook boundary reduction
   - Split hook dispatch implementation by responsibility before changing interfaces.
   - Replace the single 21-method session hook dependency with grouped domain subinterfaces collected in a hook-set container injected into the session manager.
   - Provide no-op group defaults so tests only implement the groups they exercise.
5. Phase 5: transport, registry, and shared value objects
   - Split the large CLI skill, HTTP server, and skills registry files by concern while keeping package boundaries and external entrypoints unchanged.
   - Flatten session SSE streaming into helpers for backlog replay, polling, and stop-event emission.
   - Move CLI SSE decoding into a neutral shared package.
   - Introduce a tiny neutral workspace-reference package reused across payload types without changing external JSON field names.

## Interfaces And Types

- Replace the current aggregate session hook dependency with grouped internal interfaces plus a hook-set container.
- Add private orchestration types such as `sessionStartSpec` and `bootState`.
- Add a neutral shared workspace-reference value object reused across payload types without changing external JSON field names.

## Test Plan

- End every phase with targeted package tests first, then `make verify`.
- Run `make test-integration` after phases that change daemon, session, hooks, ACP startup, or API transport behavior.
- Verify create vs resume parity, failed-start cleanup, ACP invalid-params and method-not-found handling, daemon boot cleanup ordering, hook mutation and denial behavior, HTTP route coverage parity, SSE replay/poll/stop behavior, and JSON compatibility for embedded workspace refs.
- Verify frontend cleanup with `make web-test` and `make web-build` after unused UI component removal.

## Assumptions

- Scope is the full sweep from the techspec, including unused UI cleanup and task-artifact synchronization.
- Delivery is phased; no phase is complete if it leaves behind transitional bridge code.
- The missing `kb-refac` ADR references should be created unless implementation proves one obsolete.
