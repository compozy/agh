# TC-FUNC-021 — `agh__skill_view` returns real skill content with budget truncation

- **Priority:** P0
- **Type:** Functional / native tool
- **Trace:** Task 05, TechSpec Skills, ADR-004

## Objective

Prove `agh__skill_view` calls into `internal/skills.Registry`, respects workspace overlays, reuses content verification, and applies registry result budgeting. Oversized content sets `truncated = true`, returns `next_offset`, and uses an artifact reference strategy.

## Test Steps

1. Invoke for a small known skill.
   - **Expected:** Full content returned; `truncated = false`.
2. Invoke for a synthetic large skill > `default_max_result_bytes`.
   - **Expected:** `truncated = true`, `next_offset` typed, partial content; artifact ref present if applicable.
3. Invoke for an unknown skill id.
   - **Expected:** `tool_invalid_input` or `not_found` per descriptor.
4. Workspace overlay alters the skill content; second invoke returns overlay.
5. Confirm `internal/skills.MCPResolver` trust gate respected for sidecar MCP entries the skill declares.

## Automation

- **Target:** Integration
- **Status:** Existing partial
- **Command/Spec:** `go test ./internal/tools ./internal/skills -run TestSkillViewIntegration`
