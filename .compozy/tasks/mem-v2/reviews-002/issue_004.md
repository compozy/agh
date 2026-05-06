---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1283
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cOm,comment:PRRC_kwDOR5y4QM6-Uf9t
---

# Issue 004: _⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_

**Add the missing `memcontract` import.**

`memcontract.ScopeWorkspace` is referenced here, but this file never imports that package, so the test file does not compile.


<details>
<summary>💡 Minimal fix</summary>

```diff
 import (
 	"bufio"
 	"context"
 	"encoding/json"
 	"errors"
 	"fmt"
 	"io"
 	"net/http"
 	"os"
 	"path/filepath"
 	"strconv"
 	"strings"
 	"sync"
 	"testing"
 	"time"
 
 	"github.com/pedronauck/agh/internal/acp"
 	"github.com/pedronauck/agh/internal/api/contract"
 	core "github.com/pedronauck/agh/internal/api/core"
 	automationpkg "github.com/pedronauck/agh/internal/automation"
 	bridgepkg "github.com/pedronauck/agh/internal/bridges"
 	aghconfig "github.com/pedronauck/agh/internal/config"
 	"github.com/pedronauck/agh/internal/memory"
+	memcontract "github.com/pedronauck/agh/internal/memory/contract"
 	"github.com/pedronauck/agh/internal/observe"
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	"github.com/pedronauck/agh/internal/observe"
)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 1282 - 1283,
The test references memcontract.ScopeWorkspace but the package is not imported;
add an import for the memcontract package in
internal/api/httpapi/httpapi_integration_test.go (so memcontract.ScopeWorkspace
can be resolved) — update the import block that contains the test functions
(where runtime.workspace and payload.Dream are used) to include the memcontract
package name.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestHTTPMemoryDreamTriggerIntegration` references `memcontract.ScopeWorkspace`, but the integration test file does not import `internal/memory/contract`.
- Evidence: the import block currently includes `internal/memory` but not the `memcontract` alias used later in the file.
- Fix plan: add the missing import and keep the existing assertion compiling under the integration build tag.
- Resolution: added the missing `memcontract` import in `internal/api/httpapi/httpapi_integration_test.go`.
- Verification: targeted integration-tagged `go test` for `internal/api/httpapi` passed, and fresh `make verify` passed on 2026-05-06.
