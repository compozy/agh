# TC-FUNC-014: resources/list returns only same-source records

**Priority:** P1
**Type:** Functional
**Package:** internal/extension
**Related Tasks:** 05

## Objective

Validate that the `resources/list` JSON-RPC method enforces source isolation. When extension A calls `resources/list`, it must only see records published by its own source. Records published by extension B must not be visible, even if they share the same kind, scope, or workspace. This is a fundamental tenant isolation property of the extension protocol.

## Preconditions

- The extension runtime is initialized with two active extension sessions:
  - Extension A: `SourceKind="extension"`, `SourceID="ext-A"`, granted kinds `["tool", "skill"]`.
  - Extension B: `SourceKind="extension"`, `SourceID="ext-B"`, granted kinds `["tool"]`.
- Both extensions operate in the same workspace scope (`ws-1`).
- Extension A has published: `tool/a-grep`, `tool/a-sed`, `skill/a-refactor`.
- Extension B has published: `tool/b-lint`, `tool/b-format`.

## Test Steps

1. As extension A, call `resources/list` with `kind="tool"`.
   **Expected:** Returns exactly 2 records: `tool/a-grep` and `tool/a-sed`. Does NOT include `tool/b-lint` or `tool/b-format`.

2. As extension B, call `resources/list` with `kind="tool"`.
   **Expected:** Returns exactly 2 records: `tool/b-lint` and `tool/b-format`. Does NOT include `tool/a-grep` or `tool/a-sed`.

3. As extension A, call `resources/list` with `kind="skill"`.
   **Expected:** Returns exactly 1 record: `skill/a-refactor`.

4. As extension B, call `resources/list` with `kind="skill"`.
   **Expected:** Returns 0 records. Extension B's granted kinds do not include `skill`, and even if they did, there are no `skill` records from source `ext-B`.

5. As extension A, call `resources/list` with no kind filter (list all kinds).
   **Expected:** Returns 3 records: `tool/a-grep`, `tool/a-sed`, `skill/a-refactor`. All from source `ext-A` only.

6. As extension B, call `resources/list` with no kind filter.
   **Expected:** Returns 2 records: `tool/b-lint`, `tool/b-format`. All from source `ext-B` only.

7. Extension A deletes `tool/a-grep`. Extension B calls `resources/list`.
   **Expected:** Extension B's results are unchanged. The deletion of A's record has no effect on B's view.

## Edge Cases

- An extension with no published records calls `resources/list`: returns an empty list, not an error.
- Records published by the daemon itself (source `daemon`) are not visible to any extension via `resources/list`, unless the protocol explicitly exposes them.
- An extension attempts to call `resources/list` with a filter targeting another source's ID (e.g., `source_id="ext-B"`): the filter is ignored or rejected; the source boundary is always the caller's own source.
- Pagination of `resources/list` results respects source isolation on every page.
- A newly published record by extension A is immediately visible in A's next `resources/list` call (read-your-writes consistency).
- Records in the `global` scope from another source are still not visible (source isolation takes precedence over scope visibility).
