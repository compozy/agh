---
status: resolved
file: internal/httpapi/memory.go
line: 64
severity: low
author: claude-code
provider_ref:
---

# Issue 005: resolveMemoryLocation reads file twice for read/delete

## Review Comment

The `readMemory` handler (line 64) calls `resolveMemoryLocation()` which internally calls `store.Read()` to verify the file exists and determine scope, then the handler calls `store.Read()` again to get the content. This results in two disk reads for every memory read operation.

The same pattern exists in `deleteMemory` (line 113) — `resolveMemoryLocation()` reads the file, then `store.Delete()` operates on it.

**Impact**: Minor — these are local file reads on a single-user system. No user-visible latency. But the pattern is wasteful and could be simplified.

**Suggested fix**: Have `resolveMemoryLocation` return the content it already read, or split resolution into scope-resolution (which doesn't need to read file content) and content-fetch as separate steps.

**Also affected**: `internal/udsapi/memory.go` (identical pattern at same line numbers)

## Triage

- Decision: `valid`
- Root cause: `resolveMemoryLocation()` uses `store.Read()` as an existence probe, so read and delete handlers perform an unnecessary content read before the actual read/delete operation.
- Evidence: `internal/httpapi/memory.go` probes each candidate scope with `store.Read()`, then `readMemory()` calls `store.Read()` again and `deleteMemory()` calls `store.Delete()` afterward. The same pattern also exists in `udsapi`.
- Fix approach: Switch location resolution to a cheap existence check and keep the actual content read/delete as the single operation performed by the handler. Mirror the low-risk helper change in `udsapi` so the transports stay aligned.
- Resolution: Added `memory.Store.Exists()` and changed both HTTP and UDS location resolution to use existence checks instead of pre-reading file contents.
