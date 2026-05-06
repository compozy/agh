Use the `$qa-report` skill for AGH.

Task: Generate a behavior-first QA report for the implemented `agent-categories` feature.

Output path: `.compozy/tasks/agent-categories`

Repository root: `/Users/pedronauck/Dev/compozy/agh`

Context to read before writing QA artifacts:
- `.compozy/tasks/agent-categories/_techspec.md`
- `.tmp/agent-categories-peer-reviews/20260506T180501Z-agent-categories/impl-review-final-round4.pretty.json`
- `AGENTS.md`, `web/CLAUDE.md`, `internal/CLAUDE.md`

Implementation summary:
- Canonical AGENT.md frontmatter field: `category_path: ["Marketing", "Sales"]`.
- Go config parsing/validation/edit/clone/resource paths normalize/validate category_path and preserve casing/order.
- API contract and bundle activation payload expose `category_path` arrays; OpenAPI and web generated typings are co-shipped.
- CLI human/TOON outputs show `category`; workspace info outputs agent category.
- Docs under `packages/site/content/runtime/core/...` explain category_path.
- Web UI renders agents in a tree via `packages/ui/src/components/reui/tree.tsx`; command select and multi-select use categorized Command groups.
- Stories/tests/e2e were added for agent category UI.
- Peer review loop ended with Opus round 4 verdict SHIP / 0 blockers / 0 risks / 0 nits.
- Latest local verification before QA report: full `make verify` passed.

Required QA artifacts:
1. Create `.compozy/tasks/agent-categories/qa/test-plans/agent-categories-test-plan.md`.
2. Create focused test cases under `.compozy/tasks/agent-categories/qa/test-cases/` covering at least:
   - Go config parse/validation/edit/clone/resource category_path behavior.
   - API/contract/codegen category_path propagation.
   - CLI human/JSON/TOON agent and workspace output category behavior.
   - Web sidebar tree behavior with root-level and nested categories.
   - Web command select/multi-select grouping behavior in session, settings skills, and network dialogs.
   - Playwright/browser scenario for categorized agents in sidebar and session picker.
   - Regression cases for invalid category segments, casing preservation, and no slash-string aliases.
3. Create `.compozy/tasks/agent-categories/qa/scenario-contract.json` with machine-readable minimums appropriate for this feature.
4. Create `.compozy/tasks/agent-categories/qa/behavioral-scenario-charter.yaml` for qa-execution.
5. Do not run tests. Do not edit production code. This is a QA planning/reporting pass only.
6. Keep artifacts in English.

Return a concise summary listing artifact paths created and any QA execution prerequisites.
