---
status: resolved
file: internal/testutil/e2e/mock_agents.go
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcN,comment:PRRC_kwDOR5y4QM640q0s
---

# Issue 027: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's look at the file and the specific lines mentioned
cat -n internal/testutil/e2e/mock_agents.go | sed -n '80,130p'
```

Repository: compozy/agh

Length of output: 1879

---

🏁 Script executed:

```shell
# Check the full context around those lines and understand what sessionID is
cat -n internal/testutil/e2e/mock_agents.go | sed -n '1,50p'
```

Repository: compozy/agh

Length of output: 1762

---

🏁 Script executed:

```shell
# Search for how sessionID is generated or used elsewhere in the codebase
rg "sessionID" -A 2 -B 2 --max-count=20
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
# Search for sessionID generation and validation
rg "newSessionID|IDGenerator" -A 5 -B 5 internal/session/
```

Repository: compozy/agh

Length of output: 10549

---

🏁 Script executed:

```shell
# Check if URL encoding is used elsewhere in the codebase
rg "url\.PathEscape|net/url" --max-count=15
```

Repository: compozy/agh

Length of output: 7211

---

🏁 Script executed:

```shell
# Look at HTTPURL function definition
rg "func.*HTTPURL" -A 10 internal/testutil/e2e/
```

Repository: compozy/agh

Length of output: 898

---

🏁 Script executed:

```shell
# Check how other parts of codebase build session URLs
rg "/api/sessions/" -B 2 -A 2 internal/ | head -60
```

Repository: compozy/agh

Length of output: 4374

---



</details>

**Use `url.PathEscape()` when splicing `sessionID` into API paths.**

Both helpers concatenate the session ID directly into the path string. This is inconsistent with how the rest of the codebase handles similar path parameters—the CLI client, HTTP API transport tests, and other test utilities all use `url.PathEscape()` to safely handle IDs. Apply it here too:

<details>
<summary>Lines 88–94</summary>

```go
h.HTTPURL("/api/sessions/"+url.PathEscape(sessionID)+"/prompt")
```
</details>

<details>
<summary>Lines 117–123</summary>

```go
h.HTTPURL("/api/sessions/"+url.PathEscape(sessionID)+"/approve")
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/mock_agents.go` around lines 88 - 94, The path
construction currently concatenates sessionID directly into the URL when calling
h.HTTPURL in the calls around doRequest; update those calls (the ones that build
"/api/sessions/"+sessionID+"/prompt" and "/api/sessions/"+sessionID+"/approve")
to wrap sessionID with url.PathEscape(sessionID) before passing to h.HTTPURL so
IDs are safely escaped; keep the surrounding callsites (doRequest, h.HTTPClient,
http.MethodPost/Put) unchanged and import net/url if not already present.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The helpers splice `sessionID` directly into HTTP paths. If a session ID ever
  contains path-sensitive bytes, the test would hit the wrong route. The rest
  of the codebase already escapes path parameters, so these helpers should do
  the same with `url.PathEscape`.

## Resolution

- Session-targeted helper URLs now escape session IDs before embedding them in
  HTTP paths, and the regression test covers IDs containing slashes and spaces.
