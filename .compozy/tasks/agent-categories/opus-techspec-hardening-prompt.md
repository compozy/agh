# Agent Categories TechSpec Hardening

You are Claude Code Opus reviewing and improving the AGH task `.compozy/tasks/agent-categories/_techspec.md`.

## Objective

Rewrite `.compozy/tasks/agent-categories/_techspec.md` so it follows the complete structure from `.agents/skills/cy-create-techspec/references/techspec-template.md` and expands the required testing scenarios before implementation begins.

## Hard Constraints

- Edit only `.compozy/tasks/agent-categories/_techspec.md`.
- Do not edit Go, TypeScript, package manifests, generated files, docs, lockfiles, or any other task artifact.
- Keep the artifact in English.
- Follow the repo's greenfield-alpha posture: no compatibility aliases, no old-state migration, no slash-string fallback, no `categories` alias.
- Preserve the canonical field decision unless you find a root-cause contradiction in the current repo: `category_path: ["Marketing", "Sales"]`.
- Treat `category_path` as display-only metadata. It must not alter runtime agent execution behavior.
- Keep backend/API payloads flat; the web UI builds tree/group presentation client-side.
- Web/UI requirements must retain the user-directed components:
  - `packages/ui/src/components/reui/tree.tsx` for sidebar hierarchy.
  - `packages/ui/src/components/command.tsx` for dedicated command-based agent selection.
- Include extensibility, agent-manageability, and config lifecycle analysis. If no config key is added, state why.
- Include web/docs impact explicitly.
- Include delete targets for rejected/obsolete approaches, if any.

## Required Testing Expansion

Add a stronger, implementation-ready test matrix covering at least:

- AGENT.md frontmatter parsing, normalization, validation, and error messages.
- Preservation through `EditAgentDefFile`, skill enable/disable, clone helpers, workspace clone, resource validation, resource codec, bundle materialization/projection, bundle activation payloads, and daemon sync.
- Public API contract conversion and generated OpenAPI/TypeScript payload shape.
- CLI human, toon, and JSON output for `agent list`, `agent info`, and workspace agent views.
- Native/agent-manageable surfaces such as `workspace_describe` where applicable.
- Web category utilities: stable IDs, nested folders, root-level leaves, sorting, empty/loading/error states.
- Sidebar tree behavior: active route, active session indicator, keyboard navigation, default expansion, deterministic test IDs.
- Command selector behavior: keyboard search, grouped results, empty state, single-selection close behavior, multi-selection open behavior, checked state, selected-count display, preserved test IDs.
- Fixtures/stories/mocks that make category behavior visible.
- Negative cases: blank segments, `"."`, `".."`, slash/backslash, whitespace-only values, non-array values, and unsupported aliases.
- Verification commands and codegen/docs regeneration requirements.

## Output Requirements

After editing the TechSpec, print:

1. The file path changed.
2. A concise summary of the new/expanded testing scenarios.
3. Any implementation risks you intentionally captured in the TechSpec.

