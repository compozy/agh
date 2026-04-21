# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Align `packages/site` runtime capability docs with the unified model defined in `_techspec.md`, `docs/agents/capabilities.md`, and ADRs 001/002.
- Keep runtime pages operator-focused: authoring, projection, digest, and the three wire roles (brief, rich, transfer), without duplicating the full protocol reference.

## Important Decisions

- Kept the runtime page operator-focused and linked to `protocol/capability-discovery` and `protocol/message-kinds/#capability` for the wire contract instead of restating envelope/validation rules.
- Added explicit `version`, `requirements`, and runtime-derived `digest` coverage to the runtime schema table and validation rules so site docs do not drift from `docs/agents/capabilities.md`.
- Left `packages/site/content/runtime/core/overview/what-is-agh.mdx` and `overview/architecture.mdx` untouched: both already use generic "capabilities" wording that is consistent with the unified model and never mention `recipe`.
- Left `runtime/core/agents/meta.json` untouched; the page list and ordering still reflect the unified story.

## Learnings

- `runtime/core/configuration/agent-md.mdx` had a stale sidecar description implying capabilities were only "advertised to network peers". Tightened to cover brief, rich, and transfer roles.
- `runtime/core/agents/definitions.mdx` had similarly narrow wording for the capability sidecar; aligned with the unified model.
- Site verification gate: `bun run source:generate && bun run test && bun run typecheck` inside `packages/site` is sufficient to catch MDX-breaking edits; no additional docs linter is wired in.

## Files / Surfaces

- Rewrote `packages/site/content/runtime/core/agents/capabilities.mdx` (schema + digest + three-role frame + transfer example + `artifacts_supported` note).
- Updated capability sidecar descriptions in `packages/site/content/runtime/core/agents/definitions.mdx` and `packages/site/content/runtime/core/configuration/agent-md.mdx`.

## Errors / Corrections

- None.

## Ready for Next Run

- Site runtime copy is consistent with `docs/agents/capabilities.md` and the accepted ADRs; task_09 (QA plan) can treat runtime + protocol site docs as a coherent pair.
