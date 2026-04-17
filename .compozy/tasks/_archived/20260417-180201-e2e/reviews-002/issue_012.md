---
status: resolved
file: internal/testutil/e2e/config_seed.go
line: 157
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10_,comment:PRRC_kwDOR5y4QM644c9K
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Do not ignore filesystem errors with `_`.**

Both error drops hide real IO issues during test setup/cleanup. Handle them explicitly (or log a clear justification in cleanup paths).

<details>
<summary>🔧 Suggested fix</summary>

```diff
 import (
+	"errors"
 	"fmt"
 	"maps"
 	"os"
@@
 	if err := toml.NewEncoder(file).Encode(overlay); err != nil {
-		_ = file.Close()
+		if closeErr := file.Close(); closeErr != nil {
+			t.Fatalf("toml encode config %q error = %v (close error: %v)", homePaths.ConfigFile, err, closeErr)
+		}
 		t.Fatalf("toml encode config %q error = %v", homePaths.ConfigFile, err)
 	}
@@
 	t.Cleanup(func() {
-		_ = os.Remove(path)
+		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
+			t.Logf("cleanup socket %q error: %v", path, err)
+		}
 	})
```
</details>


As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".


Also applies to: 340-342

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/e2e/config_seed.go` at line 157, Replace the ignored
file.Close() calls (e.g., "_ = file.Close()") with explicit error handling:
capture the return value (if err := file.Close(); err != nil { ... }) and either
fail the test (t.Fatalf/t.Fatalf with context) or log the error
(t.Logf/processLogger.Error) with a clear justification in cleanup paths; apply
the same change to the other occurrences around lines 340-342 so no filesystem
error is silently dropped and the handling matches the surrounding function's
error reporting style.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `SeedConfig()` and `shortSocketPath()` discard filesystem errors with `_`, which violates the workspace rule against silent error drops and hides rare but actionable test-harness failures.
- Fix plan: handle `file.Close()` explicitly on encode failure and log non-`ENOENT` cleanup errors when removing short-lived socket paths.
- Test impact: the normal success path is already covered by `internal/testutil/e2e/config_seed_test.go`; a small supporting test update there is appropriate because the new failure branches are not directly injectable with the current API.
- Resolution: implemented. `SeedConfig()` now handles close failures explicitly on encode errors, `shortSocketPath()` logs non-`ENOENT` cleanup failures, and `config_seed_test.go` was extended to exercise the helper path shape and early-removal cleanup path.
- Verification: `go test ./internal/testutil/e2e`, `make verify`.
