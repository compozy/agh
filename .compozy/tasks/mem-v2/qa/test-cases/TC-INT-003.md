# TC-INT-003: CLI, HTTP, UDS, Native Tool, And Reference Parity

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify Memory v2 is agent-manageable across every public control surface and that generated CLI/API references describe the same final hard-cut surface.

## Preconditions

- [ ] Isolated daemon exposes HTTP and UDS endpoints.
- [ ] Native tool invocation path is available in a root agent session.
- [ ] Generated CLI/API reference files exist from task_24.

## Test Steps

1. **List/show/search parity**
   - Input: run `agh memory list`, `agh memory show`, `agh memory search`; run matching HTTP/UDS/native tool calls.
   - **Expected:** JSON payloads agree on selector, filename, scope, type, content summary, and deterministic error shape.

2. **Write/edit/delete parity**
   - Input: run CLI write/edit/delete and matching HTTP/UDS routes where supported.
   - **Expected:** All mutation paths return controller-backed decisions and emit compatible event rows.

3. **Decision history/revert parity**
   - Input: `agh memory decisions list/show/revert` and matching API routes.
   - **Expected:** Decision payloads are redaction-safe and can be correlated to events.

4. **Native tool IDs**
   - Input: list native built-ins and invoke `agh__memory_list`, `agh__memory_show`, `agh__memory_search`, `agh__memory_propose`, `agh__memory_note`.
   - **Expected:** Final IDs exist; `agh__memory_read`, `agh__memory_history`, and write bypass IDs do not exist.

5. **Generated reference truth**
   - Input: inspect generated docs and run focused site tests.
   - **Expected:** `show` and `dream trigger` exist; `read`, `consolidate`, `GET /api/memory/search`, and current-tense `PUT /api/memory/{filename}` do not.

## Evidence To Capture

- Side-by-side CLI, HTTP, UDS, and native-tool JSON files.
- Diff or jq-normalized comparison output.
- Native tool list.
- Focused site test log.

## Failure Criteria

- Any public surface still advertises or accepts a hard-cut Memory v1 verb/route/tool ID.
- Mutation payloads differ in controller decision semantics.

