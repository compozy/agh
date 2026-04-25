---
status: resolved
file: internal/observe/health.go
line: 239
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLig,comment:PRRC_kwDOR5y4QM67SmDn
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Sort failure rows before trimming `Recent`.**

`health.Recent` currently keeps the first 10 failures returned by `ListSessions`, not the 10 newest by `UpdatedAt`. If the registry returns insertion or ID order, newer failures can be omitted or appear after older ones, so the health payload's "recent" section becomes misleading.

<details>
<summary>💡 Suggested fix</summary>

```diff
 import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
+	"sort"
	"strings"
	"time"
@@
	health := FailureHealth{
		Status: observeHealthStatusOK,
		ByKind: make(map[store.FailureKind]int),
-		Recent: make([]SessionFailureHealth, 0),
	}
+	recent := make([]SessionFailureHealth, 0, len(sessions))
	for _, info := range sessions {
@@
-		if len(health.Recent) < 10 {
-			health.Recent = append(health.Recent, SessionFailureHealth{
-				SessionID:       strings.TrimSpace(info.ID),
-				AgentName:       strings.TrimSpace(info.AgentName),
-				Provider:        strings.TrimSpace(info.Provider),
-				WorkspaceID:     strings.TrimSpace(info.WorkspaceID),
-				State:           strings.TrimSpace(info.State),
-				FailureKind:     failure.Kind,
-				Summary:         diagnostics.RedactAndBound(failure.Summary, maxFailureHealthBytes),
-				CrashBundlePath: diagnostics.RedactAndBound(failure.CrashBundlePath, maxFailureHealthBytes),
-				UpdatedAt:       info.UpdatedAt,
-			})
-		}
+		recent = append(recent, SessionFailureHealth{
+			SessionID:       strings.TrimSpace(info.ID),
+			AgentName:       strings.TrimSpace(info.AgentName),
+			Provider:        strings.TrimSpace(info.Provider),
+			WorkspaceID:     strings.TrimSpace(info.WorkspaceID),
+			State:           strings.TrimSpace(info.State),
+			FailureKind:     failure.Kind,
+			Summary:         diagnostics.RedactAndBound(failure.Summary, maxFailureHealthBytes),
+			CrashBundlePath: diagnostics.RedactAndBound(failure.CrashBundlePath, maxFailureHealthBytes),
+			UpdatedAt:       info.UpdatedAt,
+		})
	}
	if health.Total == 0 {
		health.ByKind = nil
		health.Recent = nil
	} else {
		health.Status = observeHealthStatusDegraded
+		sort.SliceStable(recent, func(i, j int) bool {
+			return recent[i].UpdatedAt.After(recent[j].UpdatedAt)
+		})
+		if len(recent) > 10 {
+			recent = recent[:10]
+		}
+		health.Recent = recent
	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/health.go` around lines 220 - 239, health.Recent is keeping
the first 10 entries returned by ListSessions instead of the 10 newest by
UpdatedAt, so sort health.Recent by UpdatedAt descending before you trim it:
after populating health.Recent (but before the len check/trim) use sort.Slice to
order entries by UpdatedAt (newest first), then slice to the first 10 entries;
ensure you still preserve the redaction fields and the existing
Total/ByKind/Status logic (referencing health.Recent, UpdatedAt, and the
ListSessions population loop) so the "recent" section reliably contains the
newest failures.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `collectFailureHealth` appends the first ten failed sessions returned by the registry and never sorts by `UpdatedAt`, so the `Recent` payload depends on registry order rather than recency.
- Fix approach: collect all failure rows, sort by `UpdatedAt` descending, and trim to the ten newest entries after sorting.
