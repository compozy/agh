---
status: resolved
file: internal/httpapi/memory.go
line: 1
severity: high
author: claude-code
provider_ref:
---

# Issue 002: ~420 lines of memory handler code duplicated across httpapi/udsapi

## Review Comment

`internal/httpapi/memory.go` and `internal/udsapi/memory.go` are near-identical (~420 lines each). Every type, handler method, and helper function is copy-pasted between the two files. The only differences are error message prefixes (`httpapi:` vs `udsapi:`).

Duplicated items include:
- 7 type definitions: `memoryWriteRequest`, `memoryReadResponse`, `memoryMutationResponse`, `memoryConsolidateRequest`, `memoryConsolidateResponse`, `memoryHealthPayload`, `memoryLocation`
- 5 helper functions with zero transport-layer specificity: `resolveMemoryWriteScope`, `parseOptionalMemoryScope`, `resolveMemoryWorkspace`, `newMemoryValidationError`, `statusForMemoryError`
- 5 shared business logic methods: `listMemoryHeaders`, `resolveMemoryLocation`, `memoryStoreFor`, `memoryHealthWorkspaces`, `memoryHealth`

**Impact**: Any bug fix or behavior change must be applied in two places. This has already produced identical test files (`memory_test.go` in both packages). As the memory API grows, divergence bugs become inevitable.

**Suggested fix**: Extract the transport-agnostic types and helper functions into `internal/memory/`. Move shared business logic (scope resolution, validation, header listing, health aggregation) into methods on `memory.Store` or a new `memory.APIHelper` struct. The Gin handler methods stay in each transport package but become thin wrappers.

**Also affected**: `internal/udsapi/memory.go` (identical code)

## Triage

- Decision: `invalid`
- Reasoning: The duplication between `internal/httpapi/memory.go` and `internal/udsapi/memory.go` is real, but in the current batch it is a maintainability concern rather than a demonstrated correctness defect. The two handlers are still behaviorally aligned, mirrored by equivalent tests, and the requested extraction would require a broad cross-package refactor that materially exceeds the concrete bug-fix scope of this round.
- Scope note: I will still keep mirrored low-risk behavior fixes aligned where needed, but I am not treating the full transport deduplication request as a blocking issue for this batch.
- Resolution: Closed as non-blocking for this batch after code inspection confirmed no current behavioral divergence. Mirrored low-risk behavior fixes were still kept aligned across both transports.
