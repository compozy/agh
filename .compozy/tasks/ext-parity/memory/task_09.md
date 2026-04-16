# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Task 09 migrates `agent` and `skill` desired-state authority to canonical resource records.
- Acceptance requires typed codecs/stores/projectors, config/workspace/extension publication paths, preserved `internal/skills` domain ownership for content/provenance/MCP sidecars, canonical tool/MCP reference alignment from task 08, cutover coverage, clean `make verify`, tracking updates, and one local commit.

## Important Decisions

- The approved PRD/TechSpec is the design source for this execution task; no separate brainstorming approval loop is needed.
- This run must follow a clean cutover: no dual-write, compatibility shim, or legacy definition authority after agent/skill resource publication is authoritative.
- `internal/config` and `internal/skills` own typed resource codecs/spec validation for their domain shapes; daemon wiring owns publication, projection, and source sync.
- `internal/skills` remains responsible for skill content parsing, provenance verification, and `mcp.json` sidecar merge; the resource runtime stores validated desired state and never parses skill files.
- Extension-provided skills are no longer registered directly into `skills.Registry`; extension snapshots feed daemon source sync and canonical resource projection.
- Session start/resume and API agent listing now use an injected resource-backed agent catalog when present, leaving config fallback only for runtimes without the shared resource kernel.

## Learnings

- Shared workflow memory confirms task 08 completed the `tool` and `mcp_server` cutover; agent tool references and skill MCP attachments should resolve against those canonical catalogs.
- Boot/reload ordering matters: agent/skill source sync must run before hook binding sync so hook declarations on projected agents/skills are visible to the hook-binding source syncer.

## Files / Surfaces

- Touched surfaces: `internal/config`, `internal/skills`, `internal/daemon`, `internal/session`, `internal/api/{core,httpapi,udsapi}`, `internal/extension`, task tracking, and workflow memory.

## Errors / Corrections

- `make verify` initially failed on lint-only issues after the cutover; fixed by shortening projector registration construction, removing an unused legacy resume validator, replacing nil-context test input, and cleaning up repeated constants/unused parameters.
- A full exploratory `go test -tags integration ./internal/extension` hit unrelated provider conformance failures; task-relevant extension integration coverage passed, and the required full `make verify` gate passed after fixes.

## Ready for Next Run

- Current phase: complete. Code-only commit `38a9b7b` (`refactor: migrate agents and skills to resources`) was created, and post-commit `make verify` passed.
