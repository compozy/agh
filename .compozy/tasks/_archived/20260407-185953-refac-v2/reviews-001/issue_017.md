---
status: resolved
file: internal/cli/client.go
line: 106
severity: medium
author: claude-code
provider_ref:
---

# Issue 017: WorkspaceDetailRecord not centralized in contract

## Review Comment

`WorkspaceDetailRecord` is the only composite response type defined locally in the CLI instead of referencing `api/contract`. Every other response type (`SessionRecord`, `AgentRecord`, `WorkspaceRecord`, etc.) is aliased from `contract`. This means the CLI and server can silently drift: if the server adds a new field to the workspace detail response, the CLI struct won't pick it up unless manually updated.

This violates ADR-002 which established `api/contract` as "the canonical home for shared API request/response DTOs" used by CLI and transports.

**Fix:** Add a `WorkspaceDetailPayload` to `internal/api/contract/contract.go` with the composite fields (workspace record + sessions + recent events), then alias it in `cli/client.go`.

## Triage

- Decision: `valid`
- Root cause: Workspace detail is the only shared API payload still defined privately in the CLI layer, which breaks the refac-v2 contract centralization goal and creates silent drift risk between transport and client code.
- Fix approach: Move the composite workspace-detail DTO into `internal/api/contract` and alias it from the CLI client, then update dependent tests and helpers to use the shared type.
- Resolution: Implemented by centralizing the shared payload in `api/contract`; full repository verification passed.
