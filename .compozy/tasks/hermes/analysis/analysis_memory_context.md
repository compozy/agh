# Hermes vs AGH — Memory & Context Management

## Executive Summary

- **AGH has a more rigorous OS-memory substrate** (dual-scope Markdown files + SQLite FTS catalog + frontmatter taxonomy + dream consolidation) than Hermes' built-in memory, which is just two flat files (`MEMORY.md`, `USER.md`). The AGH catalog at `internal/memory/catalog.go:29-82` (FTS5 + operation log + scope state) is architecturally ahead of anything in Hermes' builtin tier.
- **Hermes wins on the plugin/provider abstraction and the user-visible lifecycle**: a `MemoryProvider` ABC (`.resources/hermes/agent/memory_provider.py:42-232`), a `MemoryManager` router (`memory_manager.py:83-373`), a plugin discovery layer (`plugins/memory/__init__.py:122-285`), and a curses wizard (`hermes_cli/memory_setup.py:221-356`). AGH has exactly one backend, wired directly in `daemon/`.
- **Hermes has structured turn hooks** (`on_turn_start`, `on_session_end`, `on_pre_compress`, `on_memory_write`, `on_delegation`, `prefetch`/`queue_prefetch`, `sync_turn`). AGH only injects memory pre-turn via `recall.go:22-59` and reacts post-turn not at all — memory writes only happen when a dream session spawns.
- **Frozen-snapshot prompt-cache discipline** (`tools/memory_tool.py:124-140, 359-370`): memory in the system prompt is captured once at session start and never mutated mid-session. AGH re-reads indexes on every prompt assembly (`assembler.go:58-81`); fine today, but worth adopting if we ever add prompt caching.
- **Hermes hardens memory writes** (`tools/memory_tool.py:65-102`): regex scan for prompt-injection, invisible-unicode detection, exfil patterns. AGH has zero scanning at write time. Hermes' parallel `@ reference` sensitive-path blocklist (`context_references.py:342-361`) is likewise missing in AGH.

## Capability-by-Capability Gap Analysis

### Memory scopes

- **Hermes**: two built-in files (global `MEMORY.md` / `USER.md`) scoped to `$HERMES_HOME/memories/`. No workspace scope in the built-in. External providers add their own scoping (peer/session in Honcho). See `tools/memory_tool.py:53-55`.
- **AGH**: dual-scope (`ScopeGlobal`, `ScopeWorkspace`) baked into the type system (`types.go:25-33`), with per-workspace `.agh/memory/` directories. Stronger.
- **Gap for AGH**: no session-scope or agent-scope. Hermes' provider API carries `agent_context`, `agent_identity`, `parent_session_id`, `user_id` (`memory_provider.py:61-81`); AGH memory has `AgentName` in the header but no runtime-level scoping.

### Memory write API

- **Hermes**: LLM-facing `memory` tool with `add|replace|remove` (`tools/memory_tool.py:463-562`); `replace`/`remove` use short-unique-substring matching. Rejects duplicates, enforces char budgets (2200 / 1375), scans for injection, file lock + atomic temp-rename (`tools/memory_tool.py:142-178, 431-460`).
- **AGH**: file-based write (`store.go:149-172`) with frontmatter validation and closed taxonomy (`types.go:155-167`). Atomic write + derived-catalog sync. No injection scanning, no substring edit, no per-file size budget (only prompt-index truncation at `store.go:806-834`).
- **Gap**: (a) no injection scan, (b) no substring edit, (c) no soft scope budget.

### Memory read / retrieval

- **Hermes built-in**: read-only via system-prompt injection (frozen snapshot at session start, `tools/memory_tool.py:124-140`). No search, no per-entry fetch. Plugins add their own (Honcho `honcho_search`, `honcho_profile`, `honcho_context`, `honcho_reasoning`).
- **AGH**: Full FTS5 BM25 search (`catalog.go:391-454`), lexical fallback (`catalog.go:653-693`), per-file read, scope-filtered lists, snippet extraction. Catalog keeps a `content_hash` per entry for drift detection.
- **AGH strength**: structurally ahead of Hermes' built-in. No gap here.

### Memory consolidation

- **Hermes**: no equivalent. Honcho relies on server-side consolidation; holographic plugin uses HRR vectors for compositional storage. No periodic "dream" loop.
- **AGH**: dream consolidation runtime (`consolidation/runtime.go:39-201`) with time + session gates (`dream.go:171-212`), PID-based cross-process lock with mtime semantics (`lock.go:32-273`), workspace-aware spawning. The consolidation prompt (`prompt.go:5-56`) is a four-phase orient/gather/consolidate/prune script.
- **AGH strength**: well ahead. **But** there's no per-memory TTL, no automatic dedup beyond what the LLM does in-prompt, no eviction beyond what the LLM performs during Phase 4. Hermes' memory plugins offload this to external backends; AGH does it in-process which is cleaner but entirely trust-based on the LLM's pruning.

### Context references / attachments

- **Hermes**: rich `@file:`, `@folder:`, `@diff`, `@staged`, `@git:`, `@url:` reference system (`agent/context_references.py:16-233`). Enforces 50% hard / 25% soft token budget, sensitive-path blocklist (`.ssh`, `.aws`, `.gnupg`, `.env`, `.netrc`, `.pgpass`, etc. at `context_references.py:21-37, 342-361`), line-range slicing, binary detection, ripgrep-accelerated folder listing.
- **AGH**: no equivalent. AGH has no user-facing way to pin a file or diff into a session's context; the CLI takes a raw prompt.
- **Gap for AGH**: significant for UX. Adding `@` reference syntax with the same sensitive-path guard is high-value, low-risk.

### User-visible memory management

- **Hermes**: `hermes memory setup|status` (`memory_setup.py:221-442`) with curses picker, per-plugin config schemas, env-var writes to `.env`, dependency auto-install via `uv`, post-setup connection tests.
- **AGH**: `agh memory list|search|read|reindex|write` (per `internal/memory/prompt.go:23-29`). No `memory setup` / `memory status` / `memory health` flow for operators.
- **Gap for AGH**: need a `memory health` command (drives `HealthStats` already computed in `store.go:404-458`), and an interactive TUI for listing/editing memory. The backend is ready; the CLI surface is thin.

### Memory backup / export / import

- Neither system has explicit export/import. Both rely on Markdown files on disk.
- **Gap for both; priority low for AGH** — tar of `~/.agh/memory/` + per-workspace `.agh/memory/` is already enough for manual backup.

### Privacy / redaction before storing

- **Hermes**: regex scan at write (`tools/memory_tool.py:65-102`) catches prompt-injection (`ignore previous instructions`), role-hijack, exfiltration (`curl`/`wget` with secret patterns), SSH/env access patterns, invisible unicode. Hermes also has `agent/redact.py` for broader runtime redaction.
- **AGH**: none at memory-write time. `@ references` in Hermes have the sensitive-path blocklist; AGH doesn't because it doesn't have `@ references` yet.
- **Gap for AGH**: mandatory. Memory persists across sessions and is injected into every agent — it is the highest-leverage prompt-injection vector in the system.

### Memory plugin architecture

- **Hermes**: full plugin system (`plugins/memory/__init__.py`) with bundled + user directories (`$HERMES_HOME/plugins/`), `plugin.yaml` manifests, `register(ctx)` hook, `save_config()` for native file formats, `post_setup()` wizard override. Only **one** external provider active at a time (`memory_manager.py:97-141`). Five bundled: Honcho, Hindsight, Mem0, Holographic, RetainDB, Supermemory, OpenViking, Byterover.
- **AGH**: none. `Backend` interface exists (`types.go:88-96`) but only one implementation. Adding a Honcho/Mem0-style backend requires code changes to `daemon/`.
- **Gap for AGH**: depending on roadmap. If AGH plans to support external memory backends (Honcho, Mem0, etc.) — plugin system is needed. If not — keep it single-impl and reject the gap.

### Per-user vs multi-user memory

- Hermes is single-user but the `user_id`/`peer` abstraction in Honcho supports multi-peer.
- AGH is single-user with no peer abstraction.
- **Not a current gap**; revisit if AGH gets gateway/multi-user mode.

### Initial setup UX

- **Hermes**: `hermes memory setup` wizard with dep auto-install + secret prompts to `.env`.
- **AGH**: zero first-run UX. User learns memory exists by reading docs.
- **Gap for AGH**: small. `agh memory init` would drop a sample global `user.md` and set expectations.

### Conflict resolution

- **Hermes built-in**: exact-dup rejection (`tools/memory_tool.py:241-242`), deduplication at load (`tools/memory_tool.py:132-134`), `replace`/`remove` refuse ambiguous substring matches (`tools/memory_tool.py:290-299, 340-349`). Between-session conflicts: re-read under file lock before mutating (`_reload_target`, `tools/memory_tool.py:186-193`).
- **AGH**: file-level atomic writes but no conflict resolution between multiple dream consolidations (though the PID lock prevents concurrent runs). No substring-level ambiguity detection because there's no substring edit primitive.
- **Gap for AGH**: acceptable for now given the consolidation lock. Would need revisiting if multiple agent sessions can write memory concurrently.

### Size limits and compaction

- **Hermes**: per-store char budgets (2200 / 1375 chars total across all entries, `tools/memory_tool.py:116-120`) — hard-enforced at write.
- **AGH**: prompt-index truncation only (`store.go:806-834`: 200 lines / 25 KB); per-file content is unbounded. Dream consolidation does prune but is LLM-driven.
- **Gap for AGH**: consider a soft per-scope cap that triggers a forced dream consolidation, instead of relying purely on the time/session gate.

## Patterns worth stealing

1. **Injection scanner on memory writes** — port `_scan_memory_content` from `tools/memory_tool.py:65-102` into AGH `store.Write`. Memory is the highest-leverage prompt-injection surface.
2. **Sensitive-path blocklist** — port `_ensure_reference_path_allowed` (`context_references.py:342-361`) into any future AGH `@file:` feature and into CLI `agh memory write --from-file` if we ever add one.
3. **`@ reference` expansion in prompts** — full port of `preprocess_context_references` (`context_references.py:105-203`) with AGH's own sensitive-path list. Extremely useful for CLI UX; budget guard prevents runaway injection.
4. **Memory-provider-style hooks** — even without a plugin system, AGH memory would benefit from `on_turn_start` / `on_session_end` / `on_pre_compress` seams so observe/transcript packages can participate cleanly. Today `NewRecallAugmenter` is the only hook.
5. **Frozen system-prompt snapshot discipline** — when AGH eventually adds prompt caching at the orchestration layer, adopt the `_system_prompt_snapshot` pattern (`tools/memory_tool.py:124-140`): capture memory state once per session, let tool calls update disk without churning the prompt.
6. **Health / status CLI** — `HealthStats` already exists (`store.go:404-458`); expose it as `agh memory health` to surface orphan count and last-reindex timestamp.
7. **Operation log table** — Hermes has nothing analogous; AGH already has `memory_operation_log` (`catalog.go:73-81`). Worth exposing via CLI to make memory mutations auditable (`agh memory history`).

## Explicitly skip

- **Agent-internal prompt compression** (`trajectory_compressor.py`, `context_compressor.py`, `manual_compression_feedback.py`, `context_engine.py`). These operate on the LLM message list to stay under the model's context window — AGH doesn't run inference and the sub-agents (Claude Code, Codex, Gemini CLI) manage their own compression. Not applicable.
- **HRR / holographic plugin** (`plugins/memory/holographic/`). Elegant math but solves retrieval at the model-embedding layer. AGH's FTS5 + BM25 is the right tool for its scope.
- **Honcho's dialectic reasoning depth heuristic** (`plugins/memory/honcho/__init__.py:777-866`). Cost-aware LLM chaining for an external backend; only relevant if AGH adopts Honcho as a provider.
- **Honcho per-turn prefetch / queue_prefetch thread orchestration** — the complexity is warranted only for network-bound backends with dialectic LLM calls. AGH's local FTS5 read is sub-ms; no background thread needed.
- **`MemoryManager` multi-provider router**. Worth keeping in mind as a spec for future plugin work, but does not need to be adopted today when AGH has one backend.
