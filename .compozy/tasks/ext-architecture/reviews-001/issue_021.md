---
status: resolved
file: internal/extension/host_api.go
line: 499
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAab,comment:PRRC_kwDOR5y4QM62zlsm
---

# Issue 021: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Recall can panic when memory is not configured.**

`memorySourcesForRecall` returns `h.memory` directly for the default/global cases without checking for nil, and `handleMemoryRecall` then dereferences `source.store` on the first `Scan`. `memory/store` and `memory/forget` already reject a missing store via `memoryStoreFor`; recall needs the same guard.



Also applies to: 656-687

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 445 - 499, handleMemoryRecall
can panic when a memory source has no configured store because
memorySourcesForRecall may return h.memory with nil store; update
handleMemoryRecall to guard against nil stores the same way memoryStoreFor does:
after obtaining sources from memorySourcesForRecall, check each source.store for
nil and return a descriptive RPC error (e.g., invalidParamsRPCError or a new
missingStoreRPCError) instead of dereferencing it, or call the existing
memoryStoreFor helper to validate/resolve the store before calling
source.store.Scan; apply the same nil-store guard to the other recall path
referenced (the similar code around the 656-687 region).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `memorySourcesForRecall` returns `h.memory` directly for default/global recall paths, and `handleMemoryRecall` then dereferences `source.store` immediately. When the handler is created without a memory store, recall can panic.
  Fix approach: validate the memory store before constructing recall sources so recall returns a descriptive error instead of dereferencing a nil store.
