---
status: resolved
file: cmd/agh-codegen/main.go
line: 15
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q885,comment:PRRC_kwDOR5y4QM6200jL
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
head -50 cmd/agh-codegen/main.go | cat -n
```

Repository: compozy/agh

Length of output: 1475

---



</details>

**Also cancel on `SIGTERM`.**

Right now only Ctrl-C cancels `ctx`. In CI/container shutdowns that send `SIGTERM`, the formatter subprocess will miss the graceful cancellation path you just added.

<details>
<summary>♻️ Suggested change</summary>

```diff
 import (
 	"bytes"
 	"context"
 	"encoding/json"
 	"errors"
 	"fmt"
 	"log/slog"
 	"os"
 	"os/exec"
 	"os/signal"
 	"path/filepath"
 	"reflect"
 	"strings"
+	"syscall"
 
 	"github.com/pedronauck/agh/internal/api/spec"
 	"github.com/pedronauck/agh/internal/codegen/sdkts"
 )
@@
-	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
+	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@cmd/agh-codegen/main.go` around lines 3 - 15, The signal handling only
cancels on SIGINT (Ctrl-C); update the shutdown logic to also handle SIGTERM so
the formatter subprocess gets the graceful cancellation path. Modify the signal
setup around the signal.Notify/signal handling (the code that creates ctx/cancel
or the signal channel) to include syscall.SIGTERM (or switch to
signal.NotifyContext with both syscall.SIGINT and syscall.SIGTERM), and add
syscall to the imports; ensure the existing ctx cancellation/cancel() is invoked
when SIGTERM arrives so functions like the formatter that read ctx are notified.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `main()` currently creates its shutdown context with `signal.NotifyContext(..., os.Interrupt)` only, so `SIGTERM` does not follow the same cancellation path.
- That means container and CI shutdowns can bypass the graceful context cancellation already used by the formatter subprocess path.
- Fix approach: extend the shutdown signal set to include `syscall.SIGTERM` and add a regression test that locks the configured shutdown signals.
