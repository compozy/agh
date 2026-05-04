---
status: resolved
file: internal/session/query.go
line: 198
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R112,comment:PRRC_kwDOR5y4QM663fCN
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find remaining non-test usages of the uncancelable wrapper.
rg -n --type=go '\.readMeta\(' -g '!**/*_test.go'

# Show the wrapper itself for quick inspection.
sed -n '190,205p' internal/session/query.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if the file exists and show the relevant section
fd -name "query.go" -path "*/internal/session/*"

# Get context around the mentioned lines
head -n 210 internal/session/query.go | tail -n 30

# Search for all readMeta references (both test and non-test)
rg -n 'readMeta' internal/session/ -A 2

# Search more broadly for readMeta usage across the codebase
rg -n 'readMeta' --type=go
```

Repository: compozy/agh

Length of output: 25495

---



</details>

**Delete the unused `readMeta` wrapper.**

The `readMeta` method at lines 196–198 hardcodes `context.Background()` into a path that can resolve workspaces and persist legacy-provider repairs. Production code has already been converted to call `readMetaWithContext(ctx, ...)` directly; only test code in `query_test.go` references this wrapper. Remove it to eliminate the uncancelable code path.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/query.go` around lines 196 - 198, Remove the unused wrapper
readMeta that calls readMetaWithContext(context.Background(), id) to eliminate
the uncancelable code path; delete the function definition of readMeta in
Manager (the one returning m.readMetaWithContext(context.Background(), id)) and
update any remaining tests (query_test.go) to call readMetaWithContext with an
explicit context instead of readMeta. Ensure no other code references readMeta
after removal and run tests to confirm replacements are correct.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `Manager.readMeta` is an unused production wrapper that hardcodes `context.Background()` even though metadata reads can trigger legacy-provider repair and workspace resolution.
- Removing that dead path is correct, and it requires a minimal accompanying update in `internal/session/query_test.go` so the direct unit tests call `readMetaWithContext` explicitly.
- Resolved by deleting the dead wrapper from `internal/session/query.go`, updating `internal/session/query_test.go` to use explicit contexts, and verifying the session package plus the full repo gate.
