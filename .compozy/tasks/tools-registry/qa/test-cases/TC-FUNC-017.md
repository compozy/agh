# TC-FUNC-017 — Sanitized external-name collision (`conflicted_sanitized_name`)

- **Priority:** P2
- **Type:** Functional / collision
- **Trace:** Task 09, ADR-007

## Objective

Prove two distinct raw `(server, tool)` pairs that normalize to the same canonical ID surface `conflicted_sanitized_name`. Both remain operator-visible with provenance and session-hidden.

## Test Steps

1. MCP server `Foo-Bar` exposes tool `baz`; another MCP server `foo.bar` exposes tool `baz`.
2. Both normalize to `mcp__foo_bar__baz`.
3. Operator view shows both with `conflicted_sanitized_name` and raw provenance preserved.
4. Session projection excludes both.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestSanitizedCollision`
