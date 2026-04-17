---
status: resolved
file: internal/daemon/daemon_network_collaboration_integration_test.go
line: 738
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEba,comment:PRRC_kwDOR5y4QM640qzt
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cd /tmp && find . -name "daemon_network_collaboration_integration_test.go" -type f
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# First, let's locate the file
find / -name "daemon_network_collaboration_integration_test.go" 2>/dev/null | head -5
```

Repository: compozy/agh

Length of output: 138

---

🏁 Script executed:

```shell
# Try searching in common Go project locations
find . -path "*/internal/daemon/*integration_test.go" 2>/dev/null | head -10
```

Repository: compozy/agh

Length of output: 654

---

🏁 Script executed:

```shell
wc -l ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 125

---

🏁 Script executed:

```shell
# Read the specific sections mentioned in the review
sed -n '724,738p' ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 512

---

🏁 Script executed:

```shell
# Check lines 754-788
sed -n '754,788p' ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 990

---

🏁 Script executed:

```shell
# Check lines 790-813
sed -n '790,813p' ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 572

---

🏁 Script executed:

```shell
# Check the time.Sleep issue at lines 856-872
sed -n '856,872p' ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 356

---

🏁 Script executed:

```shell
# Check if there are URL encoding utilities already in the codebase
rg "url.QueryEscape|url.PathEscape" ./internal --type go
```

Repository: compozy/agh

Length of output: 7431

---

🏁 Script executed:

```shell
# Look for the e2etest.RuntimeHarness to understand how HTTPJSON works
rg "type RuntimeHarness" --type go
```

Repository: compozy/agh

Length of output: 201

---

🏁 Script executed:

```shell
# Also check if other tests in the codebase use proper URL encoding
rg "QueryEscape|PathEscape" . --type go | head -20
```

Repository: compozy/agh

Length of output: 2383

---

🏁 Script executed:

```shell
# Check if there's an encodeQuery helper or similar in the e2e test utilities
grep -n "encodeQuery\|QueryEscape" ./internal/testutil/e2e/*.go
```

Repository: compozy/agh

Length of output: 1436

---

🏁 Script executed:

```shell
# Verify the guidelines explicitly prohibit unescaped URL construction
rg "never.*hardcode|Never.*string.*concat" --type go | head -10
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Find the encodeQuery function implementation
sed -n '1287,1295p' ./internal/testutil/e2e/runtime_harness.go
```

Repository: compozy/agh

Length of output: 268

---

🏁 Script executed:

```shell
# Check what the file imports to see what utilities are available
head -30 ./internal/daemon/daemon_network_collaboration_integration_test.go
```

Repository: compozy/agh

Length of output: 869

---

🏁 Script executed:

```shell
# Double-check if encodeQuery is exported or needs to be imported
grep -n "func encodeQuery" ./internal/testutil/e2e/runtime_harness.go
```

Repository: compozy/agh

Length of output: 104

---



</details>

**Use `url.PathEscape()` and `url.QueryEscape()` for channel parameters in network API helpers.**

These helpers construct URLs by directly concatenating the `channel` parameter. A channel name with reserved URL characters (e.g., `?`, `#`, `/`, `&`) will corrupt the URL structure instead of being encoded as a safe parameter value, making the helpers unreliable for edge cases.

**Lines to fix:**
- **Line 731** (`mustHTTPNetworkPeersMaybe`): Wrap `trimmed` with `url.QueryEscape()` when building the query string.
- **Line 763** (`mustHTTPNetworkChannel`): Wrap `channel` with `url.PathEscape()` in the path.
- **Line 805** (`channelHasMessageID`): Wrap `channel` with `url.PathEscape()` in the path.

The codebase already establishes this pattern throughout (e.g., `runtime_harness.go`, `cli/client.go`), so align these helpers with the existing standard.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_network_collaboration_integration_test.go` around
lines 724 - 738, The helpers build URLs by concatenating raw channel names;
update mustHTTPNetworkPeersMaybe to wrap the trimmed channel with
url.QueryEscape when appending "?channel=" + trimmed, and update
mustHTTPNetworkChannel and channelHasMessageID to wrap channel with
url.PathEscape when inserting it into the path; locate the uses of
trimmed/channel in those functions and replace concatenation with the
corresponding url escaping call to ensure reserved characters are encoded.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the HTTP network helpers interpolate raw channel names into query strings and URL paths, so reserved characters can corrupt the request instead of being treated as data.
- Fix plan: use `url.QueryEscape` for the optional peers query and `url.PathEscape` for all channel path segments, including the channel-messages helper that shares the same bug pattern.
- Resolution: escaped the peers query parameter and all channel path segments in the network HTTP helper functions.
- Verification: `go test ./internal/daemon` passed. `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon` was rerun but is blocked before these tests execute because the branch is missing `internal/testutil/acpmock/driver/dist/index.js`. `make verify` hits the same unrelated blocker in `internal/testutil/acpmock` and `internal/testutil/e2e`.
