---
status: resolved
file: internal/memory/store.go
line: 343
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575b1Y,comment:PRRC_kwDOR5y4QM65BvYP
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate `q` before catalog warm-up, and cap `limit` server-side.**

`ensureCatalogReady()` runs before the query is validated, so `q=""` or punctuation-only input can still trigger a full reindex/scan and only then fail. Also, the HTTP layer forwards the client-provided `limit` directly into this method (`internal/api/core/memory.go:50-70`), and this path only defaults it — it never bounds it. Reject tokenless queries and clamp `limit` before Line 341.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 327 - 343, In Search, validate the
incoming query string (parameter `query`) immediately after normalizing
scope/workspace and before calling `s.ensureCatalogReady` to avoid triggering a
full catalog warm-up for empty or punctuation-only inputs; reject tokenless
queries (empty or that contain no alphanumeric tokens) by returning a clear
error. Also clamp the requested `opts.Limit` into a server-side bounded value
(e.g., min defaultSearchLimit and maxSearchLimit) and assign that to `limit`
before any work is done; update references in this function (Search,
SearchOptions, defaultSearchLimit) and add/ reuse a small helper (e.g.,
`isTokenlessQuery`) if helpful. Ensure the validation and clamping occur prior
to the `ensureCatalogReady` call so expensive operations are avoided for invalid
requests.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Store.Search` currently normalizes scope, defaults `Limit`, then warms the catalog before any query validation, so empty or punctuation-only input can trigger unnecessary catalog work before failing inside the catalog/fallback search helpers.
  - The method also accepts arbitrarily large positive limits from callers because it only applies a default when `Limit <= 0`; there is no server-side clamp before the expensive work starts.
  - Completed fix: `Store.Search` now rejects tokenless queries before catalog warm-up and clamps limits through the shared search-limit helper before any expensive work occurs.
  - Completed validation: added regression coverage for tokenless query rejection without catalog creation and oversized limit clamping in `internal/memory/store_test.go`.
  - Verification: `make verify` passed after the change set.
