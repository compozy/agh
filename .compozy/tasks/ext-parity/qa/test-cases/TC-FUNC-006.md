# TC-FUNC-006: Typed store Get/List enforces actor boundary

**Priority:** P0
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 02

## Objective

Validate that the typed `Store[T].Get` and `Store[T].List` methods enforce actor-scoped boundaries. A typed store bound to a specific actor must reject or filter out records that fall outside the actor's source, granted kinds, or scope. This prevents extensions from reading records they do not own or are not authorized to access.

## Preconditions

- A fresh resource store is initialized with schema applied.
- Two distinct actors are configured:
  - Actor A: `SourceKind="extension"`, `SourceID="ext-A"`, granted kinds `["tool", "skill"]`, scope `workspace` with `ScopeID="ws-1"`.
  - Actor B: `SourceKind="extension"`, `SourceID="ext-B"`, granted kinds `["tool"]`, scope `workspace` with `ScopeID="ws-1"`.
- Records are pre-seeded:
  - `tool/t1` owned by `ext-A`, scope `ws-1`
  - `tool/t2` owned by `ext-B`, scope `ws-1`
  - `skill/s1` owned by `ext-A`, scope `ws-1`
  - `tool/t3` owned by `ext-A`, scope `ws-2` (different workspace)

## Test Steps

1. Create a `Store[ToolSpec]` bound to Actor A. Call `List` with no filters.
   **Expected:** Returns `tool/t1` and `tool/t3` (if scope filtering is caller-side) or only `tool/t1` (if the store enforces `ScopeID="ws-1"` from the actor grant). Does NOT return `tool/t2` (owned by `ext-B`).

2. Using Actor A's `Store[ToolSpec]`, call `Get` for `tool/t2`.
   **Expected:** Returns an error or empty result. Actor A must not be able to read records owned by source `ext-B`.

3. Create a `Store[SkillSpec]` bound to Actor B. Call `List`.
   **Expected:** Returns zero results. Actor B's granted kinds are `["tool"]` only, so querying for `skill` records is either rejected at construction time or returns empty.

4. Using Actor B's `Store[ToolSpec]`, call `List`.
   **Expected:** Returns only `tool/t2`. Actor B must not see `tool/t1` or `tool/t3` (owned by `ext-A`).

5. Using Actor A's `Store[ToolSpec]`, call `Get` for `tool/t1`.
   **Expected:** Returns the full `Record[ToolSpec]` with decoded payload, confirming actor A can read its own records.

6. Using Actor A's `Store[ToolSpec]`, call `Get` for `tool/t3`.
   **Expected:** Returns the record if scope filtering does not restrict cross-workspace reads for the owning source, or returns an error/empty if the store enforces the actor's scope strictly.

## Edge Cases

- An actor with an empty granted kinds list (`[]`) cannot access any records through any typed store.
- An actor attempts to construct a `Store[T]` for a kind not in its grant: the constructor returns an error or the store always returns empty results.
- Records in the `global` scope: visible to all actors whose grants include the kind, regardless of `ScopeID`.
- A deleted or soft-deleted record is not visible through `Get` or `List`, even if the actor owns it.
- `List` with pagination: boundary enforcement applies to every page, not just the first.
