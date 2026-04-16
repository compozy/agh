# TC-SEC-001: Cross-Source Read Denial

**Priority:** P0
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01, 05

## Objective

Validate that source-level isolation is enforced on all read paths: an extension can only list and retrieve records belonging to its own source. No cross-source data leakage occurs through resources/list or resources/get.

## Preconditions

- Two extensions registered with distinct source identifiers (e.g., `ext-alpha` and `ext-beta`).
- Each extension has published at least one resource record via snapshot (e.g., `ext-alpha` owns `(tool, alpha-tool-1)`, `ext-beta` owns `(tool, beta-tool-1)`).
- Both extensions have active sessions with valid nonces.

## Test Steps

1. As `ext-alpha`, call `resources/list` with no filters.
   **Expected:** Response contains only records where `source == ext-alpha`. No records from `ext-beta` appear in the result set.

2. As `ext-alpha`, call `resources/list` with a filter matching a kind that `ext-beta` has records for but `ext-alpha` does not.
   **Expected:** Response is an empty list, not an error. No information about `ext-beta`'s records is disclosed.

3. As `ext-alpha`, call `resources/get` with the exact `(kind, id)` of a record owned by `ext-beta`.
   **Expected:** Response is 403 Forbidden (or equivalent ACP error code). The response body does not contain any field values from the target record.

4. As `ext-beta`, call `resources/list` to confirm its own records are still intact and unaffected.
   **Expected:** `ext-beta` sees only its own records. Record count and content match what was originally published.

5. Repeat step 3 with a fabricated source identifier that does not match any registered extension.
   **Expected:** 403 Forbidden. No enumeration of valid source identifiers is possible from the error response.

## Edge Cases

- Extension attempts to pass `source` as a query parameter to override filtering (parameter injection).
- Extension sends a resources/list request with an empty source field, expecting to receive all records.
- Concurrent requests from both extensions to verify isolation holds under parallel access.
- Extension unregisters and re-registers; verify it cannot access records from its previous session that were assigned to a different source.

## Threat Model

This test prevents **horizontal privilege escalation between extensions**. Without source-level read isolation, a malicious or compromised extension could enumerate and exfiltrate resources published by other extensions -- including tool definitions, hook bindings, or configuration data. This is the foundational isolation boundary for the multi-extension resource runtime; a failure here would undermine the entire trust model.
