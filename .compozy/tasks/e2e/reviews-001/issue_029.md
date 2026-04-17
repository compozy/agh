---
status: resolved
file: internal/testutil/e2e/runtime_harness_helpers_test.go
line: 1185
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcV,comment:PRRC_kwDOR5y4QM640q00
---

# Issue 029: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Check the file type and surrounding context
head -630 internal/testutil/e2e/runtime_harness_helpers_test.go | tail -20
```

Repository: compozy/agh

Length of output: 725

---

🏁 Script executed:

```shell
# Count occurrences of t.Fatalf in the handler functions
grep -n "t.Fatalf" internal/testutil/e2e/runtime_harness_helpers_test.go | head -20
```

Repository: compozy/agh

Length of output: 1223

---

🏁 Script executed:

```shell
# Check if there are tests using this newHarnessTestServer
grep -n "newHarnessTestServer" internal/testutil/e2e/runtime_harness_helpers_test.go
```

Repository: compozy/agh

Length of output: 155

---

🏁 Script executed:

```shell
# Look for the test function that references this (outside the 627-1185 range)
sed -n '1,626p' internal/testutil/e2e/runtime_harness_helpers_test.go | grep -A 5 "func Test"
```

Repository: compozy/agh

Length of output: 455

---

🏁 Script executed:

```shell
# Let's look at the specific handlers with t.Fatalf in the 627-1185 range
sed -n '627,1185p' internal/testutil/e2e/runtime_harness_helpers_test.go | grep -n "t.Fatalf" | head -15
```

Repository: compozy/agh

Length of output: 1010

---

🏁 Script executed:

```shell
# Let's check the exact context - look at a handler with t.Fatalf
sed -n '770,790p' internal/testutil/e2e/runtime_harness_helpers_test.go
```

Repository: compozy/agh

Length of output: 794

---

🌐 Web query:

`Go testing t.Fatalf called from goroutine http handler httptest unsafe`

💡 Result:

Calling t.Fatalf from a goroutine, such as one spawned by an HTTP handler under httptest, is unsafe and leads to undefined behavior like deadlocks, panics, or silent test failures. The official Go testing documentation explicitly states that t.Fatal, t.Fatalf, t.FailNow, and similar methods "must be called only from the goroutine running the Test function." In HTTP handler tests using httptest.NewServer or similar, the handler runs in a separate server goroutine, not the test goroutine. Any t.Fatalf called from there violates this rule. Safe alternatives: - Use t.Error/t.Errorf from goroutines (allowed, marks test failed but doesn't stop execution). - Communicate errors via channels: have the handler goroutine send errors to a channel that the test goroutine reads and checks with t.Fatal if needed. - Use sync.WaitGroup to wait for background work, then assert state or channel contents in the test goroutine. - For Go 1.25+, consider testing/synctest for concurrent code, but avoid real network for full isolation. Example safe pattern: func TestHandler(t *testing.T) { errc := make(chan error, 1) ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { go func { // background work if someError != nil { errc <- someError return } // success close(errc) } w.WriteHeader(http.StatusOK) })) defer ts.Close resp, err := http.Get(ts.URL) if err != nil { t.Fatal(err) } defer resp.Body.Close select { case err := <-errc: t.Fatalf("background error: %v", err) case <-time.After(5 * time.Second): t.Fatal("timeout waiting for background work") default: // success if channel closed without error } } Or use t.Error in goroutine and check t.Failed after waiting. This is a common gotcha discussed in Go issue `#15758` and linters like DeepSource GO-W6004.

Citations:

- 1: https://pkg.go.dev/testing
- 2: https://deepsource.com/directory/go/issues/GO-W6004
- 3: https://github.com/golang/go/issues/15758
- 4: https://github.com/golang/mock/issues/346

---



</details>

**Remove `t.Fatalf` calls from HTTP handler goroutines — they violate Go's testing contract.**

Handlers registered with `mux.HandleFunc` execute on server goroutines, not the test goroutine. Calling `t.Fatalf` or `t.FailNow` from those goroutines causes undefined behavior: deadlocks, panics, or silent test failures (see Go issue `#15758`, linter GO-W6004).

Replace with one of:
- **Channels**: Have handlers send validation errors to a channel; the test goroutine reads and asserts.
- **t.Error**: Use non-fatal assertions (allowed in goroutines), then check `t.Failed()` after test cleanup.
- **Return HTTP error status**: Let the test client receive and validate the error response.

Examples affected (14+ instances): `/api/network/peers`, `/api/automation/runs`, `/api/tasks`, `/api/extensions`, `/api/bridges` handlers validating query parameters and request bodies.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/runtime_harness_helpers_test.go` around lines 627 -
1185, The HTTP handlers inside newHarnessTestServer call t.Fatalf from server
goroutines (e.g. handlers for "/api/network/peers", "/api/automation/runs",
"/api/tasks", "/api/extensions", "/api/bridges" and others) which violates Go's
testing contract; remove all t.Fatalf/t.FailNow calls and instead propagate
handler validation failures to the test goroutine — e.g. create an error channel
(errCh := make(chan error, N)) in newHarnessTestServer, replace each t.Fatalf
call with errCh <- fmt.Errorf("...") (or call t.Error and also send a marker),
and ensure the test that calls newHarnessTestServer reads from errCh (or checks
t.Failed()) and fails appropriately; keep all handler logic otherwise but never
call t.Fatalf/t.FailNow from inside mux.HandleFunc closures.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `newHarnessTestServer` calls `t.Fatalf` from HTTP handlers, which run on
  server goroutines instead of the test goroutine. That violates Go's testing
  contract and can hide failures or deadlock the test. Handler validation
  errors need to be reported back to the test goroutine and asserted there.

## Resolution

- Replaced handler-goroutine `t.Fatalf` paths with cleanup-checked error
  reporting so validation failures are asserted from the test goroutine.
