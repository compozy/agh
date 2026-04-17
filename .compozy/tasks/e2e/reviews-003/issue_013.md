---
status: resolved
file: internal/testutil/acpmock/driver_binary.go
line: 64
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMn,comment:PRRC_kwDOR5y4QM645avb
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Serialize the driver build behind a one-time initializer.**

The cache check happens before the build, so parallel callers can all miss the empty cache and each run `go build`. With the new parallel driver tests, that turns this helper into a build stampede instead of a cache.

<details>
<summary>♻️ Suggested direction</summary>

```diff
 var (
-	driverBinaryMu   sync.Mutex
-	driverBinaryPath string
+	driverBinaryOnce sync.Once
+	driverBinaryPath string
+	driverBinaryErr  error
 )

 func DefaultDriverPath() (string, error) {
-	driverBinaryMu.Lock()
-	cached := driverBinaryPath
-	driverBinaryMu.Unlock()
-	if strings.TrimSpace(cached) != "" {
-		return cached, nil
-	}
-
-	repoRoot, err := repoRootFromCaller()
-	if err != nil {
-		return "", err
-	}
-	...
-	driverBinaryMu.Lock()
-	driverBinaryPath = outputPath
-	driverBinaryMu.Unlock()
-	return outputPath, nil
+	driverBinaryOnce.Do(func() {
+		driverBinaryPath, driverBinaryErr = buildDriverBinary()
+	})
+	return driverBinaryPath, driverBinaryErr
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/driver_binary.go` around lines 23 - 64, The cache
check and build in DefaultDriverPath must be serialized to avoid concurrent go
build runs: move the driverBinaryMu.Lock() to cover the entire check-and-build
sequence (lock before reading driverBinaryPath and hold the lock until after
driverBinaryPath is set) so only one goroutine builds while others block and
then read the cached path; if the build fails, ensure you unlock and leave
driverBinaryPath empty so subsequent callers can retry (do not use sync.Once
because it prevents retries on failure). Use the existing symbols
DefaultDriverPath, driverBinaryMu, and driverBinaryPath to locate and implement
the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `DefaultDriverPath` releases `driverBinaryMu` before the cache miss is resolved, so parallel callers can all observe an empty cache and start separate `go build` subprocesses.
  - That is a real determinism and performance bug now that the driver tests run in parallel.
  - Implemented: serialized the cache-check/build/store path under `driverBinaryMu` while still leaving `driverBinaryPath` unset on failure so later callers can retry.
  - Regression coverage: added `TestDefaultDriverPathSharesConcurrentBuildResult` to require one shared cached path across concurrent callers.
  - Verification: `go test ./internal/testutil/acpmock -count=1`; `make verify`.
