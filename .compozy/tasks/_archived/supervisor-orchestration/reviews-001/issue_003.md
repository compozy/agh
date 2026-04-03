---
status: resolved
file: internal/prompt/role_catalog.go
line: 65
severity: medium
author: claude-code
---

# Issue 003: BuildPlaybookCatalog silently excludes playbooks without domain



## Review Comment

`BuildPlaybookCatalog` requires both `name` and `domain` to be non-empty (line 64-66), silently skipping playbooks that have no domain set:

```go
if name == "" || domain == "" {
    continue
}
```

However, `config.Playbook.Domain` is documented as optional in the config struct. A user who creates a valid playbook without specifying a domain will find it invisible to the master agent's catalog, with no warning or indication of why.

This contrasts with `BuildRoleCatalog` where `Type` is a required field that determines agent capabilities, making the skip semantically correct.

Two options:
1. Use a fallback like `"general"` or `"unspecified"` for empty domains, similar to how `RenderContext` uses `"(unspecified)"` for missing fields.
2. Render entries without domain using a simpler format: `- playbook-name - Description text`.

Option 1 is the simpler fix:

```go
if domain == "" {
    domain = "general"
}
```

## Triage

- Decision: `valid`
- Notes:
  `config.Playbook.Domain` is optional in the config model, but `BuildPlaybookCatalog` currently drops otherwise valid playbooks when the field is blank. That makes valid user-authored playbooks disappear from the master prompt catalog with no feedback. The renderer should preserve those entries with a stable fallback domain label instead of silently excluding them.
  Resolved by defaulting missing playbook domains to `general` in `BuildPlaybookCatalog` and adding a regression test for the fallback rendering.
