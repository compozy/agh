---
name: agh-memory-guide
description: Manage AGH persistent memory entries, scopes, and dream checks from the CLI.
version: "1.0.0"
---

# AGH Memory Guide

Use this guide when you need to inspect or maintain AGH's persistent memory layer.

## What AGH memory stores

AGH memory is durable markdown stored outside the transient session prompt. It is designed for information that should survive across sessions, such as project context, durable user preferences, or reusable reference notes.

AGH memory is organized by scope:

- `global`: information that should apply across workspaces
- `workspace`: information that belongs to one repository or worktree

When in doubt, keep information in the narrowest scope that still makes it reusable.

## List and show memory entries

List all visible memory files:

```bash
agh memory list
```

List only global memory:

```bash
agh memory list --scope global
```

List only workspace memory:

```bash
agh memory list --scope workspace
```

Show a specific memory entry:

```bash
agh memory show architecture.md --scope workspace
```

If the same filename exists in multiple scopes, pass `--scope` so you know exactly which record you are reading.

## Write durable memory

Create or update a workspace memory file:

```bash
agh memory write \
  --name "Architecture decisions" \
  --scope workspace \
  --type project \
  --description "Architecture decisions for the current repository" \
  --content "Keep this file focused on durable decisions and constraints."
```

Create a global user preference memory:

```bash
agh memory write \
  --name "Coding preferences" \
  --scope global \
  --type user \
  --description "Reusable coding preferences" \
  --content "Prefer explicit errors and table-driven tests."
```

Use memory for durable facts, not session transcripts. If the note is just temporary working state, keep it in the task or chat context instead.

## Delete and trigger dreams

Delete an outdated memory file:

```bash
agh memory delete architecture.md --scope workspace
```

Trigger a gated dream check for the current workspace:

```bash
agh memory dream trigger
```

Dream checks are useful after a large batch of edits or when you want AGH to review whether the current workspace has enough signal for promotion before the next session.

## Practical rules

1. Put user-wide preferences in `global`.
2. Put repository-specific facts in `workspace`.
3. Keep each file narrow and durable.
4. Prefer updating an existing memory file over scattering the same fact across multiple files.
5. Run `agh memory list` before writing a new file so you do not duplicate an existing note.

If a memory file is becoming a running log, split the durable facts into one stable file and move the transient notes elsewhere.
