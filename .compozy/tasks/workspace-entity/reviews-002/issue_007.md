---
status: resolved
file: internal/cli/session.go
line: 568
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoB5,comment:PRRC_kwDOR5y4QM61T6HK
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject relative `--cwd` values before building the request.**

`resolveSessionCreateWorkspace` forwards `trimmedCWD` as-is. `agh session new --cwd .` or `../repo` will therefore send a relative workspace path that the daemon can resolve against its own working directory or reject outright.


<details>
<summary>Suggested fix</summary>

```diff
 import (
 	"encoding/json"
 	"errors"
 	"fmt"
+	"path/filepath"
 	"strconv"
 	"strings"
 	"time"
 )
...
 	switch {
 	case trimmedWorkspace != "" && trimmedCWD != "":
 		return "", "", errors.New("cli: --workspace and --cwd are mutually exclusive")
 	case trimmedWorkspace != "":
 		return trimmedWorkspace, "", nil
 	case trimmedCWD != "":
-		return "", trimmedCWD, nil
+		if !filepath.IsAbs(trimmedCWD) {
+			return "", "", fmt.Errorf("cli: --cwd must be an absolute path: %q", trimmedCWD)
+		}
+		return "", filepath.Clean(trimmedCWD), nil
 	default:
 		workspacePath, err := currentWorkingDirectory(deps)
 		if err != nil {
 			return "", "", err
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/session.go` around lines 551 - 568, In
resolveSessionCreateWorkspace, reject relative --cwd values by checking
trimmedCWD with filepath.IsAbs (or equivalent) and returning an error when it's
not absolute; specifically, when trimmedCWD != "" validate
filepath.IsAbs(trimmedCWD) and return an error like "cli: --cwd must be an
absolute path" instead of forwarding trimmedCWD, otherwise proceed to return "",
trimmedCWD, nil; keep currentWorkingDirectory usage unchanged for the default
branch.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `resolveSessionCreateWorkspace` currently forwards `--cwd` as-is.
  - That lets `agh session new --cwd .` or `--cwd ../repo` reach the daemon with a relative path, where resolution can depend on the daemon process working directory instead of the CLI caller’s directory.
  - I will reject relative `--cwd` values in the CLI and add test coverage for the validation.
