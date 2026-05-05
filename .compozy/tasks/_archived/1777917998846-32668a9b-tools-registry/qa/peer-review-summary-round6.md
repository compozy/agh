# Peer Review Summary Round 6

- Verdict from Opus: `NEEDS_REWORK`
- Blocking findings:
  - `B-001` Config validation contradicted the rest of the spec by rejecting remote `sse` while ADR-010, ADR-011, the MCP library section, tests, and Safety Invariant 26 preserved it.
  - `B-002` `approval_token` had no defined producer surface for CLI/HTTP/UDS flows.

Disposition after review:

- `B-001` addressed in `_techspec.md` by removing the legacy `sse` rejection and making validation explicitly accept `{stdio, http, sse}` with no silent transport rewrites.
- `B-002` addressed in `_techspec.md` by adding `POST /api/tools/{id}/approvals`, UDS/CLI parity, issuance/storage/TTL/single-use semantics, deterministic error codes, and Safety Invariant 27.

Non-blocking nits were triaged in the TechSpec `## Nits` section under “Peer review round 6 blockers and nits disposition”.
