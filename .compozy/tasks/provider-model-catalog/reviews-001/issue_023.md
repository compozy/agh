---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/modelcatalog/service_integration_test.go
line: 21
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tF,comment:PRRC_kwDOR5y4QM6-6bs1
---

# Issue 023: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Missing `//go:build integration` tag.**

As per coding guidelines, integration test files should use the `//go:build integration` build tag to conditionally compile them. This file is named `*_integration_test.go` but lacks the build constraint.



<details>
<summary>🔧 Proposed fix</summary>

```diff
+//go:build integration
+
 package modelcatalog_test
 
 import (
```
</details>

As per coding guidelines: "Use `//go:build integration` tag and `//go:build !integration` for conditional integration and unit test compilation".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
//go:build integration

package modelcatalog_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	_ "modernc.org/sqlite"
)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/service_integration_test.go` around lines 1 - 21, This
integration test file is missing the Go build constraint; add a top-of-file
build tag line "//go:build integration" (followed by a blank line) before the
"package modelcatalog_test" declaration so the file only compiles with the
integration build tag; ensure you use the modern "//go:build integration" form
(optionally also add the old-style "+build integration" comment if your repo
requires legacy tags) to match the project's build/tagging conventions.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/modelcatalog/service_integration_test.go` is an integration test file by name but currently has no `//go:build integration` constraint.
  - That violates the repo test-tag convention and causes the file to compile in the default unit-test build.
  - Fix approach: add the build tag at the top of the file and keep the rest of the test unchanged.
  - Resolved in `internal/modelcatalog/service_integration_test.go`; verified with focused package tests and full `make verify`.
