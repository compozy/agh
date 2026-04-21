---
status: resolved
file: internal/api/udsapi/stream_helpers_test.go
line: 179
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMl,comment:PRRC_kwDOR5y4QM65IPD4
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Guard `close(done)` to avoid poll-loop panics.**

Line 178 closes `done` unconditionally from a polling callback. If the callback is called again, this panics (`close of closed channel`) and makes the test flaky.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 import (
     "bytes"
     "context"
     "encoding/json"
     "net/http"
     "net/http/httptest"
+    "sync"
     "testing"
     "time"
@@
     homePaths := newTestHomePaths(t)
     done := make(chan struct{})
+    var doneOnce sync.Once
     observer := stubObserver{
         QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
-            close(done)
+            doneOnce.Do(func() { close(done) })
             return []store.EventSummary{{
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sync"
    "testing"
    "time"
)

    homePaths := newTestHomePaths(t)
    done := make(chan struct{})
    var doneOnce sync.Once
    observer := stubObserver{
        QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
            doneOnce.Do(func() { close(done) })
            return []store.EventSummary{{
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/stream_helpers_test.go` around lines 177 - 179, The test
currently calls close(done) unconditionally inside the QueryEventsFn callback
which races if the callback is invoked multiple times; fix this by guarding the
close with a sync.Once (e.g., declare a doneOnce sync.Once in the test setup)
and replace the direct close(done) in QueryEventsFn with doneOnce.Do(func(){
close(done) }), ensuring only the first invocation closes the channel and
prevents "close of closed channel" panics.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `QueryEventsFn` can be invoked more than once by the polling helper, but the test closes `done` unconditionally on every callback.
  - A repeated callback would panic with `close of closed channel`, which makes the test flaky instead of deterministic.
  - I will guard the close with `sync.Once` in the UDS stream-helper test.
