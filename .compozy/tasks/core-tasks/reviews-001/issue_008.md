---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1192
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562anr,comment:PRRC_kwDOR5y4QM63mgQ4
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Protect the session counter from concurrent access.**

`StartTaskSession` increments `started` without synchronization. These HTTP tests exercise a real server, so concurrent starts can race here and make `-race` fail or reuse session IDs.


<details>
<summary>Suggested fix</summary>

```diff
 type integrationTaskSessionExecutor struct {
+	mu      sync.Mutex
 	started int
 }

 func (e *integrationTaskSessionExecutor) StartTaskSession(_ context.Context, _ taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
+	e.mu.Lock()
+	defer e.mu.Unlock()
 	e.started++
 	return &taskpkg.SessionRef{SessionID: fmt.Sprintf("task-sess-%d", e.started)}, nil
 }
```
</details>
As per coding guidelines, `**/*_test.go`: Run tests with -race flag before committing — zero tolerance for race conditions.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
type integrationTaskSessionExecutor struct {
	mu      sync.Mutex
	started int
}

func (e *integrationTaskSessionExecutor) StartTaskSession(_ context.Context, _ taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started++
	return &taskpkg.SessionRef{SessionID: fmt.Sprintf("task-sess-%d", e.started)}, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 1185 - 1192,
The StartTaskSession method on integrationTaskSessionExecutor increments the
started counter without synchronization which can race under concurrent HTTP
tests; fix by making started a thread-safe counter (e.g., change started to an
int32/int64 and use atomic.AddInt32/AddInt64 in StartTaskSession, or add a
sync.Mutex on integrationTaskSessionExecutor and lock/unlock while
incrementing/reading started) so that StartTaskSession and the SessionID
generation are safe under concurrent calls.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `integrationTaskSessionExecutor.StartTaskSession` increments shared mutable state without synchronization even though the integration server can service concurrent requests under `-race`.
- Fix approach: make the counter increment and session-id generation thread-safe.

## Resolution

- Guarded the integration task-session executor counter with a mutex so the test harness is race-safe under concurrent request handling.
- Verified in the final `make verify` run.
