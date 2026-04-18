---
status: resolved
file: internal/api/core/tasks_surface_test.go
line: 733
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lbx,comment:PRRC_kwDOR5y4QM65B8fO
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
head -n 750 internal/api/core/tasks_surface_test.go | tail -n 30
```

Repository: compozy/agh

Length of output: 1256

---

🏁 Script executed:

```shell
# Search for the integration test mentioned in the PR
rg "text/event-stream" --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 7319

---

🏁 Script executed:

```shell
# Check if there are other Content-Type assertions in the codebase
rg "Content-Type" --type go -B 1 -A 1 | head -100
```

Repository: compozy/agh

Length of output: 7056

---

🏁 Script executed:

```shell
# Check the SSE preparation in sse.go to see what it sets
cat -n internal/api/core/sse.go | head -30
```

Repository: compozy/agh

Length of output: 945

---

🏁 Script executed:

```shell
# Look at what handlers_test.go tests and see if it's similar to tasks_surface_test.go
rg -A 10 "Content-Type.*text/event-stream" internal/api/core/tasks_surface_test.go
```

Repository: compozy/agh

Length of output: 538

---

🏁 Script executed:

```shell
# Check if there's any test that validates charset is added
rg "charset" --type go -i -B 2 -A 2 | grep -A 5 -B 5 "text/event-stream"
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at the integration test to understand the context
rg -B 5 "strings.HasPrefix.*text/event-stream" internal/api/udsapi/udsapi_integration_test.go
```

Repository: compozy/agh

Length of output: 378

---

🏁 Script executed:

```shell
# Check if there's any HTTP middleware or wrapper that adds charset
rg "charset" --type go -B 3 -A 3 | head -80
```

Repository: compozy/agh

Length of output: 4764

---

🏁 Script executed:

```shell
# Look at what the actual HTTP response handling does
rg "Header.*Set" internal/api/core/sse.go -A 1
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check current imports in tasks_surface_test.go
head -n 20 internal/api/core/tasks_surface_test.go
```

Repository: compozy/agh

Length of output: 574

---

🏁 Script executed:

```shell
# Verify strings is already imported
rg "^import" -A 20 internal/api/core/tasks_surface_test.go | head -30
```

Repository: compozy/agh

Length of output: 624

---

🏁 Script executed:

```shell
# Check if other similar tests in the same file have Content-Type checks
rg -n "Content-Type" internal/api/core/tasks_surface_test.go
```

Repository: compozy/agh

Length of output: 140

---

🏁 Script executed:

```shell
# Check what the difference is between unit test using http.ResponseRecorder vs integration test using actual HTTP
rg "type.*Recorder\|httptest" internal/api/core/tasks_surface_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Understand the integration test setup
rg -B 10 "udsapi_integration_test.go.*Content-Type" internal/api/udsapi/udsapi_integration_test.go | head -40
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there's any HTTP middleware adding charset to responses
rg -i "middleware\|handler" internal/api/core/ --type go | grep -i "type\|func" | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there's any framework or middleware configuration that adds charset
rg -i "charset" internal/api/core/sse.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at how the integration test gets the response (what HTTP client does it use)
rg -B 20 "udsapi_integration_test.go" internal/api/udsapi/udsapi_integration_test.go | grep -A 10 "http\." | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Final check: See if there are other Content-Type checks that already use prefix matching
rg "HasPrefix.*Content-Type\|Content-Type.*HasPrefix" --type go
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Use prefix matching to accept valid SSE `Content-Type` parameters.**

The exact match for `"text/event-stream"` doesn't account for responses like `"text/event-stream; charset=utf-8"`, which is valid per the SSE specification. The integration tests (`udsapi_integration_test.go`, `httpapi_integration_test.go`) already use prefix matching for this check; the unit test should align with that pattern.

Add `"strings"` to the imports and use `strings.HasPrefix(got, "text/event-stream")` instead of the exact equality check.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_test.go` around lines 731 - 733, Replace the
exact Content-Type equality check with a prefix match to accept parameters:
import the "strings" package and change the test assertion that inspects
streamResp.Header().Get("Content-Type") to use strings.HasPrefix(got,
"text/event-stream") (and adjust the failure message accordingly) so responses
like "text/event-stream; charset=utf-8" are accepted.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Confirmed. The unit test currently requires an exact `Content-Type` match of `text/event-stream`, but valid SSE responses may include parameters such as `charset=utf-8`. The repo’s integration tests already use prefix matching, so I’ll align these tests to the production-tolerant check.
