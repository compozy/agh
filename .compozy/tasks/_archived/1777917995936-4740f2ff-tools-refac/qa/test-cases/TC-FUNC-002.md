# TC-FUNC-002: Tools Prompt Section And Bundled `agh-tools-guide`

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the new startup prompt section `tools` is rendered, that bundled `agh-tools-guide` content is loaded by the skill catalog, and that catalog text references `agh__skill_view` first when callable. Confirm the canonical loop `agh__tool_search → agh__tool_info → invoke` is taught.

## Traceability

- Task: task_02 (Tools Guidance Assets and Startup Prompt Section).
- TechSpec: "Skills, Tools, Resources, Bundles", "Architectural Boundaries", "Implementation Steps".
- ADR: ADR-001.
- Surfaces: `internal/daemon/prompt_sections.go`, `internal/daemon/composed_assembler.go`, `internal/skills/catalog.go`, `internal/skills/bundled/skills/agh-tools-guide`, `internal/skills/bundled/skills/agh-agent-setup`.

## Preconditions

- Isolated `AGH_HOME` from `agh-qa-bootstrap`.
- `[tools].enabled = true` (default).
- A session bound to a default agent definition.

## Test Steps

1. Start a session and capture the rendered system prompt:
   ```bash
   agh session start --agent default --workspace $WS_ID -o json | tee qa/logs/TC-FUNC-002/session-start.json
   agh session prompt $SID -o text | tee qa/logs/TC-FUNC-002/prompt.txt
   ```
   - **Expected:** Prompt contains a `## Tools` (or equivalent canonical heading) section ordered after the `skills` section and before the `network` section.

2. Confirm bundled `agh-tools-guide` content is part of the catalog and is referenced or rendered:
   ```bash
   agh tool invoke agh__skill_view --input '{"id":"agh-tools-guide"}' -o json | tee qa/logs/TC-FUNC-002/skill-view.json
   ```
   - **Expected:** Returns the bundled guide content. Body teaches `agh__tool_search → agh__tool_info → invoke` and clarifies that CLI is a management/fallback path.

3. Confirm catalog text references `agh__skill_view` first:
   ```bash
   agh tool invoke agh__skill_list -o json | tee qa/logs/TC-FUNC-002/skill-list.json
   ```
   - **Expected:** Catalog text in usage instructions points agents to `agh__skill_view` when the tool is callable. Conditional CLI fallback is shown only when the tool is denied (no other text should advertise CLI-first).

4. Confirm `agh-agent-setup` no longer treats `agh__catalog` as opt-in:
   ```bash
   agh tool invoke agh__skill_view --input '{"id":"agh-agent-setup"}' -o json | tee qa/logs/TC-FUNC-002/skill-view-setup.json
   ```
   - **Expected:** Examples reflect default discovery and tool-first behavior. No "you must explicitly enable `agh__catalog`" prose.

5. Run focused Go tests:
   ```bash
   go test ./internal/daemon -run "TestPromptSection|TestComposedAssembler" -count=1 | tee qa/logs/TC-FUNC-002/daemon-prompt-tests.log
   go test ./internal/skills ./internal/skills/bundled -count=1 | tee qa/logs/TC-FUNC-002/skills-tests.log
   ```
   - **Expected:** All tests pass; tests cover ordering of `HarnessPromptSectionTools` and bundled-content registration.

## Evidence To Capture

- `qa/logs/TC-FUNC-002/session-start.json`
- `qa/logs/TC-FUNC-002/prompt.txt`
- `qa/logs/TC-FUNC-002/skill-view.json`
- `qa/logs/TC-FUNC-002/skill-list.json`
- `qa/logs/TC-FUNC-002/skill-view-setup.json`
- `qa/logs/TC-FUNC-002/daemon-prompt-tests.log`
- `qa/logs/TC-FUNC-002/skills-tests.log`

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| `[tools].enabled=false` in config | `agh config set tools.enabled false` | Prompt section omitted; catalog still references `agh__skill_view` if callable |
| `agh__skill_view` denied by policy | agent that denies `agh__skill_view` | Catalog text falls back to CLI guidance conditionally; tool-first prose still preferred when re-enabled |
| Two sequential prompt rebuilds | Reload agent definition mid-session | Second prompt still includes the tools section with stable bundled content |

## Channels Exercised

- Daemon prompt assembly.
- Bundled skill registry.
- CLI / tool invocation of `agh__skill_view` and `agh__skill_list`.

## Related Test Cases

- TC-FUNC-001 (default discovery overlay).
- TC-REG-005 (catalog and `agh-agent-setup` regression).
- TC-INT-002 (transport parity for `agh__skill_*` and `agh__tool_*`).
