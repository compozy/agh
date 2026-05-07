---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/service.go
line: 85
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaqi,comment:PRRC_kwDOR5y4QM6-7HZE
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't override the caller's stale filter here.**

Setting `listOpts.IncludeAll = true` disables the store's stale-row predicate, so `opts.IncludeStale` never changes what `CatalogService.ListModels` merges. A caller asking for `include_stale=false` can still get stale-only models back through the merged projection.

 

Based on learnings "User-visible runtime capabilities in Go backend must expose stable machine-readable control surfaces: CLI verbs with `-o json`/`-o jsonl`, HTTP/UDS parity, discoverable status/config output."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/service.go` around lines 81 - 85, The code currently
forces listOpts.IncludeAll = true before calling s.store.ListRows, which
overrides the caller's stale filter and prevents opts.IncludeStale from taking
effect; change the logic in CatalogService.ListModels so you only set
listOpts.Now = defaultNow(opts.Now) (preserve the copy of opts as listOpts) and
do NOT overwrite IncludeAll, or explicitly set listOpts.IncludeAll =
opts.IncludeStale if you need clarity, then call s.store.ListRows with that
listOpts so the store's stale-row predicate respects the caller's intent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CatalogService.ListModels` currently forces `listOpts.IncludeAll = true` before reading from the store.
  - That bypasses the store’s stale-row filter, so callers asking for `include_stale=false` can still receive stale rows in the merged projection.
  - Fix plan: preserve the caller’s stale-filter intent when listing rows and add a regression test covering the stale filter.
  - Fixed in `internal/modelcatalog/service.go` with regression coverage in `internal/modelcatalog/service_test.go`, then verified with focused package tests plus `make verify`.
