---
status: resolved
file: internal/mcp/auth/service_test.go
line: 84
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLib,comment:PRRC_kwDOR5y4QM67SmDg
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`Can Go's `testing.T.Fatal` / `testing.T.FailNow` be called safely from goroutines started by a test, such as `httptest` handlers?`

💡 Result:

No, Go's testing.T.Fatal and testing.T.FailNow cannot be called safely from goroutines started by a test, including httptest handlers. The official documentation explicitly states that these methods "must be called only from the goroutine running the Test function" and "FailNow must be called from the goroutine running the test or benchmark function, not from other goroutines created during the test." Calling them from other goroutines, such as those spawned by httptest.NewServer handlers, leads to data races (detectable by the race detector), unreliable test termination (only the calling goroutine exits via runtime.Goexit, not the test goroutine), and potential silent failures or deadlocks. Instead, use t.Error from goroutines (which is safe) combined with proper synchronization (e.g., channels or t.Cleanup), or use a shared error channel to signal failures back to the test goroutine.

Citations:

- 1: https://pkg.go.dev/testing/
- 2: https://pkg.go.dev/testing@go1.25.6
- 3: https://pkg.go.dev/testing@go1.26rc1
- 4: https://go.dev/src/testing/testing.go?s=16299%3A16316
- 5: https://github.com/golang/go/blob/go1.21.5/src/testing/testing.go
- 6: https://github.com/golang/go/issues/15758
- 7: https://github.com/golang/go/issues/24678
- 8: https://github.com/dominikh/go-tools/issues/157
- 9: https://deepsource.com/directory/go/issues/GO-W6004

---

🏁 Script executed:

```shell
# Check if the file exists and examine the relevant sections
if [ -f "internal/mcp/auth/service_test.go" ]; then
  echo "=== File exists, checking line count ==="
  wc -l internal/mcp/auth/service_test.go

  echo -e "\n=== Lines 27-84 ==="
  sed -n '27,84p' internal/mcp/auth/service_test.go

  echo -e "\n=== Lines 174-185 ==="
  sed -n '174,185p' internal/mcp/auth/service_test.go

  echo -e "\n=== Lines 273-278 ==="
  sed -n '273,278p' internal/mcp/auth/service_test.go

  echo -e "\n=== Looking for writeJSON helper definition ==="
  grep -n "func writeJSON" internal/mcp/auth/service_test.go
else
  echo "File not found at internal/mcp/auth/service_test.go"
  # Search for it
  find . -name "service_test.go" -path "*/auth/*"
fi
```

Repository: compozy/agh

Length of output: 3005

---



</details>

**Avoid `t.Fatal*` inside `httptest` handlers and helper functions called from them.**

These handlers run on server goroutines, so `t.Fatalf()` only aborts that goroutine, not the test. This causes data races (caught by the race detector), unreliable test termination, and potential deadlocks. The Go testing documentation explicitly requires these methods to be called only from the test goroutine. Capture handler failures in a channel/mutex and fail from the main test goroutine instead.

This applies to direct `t.Fatalf()` calls at lines 39, 42, 47, 50, 61, 69, 74, and also to `writeJSON()` (lines 273–278), which is called from handlers at lines 174–185 and lines 27–84.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/mcp/auth/service_test.go` around lines 27 - 84, The test's httptest
handler (switch on metadataWellKnownPath, "/token", "/revoke") and helper
writeJSON must not call t.Fatalf/t.Fatal from handler goroutines; instead create
an error channel (e.g., errCh := make(chan error, 1) or an errors slice
protected by mu) and have handler code replace each t.Fatalf/t.Fatal and
writeJSON-internal test failures with sending errors to errCh (or return an
error value from writeJSON). In writeJSON change its signature to return error
instead of calling t.Fatal*, and have handlers check that error and send it to
errCh; in the main test goroutine select/receive from errCh after exercising the
server and call t.Fatalf/t.Fatal there if any handler-reported error exists;
also stop using t.Fatal* inside ParseForm error branches and instead send those
errors to errCh. This keeps refreshCalled/revokedToken/mu logic intact but
ensures all test failures are raised from the main test goroutine.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `httptest` handlers in `service_test.go` call `t.Fatal*` directly and through `writeJSON`; those handlers run in server goroutines, so `FailNow` does not stop the test goroutine reliably.
- Fix approach: replace handler `t.Fatal*` calls with a synchronized handler-error recorder, make `writeJSON` return errors, and assert recorded handler errors from the test goroutine after the flow.
