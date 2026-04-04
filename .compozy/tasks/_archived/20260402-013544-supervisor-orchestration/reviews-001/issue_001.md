---
status: resolved
file: internal/kernel/prompt_catalog.go
line: 28
severity: medium
author: claude-code
---

# Issue 001: Inconsistent graceful degradation in buildPromptCatalogsForAgent



## Review Comment

`buildPromptCatalogsForAgent` applies two different error-handling strategies for additive catalog metadata:

- **`config.LoadPlaybooks` failure** (line 31-35): gracefully degrades by returning the roles catalog without the playbooks catalog. The comment correctly notes that "catalog injection is additive metadata."
- **`config.ResolvePaths` failure** (line 27-29): returns a full error, discarding the already-built `rolesCatalog` and propagating the failure to the caller.

Since both callers (`buildBootstrapPrompt` in session_manager.go and `spawnAgent` in api.go) treat any error from this function as fatal (killing the agent and aborting startup), a `ResolvePaths` failure on an otherwise valid workspace would prevent agent spawning entirely due to non-essential metadata.

The two paths should follow the same pattern. If catalog injection is truly additive, `ResolvePaths` failure should also degrade gracefully:

```go
paths, err := config.ResolvePaths(workspace)
if err != nil {
    return rolesCatalog, "", nil
}
```

Alternatively, if `ResolvePaths` failure indicates a genuine misconfiguration, the `LoadPlaybooks` failure should also propagate the error. Choose one strategy and apply it consistently.

## Triage

- Decision: `valid`
- Notes:
  `buildPromptCatalogsForAgent` explicitly treats catalog injection as additive metadata for playbooks, but `ResolvePaths` still aborts bootstrap/spawn even though the already-built role catalog is usable. `ResolvePaths` only canonicalizes the workspace path, so failing there should not be allowed to kill agent startup for master prompts. This should degrade the same way `LoadPlaybooks` already does.
  Resolved by making `ResolvePaths` failures return the already-built roles catalog with an empty playbooks catalog, matching the existing additive-metadata behavior.
