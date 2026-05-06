---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/udsapi/routes.go
line: 310
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isn,comment:PRRC_kwDOR5y4QM6-UFVr
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Preserve a compatibility path for the old memory endpoints.**

This hard-cuts an unversioned API surface by replacing the previous memory routes (`PUT /api/memory/:filename`, `GET /api/memory/search`, `POST /api/memory/consolidate`) with new verbs/paths. Existing clients will start failing with 404/405 as soon as this ships. Please either keep temporary compatibility aliases or put the Memory v2 surface behind a versioned prefix with an explicit migration window.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/udsapi/routes.go` around lines 278 - 310, Add a
backward-compatibility path: keep the current new handlers but add aliases that
preserve the old unversioned endpoints and/or introduce a versioned prefix for
the new surface. Concretely, create a v2 group (e.g., memoryV2) and move the new
handlers (ReadMemory, WriteMemory, SearchMemory, ReindexMemory, PromoteMemory,
EditMemory, DeleteMemory, etc.) under "/v2/memory"; then add compatibility
routes on the original unversioned group that map the legacy endpoints to the
appropriate current handlers — e.g., register PUT("/:filename") to the handler
that previously served file writes (WriteMemory or EditMemory as appropriate),
GET("/search") -> SearchMemory, and POST("/consolidate") -> ReindexMemory (or
the exact handler that implements consolidation), so existing clients keep
working while new clients use /v2/memory during the migration window.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The requested compatibility aliases conflict directly with this repo’s root `CLAUDE.md`, which mandates greenfield hard cuts and explicitly says not to add compatibility routes, aliases, or migration bridges for removed public surfaces.
  - `internal/api/udsapi/routes.go` intentionally exposes the current Memory v2 route set only; reintroducing legacy `/api/memory/search`, `/api/memory/consolidate`, or `PUT /api/memory/:filename` aliases would violate current repository policy.
  - No code change is required for this batch item.
