---
status: resolved
file: internal/daemon/daemon_integration_test.go
line: 3377
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uT_,comment:PRRC_kwDOR5y4QM65IlPA
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify unwrapped returns in this helper.
rg -n -C3 'func ensureDetachedHarnessWorkspaceIndex|return err|return db.InsertWorkspace' internal/daemon/daemon_integration_test.go
```

Repository: compozy/agh

Length of output: 2160

---



</details>

**Wrap error returns with context in `ensureDetachedHarnessWorkspaceIndex`.**

Per coding guidelines, wrap all error returns with `fmt.Errorf("context: %w", err)`. The function currently returns unwrapped errors from `GetWorkspace`, `MkdirAll`, and `InsertWorkspace`.

<details>
<summary>♻️ Proposed fix</summary>

```diff
 func ensureDetachedHarnessWorkspaceIndex(
 	t *testing.T,
 	db *globaldb.GlobalDB,
 	homePaths aghconfig.HomePaths,
 	workspaceID string,
 	workspaceRoot string,
 ) error {
 	t.Helper()

 	if _, err := db.GetWorkspace(testutil.Context(t), workspaceID); err == nil {
 		return nil
 	} else if !errors.Is(err, workspacepkg.ErrWorkspaceNotFound) {
-		return err
+		return fmt.Errorf("get workspace %q: %w", workspaceID, err)
 	}

 	rootDir := strings.TrimSpace(workspaceRoot)
 	if rootDir == "" {
 		rootDir = filepath.Join(homePaths.HomeDir, workspaceID)
 	}
 	if err := os.MkdirAll(rootDir, 0o755); err != nil {
-		return err
+		return fmt.Errorf("mkdir workspace root %q: %w", rootDir, err)
 	}
-	return db.InsertWorkspace(testutil.Context(t), workspacepkg.Workspace{
+	if err := db.InsertWorkspace(testutil.Context(t), workspacepkg.Workspace{
 		ID:        workspaceID,
 		Name:      workspaceID,
 		RootDir:   rootDir,
 		CreatedAt: time.Now().UTC(),
 		UpdatedAt: time.Now().UTC(),
-	})
+	}); err != nil {
+		return fmt.Errorf("insert workspace %q: %w", workspaceID, err)
+	}
+	return nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if _, err := db.GetWorkspace(testutil.Context(t), workspaceID); err == nil {
		return nil
	} else if !errors.Is(err, workspacepkg.ErrWorkspaceNotFound) {
		return fmt.Errorf("get workspace %q: %w", workspaceID, err)
	}

	rootDir := strings.TrimSpace(workspaceRoot)
	if rootDir == "" {
		rootDir = filepath.Join(homePaths.HomeDir, workspaceID)
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return fmt.Errorf("mkdir workspace root %q: %w", rootDir, err)
	}
	if err := db.InsertWorkspace(testutil.Context(t), workspacepkg.Workspace{
		ID:        workspaceID,
		Name:      workspaceID,
		RootDir:   rootDir,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("insert workspace %q: %w", workspaceID, err)
	}
	return nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_integration_test.go` around lines 3358 - 3377, The
function ensureDetachedHarnessWorkspaceIndex returns raw errors from
db.GetWorkspace, os.MkdirAll, and db.InsertWorkspace; update each return to wrap
the underlying error with fmt.Errorf including context (e.g.,
fmt.Errorf("GetWorkspace failed: %w", err), fmt.Errorf("mkdir workspace root %s:
%w", rootDir, err), fmt.Errorf("InsertWorkspace failed: %w", err)) while
preserving the existing workspacepkg.ErrWorkspaceNotFound check and keeping the
same calls to db.GetWorkspace(testutil.Context(t), workspaceID) and
db.InsertWorkspace(...).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `ensureDetachedHarnessWorkspaceIndex` returned raw `GetWorkspace`, `MkdirAll`, and `InsertWorkspace` errors. That made failures harder to attribute and violated the workspace rule to wrap operational errors with context. I wrapped each error path with the failing operation while preserving the existing `ErrWorkspaceNotFound` branch semantics. Verified with `go test -tags integration ./internal/daemon -run TestDetachedHarnessIntegration -count=1` and `make verify`.
