# Peer Review Round 2 Summary

## Command

```bash
compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file .compozy/tasks/tools-registry/qa/peer-review-prompt-round2.md
```

## Raw Output

- `.compozy/tasks/tools-registry/qa/peer-review-result-round2.json`
- `.compozy/tasks/tools-registry/qa/peer-review-result-round2.err`

## Verdict

`NEEDS_REWORK`

The reviewer agreed the revised direction is sound, but identified five contract-level blockers before task generation: extension wire payload structs, schema digest canonicalization, MCP bearer injection boundary, hosted MCP bind-token threat model, and approval bridge timeout/cancellation.

## Disposition

- `B-001` Extension wire contracts: resolved in `_techspec.md` Core Interfaces with protocol constants, capability-method mapping, and `provide_tools` / `tools/call` request-response structs.
- `B-002` Schema digest canonicalization: resolved in `_techspec.md` Data Models and ADR-008 with RFC 8785 JCS canonicalization, lowercase SHA-256 digests, and shared SDK/daemon fixtures.
- `B-003` Remote MCP bearer injection: resolved in `_techspec.md` Core Interfaces / Existing MCP Config And Auth and ADR-010 with `MCPCallExecutor` owned by `internal/mcp`.
- `B-004` Hosted MCP bind-token contradiction: resolved in `_techspec.md` Hosted MCP authentication by replacing bearer bind tokens with non-secret bind nonces plus UDS peer credential and AGH binary validation.
- `B-005` Approval bridge wait: resolved in `_techspec.md` Hosted MCP approval bridge, Config Lifecycle, Test Strategy, and ADR-005 with `approval_timeout_seconds`, `approval_timed_out`, `approval_canceled`, and proxy-disconnect cancellation.
- `N-001` Approval timeout / bind nonce TTL defaults: resolved in Config Lifecycle and Safety Invariants.
- `N-002` Go SDK path: resolved in ADR-009 by committing to `sdk/go`.
- `N-003` acpmock and Playwright fixture updates: resolved in Test Strategy.
- `N-004` coverage/race discipline: resolved in Test Strategy.
- `N-005` long sanitized external IDs: resolved in ToolID and reason codes with `id_too_long`.
- `N-006` hook payload delete targets: resolved in Delete Targets.
- `N-007` external read-only trust: resolved in Config Lifecycle with `trusted_sources`.
- `N-008` child task lineage authority: resolved in Network And Tasks and Integration Tests.
