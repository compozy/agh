---
status: resolved
file: internal/httpapi/sessions.go
line: 0
severity: low
author: claude-code
provider_ref:
---

# Issue 021: Massive code duplication between httpapi and udsapi packages

## Review Comment

The two packages duplicate nearly all handler logic, payload types, SSE helpers, and parsing functions -- approximately 800+ lines of duplicated production code. Types like `sessionPayload`, `agentPayload`, `observeEventPayload`, `errorPayload`, `sseMessage`, `flushWriter`, etc. are defined identically in both packages. Handler implementations (`listSessions`, `createSession`, `getSession`, `stopSession`, etc.) are functionally identical.

The only meaningful differences are: (a) httpapi has CORS middleware, (b) httpapi's prompt uses Vercel AI SDK SSE format while udsapi uses raw event-per-SSE, (c) udsapi reads config.HTTP.Port directly while httpapi uses the resolved port.

Any bug fix or feature addition must be applied in two places. This is a maintenance burden and bug-duplication risk.

**Suggested fix:** Extract shared types, handler logic, and SSE helpers into a common internal package (e.g., `internal/apicore`). Have httpapi and udsapi compose shared handlers with their transport-specific middleware.

## Triage

- Decision: `invalid`
- Notes: The duplication between `httpapi` and `udsapi` is real, but it is a maintainability refactor, not a discrete correctness defect that can be resolved inside this review batch without broad architectural churn. This batch is scoped to bug remediation, not transport-layer consolidation.
