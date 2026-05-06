# TC-INT-002: Deterministic Recall, Signals, Shadowing, And Workspace Identity

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Prove deterministic recall behavior across the final scope model and verify live recall signals feed dreaming without leaking system artifacts.

## Preconditions

- [ ] Scenario workspace has global, workspace, agent-global, and agent-workspace memory fixtures sharing at least one `(type, slug)` identity.
- [ ] One stale memory fixture is older than one day.
- [ ] One `_system/` fixture exists and must not be recalled.
- [ ] Workspace `.agh/workspace.toml` exists.

## Test Steps

1. **Run focused recall tests**
   - Input: `go test ./internal/memory/recall ./internal/memory ./internal/workspace -count=1`
   - **Expected:** Ranking, trivial skip, CJK/trigram, shadowing, signal writes, and workspace identity tests pass.

2. **Search as root agent**
   - Input: `agh memory search "sentinel project decision" --scope agent --agent reviewer --agent-tier workspace -o json`
   - **Expected:** Agent-workspace entry shadows agent-global, workspace, and global entries for same identity.

3. **Search with include-shadowed**
   - Input: CLI/API search with include-shadowed or equivalent list filters.
   - **Expected:** Shadowed entries are visible only through explicit shadow/debug surface and carry clear metadata.

4. **Verify stale banner**
   - Input: query stale fixture.
   - **Expected:** Returned package includes freshness warning/banner.

5. **Verify live recall signal**
   - Input: inspect `memory_recall_signals` after non-empty recall.
   - **Expected:** `recall_count`, `last_recalled_at`, and `recall_score` update without blocking recall response.

6. **Move the workspace directory**
   - Input: move scenario workspace to a new path and rerun workspace resolve/search.
   - **Expected:** Same `workspace_id` is read from moved `.agh/workspace.toml`; rows are not orphaned.

7. **Verify `_system/` exclusion**
   - Input: search for unique text only present in `_system/`.
   - **Expected:** No result unless an explicitly supported include-system test path is used.

## Evidence To Capture

- Recall/search JSON.
- SQL rows for `workspace_id`, `memory_recall_signals`, and shadow event.
- Workspace path before/after move.
- `_system` exclusion query result.

## Pass Criteria

- Scope precedence is agent-workspace, agent-global, workspace, global.
- Recall signals update and do not bubble failure to callers.
- Workspace move does not orphan memory.

