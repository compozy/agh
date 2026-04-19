---
status: resolved
file: internal/daemon/daemon_test.go
line: 3835
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUD,comment:PRRC_kwDOR5y4QM65IlPE
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# 1) Find EventQuery definition and documented semantics.
rg -n --type=go -C3 'type\s+EventQuery\s+struct'

# 2) Find concrete Events implementations using EventQuery.
rg -n --type=go -C5 'func\s+\(.*\)\s+Events\(\s*.*EventQuery'

# 3) Inspect ordering/limit behavior used by production paths.
rg -n --type=go -C5 'AfterSequence|Limit|ORDER BY|sequence|event_sequence'
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search for the actual Events implementation in production code (not test code)
rg -n --type=go 'func.*Events\(' --glob='!*_test.go' -A 15 | head -100

# Also search specifically in session or store implementations
rg -n --type=go 'func.*\(.*\).*Events\(' --glob='internal/store/*' --glob='internal/session/*' -A 15
```

Repository: compozy/agh

Length of output: 7623

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find the recorder.Query implementation
rg -n --type=go 'func.*Query\(' --glob='internal/session/*' -A 20

# Also search for QueryRecorder or similar interfaces/types
rg -n --type=go 'type.*Recorder|QueryRecorder' --glob='internal/session/*' -A 10

# Search for session event queries in globaldb
rg -n --type=go 'SessionEvent' --glob='internal/store/globaldb/*' -B 2 -A 15 | grep -A 15 'Query\|Limit\|AfterSequence'
```

Repository: compozy/agh

Length of output: 20614

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find EventRecorder interface definition
rg -n --type=go 'type EventRecorder interface' -A 10

# Find Query method implementation in store (globaldb or similar)
rg -n --type=go 'func.*Query.*EventQuery' --glob='internal/store/*' -A 20 | grep -A 20 'SessionEvent\|session'

# Search for session event query implementation
rg -n --type=go 'Query.*ctx.*EventQuery' --glob='internal/store/globaldb/*' -B 2 -A 20
```

Repository: compozy/agh

Length of output: 831

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find implementations of EventRecorder.Query (not in tests)
rg -n --type=go 'func.*\(.*\).*Query\(ctx context.Context, query store.EventQuery\)' --glob='!*_test.go' -A 25

# Also try a broader search for SQL query building with Limit and AfterSequence
rg -n --type=go 'AfterSequence.*Limit|Limit.*AfterSequence' --glob='internal/store/globaldb/*' -B 5 -A 10
</parameter>
```

Repository: compozy/agh

Length of output: 2174

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Get more lines from the Query implementation to see the complete limit logic
sed -n '316,360p' internal/store/sessiondb/session_db.go
```

Repository: compozy/agh

Length of output: 1480

---



</details>

**Fix fake `Events` method to return events in ascending sequence order.**

The production `SessionDB.Query()` returns events sorted by sequence in ascending order (lines 340-341: `ORDER BY sequence DESC LIMIT ... ORDER BY sequence ASC`). The fake implementation at lines 3832-3833 slices filtered events but does not sort them, potentially allowing tests to pass with event streams in any order. Harness reentry tests could mask bugs in code that depends on events being properly ordered.

Update the fake to sort filtered events by sequence before returning or slicing to Limit.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3813 - 3835, The fake Events
implementation (fakeSessionManager.Events) currently filters sessionEvents but
does not guarantee ascending sequence order; update it to sort the filtered
[]store.SessionEvent by event.Sequence in ascending order (e.g., using
sort.Slice) before applying the Limit truncation based on query.Limit and
returning; ensure you sort the filtered slice (not the original
store.sessionEvents) so behavior matches production SessionDB.Query ordering and
then apply the existing slicing logic that keeps the last N events.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The fake `Events` implementation preserved insertion order from `sessionEvents`, but production `SessionDB.Query()` guarantees ascending sequence order after filtering and limiting. That mismatch meant tests could accidentally rely on the fake's internal slice order instead of the real store contract. I sorted a filtered copy by `Sequence` before applying the limit and added a regression test in the scoped daemon test file. Verified with `go test ./internal/daemon -count=1` and `make verify`.
