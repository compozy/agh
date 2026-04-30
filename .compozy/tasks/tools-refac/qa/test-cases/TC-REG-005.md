# TC-REG-005: Skill Catalog And `agh-agent-setup` Tool-First Regression

**Priority:** P1 (High)
**Type:** Regression / Skills
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `internal/skills/catalog.go` text references `agh__skill_view` first (with conditional CLI fallback only when the tool is denied), and that `internal/skills/bundled/skills/agh-agent-setup/SKILL.md` no longer treats `agh__catalog` as opt-in. Confirm bundled `agh-tools-guide` teaches the canonical loop.

## Traceability

- Tasks: task_02 (guidance bundle + prompt section), task_11 (docs alignment).
- TechSpec: "Skills, Tools, Resources, Bundles", "Delete Targets".
- ADR: ADR-001.
- Surfaces: `internal/skills/catalog.go`, `internal/skills/bundled/skills/agh-tools-guide/`, `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`, `packages/site/content/runtime/core/configuration/agent-md.mdx`.

## Preconditions

- Working tree as committed.

## Test Steps

1. Inspect `internal/skills/catalog.go` and capture the catalog text branch:
   ```bash
   grep -n "agh__skill_view\|agh skill view\|agh skill" internal/skills/catalog.go \
     | tee qa/logs/TC-REG-005/catalog-grep.txt
   ```
   - **Expected:** Catalog text includes `agh__skill_view` first; CLI fallback appears only inside the conditional branch (e.g., `if !skillViewCallable`).

2. Inspect `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`:
   ```bash
   grep -nE "opt-in|agh__catalog (is|must) (be )?enabled|agh skill view" \
     internal/skills/bundled/skills/agh-agent-setup/SKILL.md \
     | tee qa/logs/TC-REG-005/agent-setup-grep.txt
   ```
   - **Expected:** No prose calling `agh__catalog` opt-in. CLI mentions, if present, are clearly framed as fallback or operator path.

3. Inspect bundled `agh-tools-guide` content:
   ```bash
   ls internal/skills/bundled/skills/agh-tools-guide
   cat internal/skills/bundled/skills/agh-tools-guide/SKILL.md | tee qa/logs/TC-REG-005/agh-tools-guide.md
   ```
   - **Expected:** Body teaches `agh__tool_search → agh__tool_info → invoke` and clarifies CLI is management/fallback.

4. Inspect the rewritten `packages/site/content/runtime/core/configuration/agent-md.mdx`:
   ```bash
   grep -nE "tools-first|default discovery|agh__skill_view" \
     packages/site/content/runtime/core/configuration/agent-md.mdx \
     | tee qa/logs/TC-REG-005/agent-md-grep.txt
   ```
   - **Expected:** Page reflects tool-first posture and default discovery.

5. Run focused Go tests:
   ```bash
   go test ./internal/skills ./internal/skills/bundled -count=1 | tee qa/logs/TC-REG-005/skills-tests.log
   go test ./internal/daemon -run "TestPromptSectionTools|TestComposedAssembler" -count=1 \
     | tee qa/logs/TC-REG-005/daemon-prompt-tests.log
   ```

## Evidence To Capture

- All grep outputs.
- Bundled skill content snapshot.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Catalog text after `agh__skill_view` is denied by policy | runtime test path | Catalog text falls back to CLI guidance conditionally |
| `agh-tools-guide` not registered in bundled-content index | regression | `internal/skills/bundled/content.go` test fails; bundle missing entry |
| Skip-tools mode `[tools].enabled=false` | runtime config | Prompt section omitted; catalog still references `agh__skill_view` if otherwise callable |

## Channels Exercised

- Source files (no runtime exec required).
- Skills + daemon Go tests.

## Related Test Cases

- TC-FUNC-002 (prompt section + bundled guide).
- TC-REG-003 (site build).
