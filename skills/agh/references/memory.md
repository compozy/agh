# Memory

## Contents

- What memory stores
- Scopes and types
- CLI operations
- Search, reindex, promote, and reload
- Recall traces
- Hygiene
- When not to write memory

## What Memory Stores

AGH memory is durable Markdown outside transient session prompts. Use it for facts that should survive across sessions: project context, user preferences, durable decisions, and reusable references.

Do not use memory as a transcript, scratchpad, or replacement for task state. If the information is temporary working state, keep it in the current task, run summary, or conversation.

## Scopes And Types

Use the narrowest durable scope that still makes the information reusable:

- global applies across workspaces.
- workspace belongs to one repository or worktree.
- agent belongs to one agent tier or definition when supported by the current memory surface.

Common memory types include user, feedback, project, and reference. Choose the type by the purpose of the content, not by where it was discovered.

## CLI Operations

    agh memory list
    agh memory list --scope global
    agh memory list --scope workspace
    agh memory show architecture.md --scope workspace

Create or update durable memory:

    agh memory write --name "Architecture decisions" --scope workspace --type project --description "Architecture decisions for the current repository" --content "Keep this file focused on durable decisions and constraints."

Delete outdated memory:

    agh memory delete architecture.md --scope workspace

Trigger a gated consolidation check:

    agh memory dream trigger

## Search, Reindex, Promote, And Reload

Search deterministic Memory v2 recall before opening individual files:

    agh memory search "auth sessions" --scope workspace -o json
    agh memory search "review tone" --scope agent --agent reviewer --agent-tier global --include-system -o json

The search path prefers the derived catalog and falls back to deterministic lexical search when needed. Rebuild derived indexes after large memory edits or suspected catalog drift:

    agh memory reindex --scope workspace -o json

Promote durable entries across scopes through the daemon so provenance and controller decisions stay auditable:

    agh memory promote architecture.md --from workspace --to global --dry-run -o json
    agh memory promote review.md --from agent:workspace --to agent:global --agent reviewer -o json

Invalidate frozen memory snapshots for future session boots with reload:

    agh memory reload --scope workspace -o json

There is no `agh memory invalidate` command in the current CLI. Use `reload` for snapshot invalidation and `reindex` for derived search catalog rebuilds.

## Recall Traces

Use recall traces to inspect what memory entered a session turn without exposing raw transient context:

    agh memory recall trace <session_id> <turn_seq> -o json

Recall traces are diagnostic evidence. They do not authorize task state changes, review verdicts, or durable memory writes by themselves.

## Hygiene

1. Run agh memory list before writing a new memory entry.
2. Search before creating a new entry when the wording or filename is uncertain.
3. Update an existing file when the fact belongs there.
4. Keep each entry narrow and durable.
5. Prefer stable decisions and preferences over process notes.
6. Remove or rewrite outdated entries instead of layering contradictions.

If a memory file becomes a running log, extract stable facts into focused files and move transient material elsewhere.

## When Not To Write Memory

Do not write memory for raw transcripts, secrets, claim tokens, OAuth material, MCP credentials, provider state, temporary plans, unverified assumptions, or facts scoped only to the current prompt turn.

Memory should reduce future ambiguity. It should not become another source of stale context.
