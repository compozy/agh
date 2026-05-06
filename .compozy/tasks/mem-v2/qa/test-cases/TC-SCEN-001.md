# TC-SCEN-001: Controller-Backed Write Is Searchable Across Public Surfaces

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Behavioral Scenario Charter

- Startup situation: isolated daemon running against a fresh workspace with `.agh/workspace.toml` and empty Memory v2 catalog.
- Operator intent: write a memory through a supported controller-backed surface and immediately find it through search/list/show across CLI, HTTP, UDS, and native-tool paths.
- Expected business outcome: operators and agents do not need an undocumented `agh memory reindex` after a supported write.
- AGH surfaces used: CLI, HTTP, UDS, native tool, SQLite evidence, docs reference.
- Real provider/LLM expectation: no external LLM required when controller `mode = "rules"` or the candidate is a fresh slot.
- Blocked live-provider boundary, if any: document only if daemon startup or provider auth blocks runtime e2e.

## Preconditions

- [ ] Fresh `qa/bootstrap-manifest.json` exists.
- [ ] `AGH_HOME`, API base URL, and UDS socket are exported from the manifest.
- [ ] Scenario workspace path is recorded in `qa/logs/TC-SCEN-001/workspace.txt`.
- [ ] Memory config uses deterministic controller behavior for a fresh slot.

## Journey Steps

1. **Write a workspace memory through CLI**
   - Input: `agh memory write --scope workspace --type project --name qa-search-visibility --content @qa/fixtures/search-visibility.md -o json`
   - **Expected:** Response includes a controller decision with op `ADD` or committed write status, a target filename, and no raw prompt/LLM payload.

2. **Search immediately through CLI without reindex**
   - Input: `agh memory search "search visibility sentinel" --scope workspace -o json`
   - **Expected:** Search returns the newly written memory and includes the same filename or selector produced by the write.

3. **Search the same memory through UDS**
   - Input: `curl --unix-socket "$AGH_UDS_SOCKET" -X POST http://localhost/api/memory/search ...`
   - **Expected:** UDS response returns the same memory and deterministic JSON error envelope is absent.

4. **Search the same memory through HTTP**
   - Input: `curl -s -X POST "$AGH_HTTP_BASE/api/memory/search" ...`
   - **Expected:** HTTP response matches the UDS memory identity and score/order is stable enough to place the sentinel in returned entries.

5. **Show the memory through native tool path**
   - Input: invoke `agh__memory_show` or equivalent hosted/native tool route for the returned filename.
   - **Expected:** Tool output is read-only, redacted, and agrees with CLI/API show content.

6. **Inspect durable evidence**
   - Input: SQL queries against the workspace DB for `memory_decisions`, `memory_events`, and catalog/chunk tables.
   - **Expected:** One decision row exists before/with mutation evidence, one `memory.write.committed` event exists, and searchable chunks exist without manual reindex.

7. **Disruption probe: restart daemon and search again**
   - Input: stop/start isolated daemon, then rerun CLI and UDS search.
   - **Expected:** Search remains visible; pending decision replay is idempotent and does not duplicate catalog rows.

## Required Evidence

- CLI write JSON and immediate search JSON.
- UDS and HTTP search payloads.
- Native-tool show payload.
- SQL evidence for `memory_decisions`, `memory_events`, catalog entries, and chunks.
- Daemon restart log.
- Explicit note that `agh memory reindex` was not run before the first successful search.

## Pass Criteria

- The memory is searchable through CLI, UDS, and HTTP immediately after write.
- Native show returns the same entry.
- Durable DB evidence proves controller/WAL/event path.
- No undocumented reindex is required.

## Failure Criteria

- Search misses until `agh memory reindex` is run.
- CLI/API/UDS disagree about identity, content, or deterministic error shape.
- Direct storage mutation appears without a controller decision.

