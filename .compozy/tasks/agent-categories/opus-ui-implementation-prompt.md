You are Claude Code Opus working in `/Users/pedronauck/Dev/compozy/agh`.

Implement the web/UI portion of `.compozy/tasks/agent-categories/_techspec.md`.

Non-negotiable context:

- Conversation can be Portuguese, but all artifacts/code/docs must be English.
- This workspace has a dirty worktree. Do not revert or overwrite unrelated edits. Work with existing files.
- Do not run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`).
- Backend, OpenAPI, and generated TS already include `category_path?: string[]`.
- Do not edit Go/internal/backend/docs files. This task is UI-only.
- Preserve AGH design rules from `DESIGN.md`, `web/CLAUDE.md`, and `web/AGENTS.md`: warm dark tokens, flat depth, no shadows/gradients, no invented colors, no truthful-UI violations.

Allowed write scope:

- `packages/ui/src/index.ts`
- `packages/ui/src/components/reui/tree.tsx` only if an export/typing fix is required
- `web/package.json` and `bun.lock` only via package manager if `@headless-tree/react` is not already present
- `web/src/**`
- `web/e2e/**` if fixtures/tests need category support

Required UI implementation:

1. Export the tree primitives from `@agh/ui`:
   - `Tree`
   - `TreeItem`
   - `TreeItemLabel`
   - `TreeDragLine`

2. If missing, install the React peer exactly with the package manager:
   - `bun add @headless-tree/react@^1.6.3 --filter agh-web`
   - Do not hand-edit package dependency entries.

3. Add agent category utilities inside the agent system using kebab-case file names:
   - Build a deterministic client-side tree from flat `AgentPayload[]`.
   - Root-level agents remain root-level leaves; do not add an "Uncategorized" folder.
   - Folder IDs must be prefixed with `category:`.
   - Leaf IDs must be prefixed with `agent:`.
   - Siblings sort case-insensitively; folders before leaves.
   - Format labels as `Marketing / Sales`.
   - Treat `undefined` and `[]` as root-level.

4. Replace the flat `AgentList` in `web/src/components/app-sidebar.tsx` with a dedicated `AgentCategoryTree`.
   - It must use `packages/ui/src/components/reui/tree.tsx` via `@agh/ui`.
   - It must use `@headless-tree/react` for tree behavior.
   - Preserve existing loading/empty/error states and test IDs:
     - `agents-loading`
     - `agents-empty`
     - `agent-row-${agent.name}`
     - `agent-active-${agent.name}`
     - `agent-status-dot-${agent.name}`
   - Add deterministic category test IDs such as `agent-category-${joinedPath}`.
   - Active agent route ancestors should expand initially; if no active agent, top-level categories should be expanded.
   - Active-session status dots must still work.

5. Add command-based agent selectors under the agent system:
   - `AgentCommandSelect` for single selection.
   - `AgentCommandMultiSelect` for multi selection.
   - Shared private list component is fine.
   - Must use `packages/ui/src/components/command.tsx` via `@agh/ui`.
   - Group results by formatted category label; root-level group heading is `Agents`.
   - Show agent name, provider, and category metadata.
   - Single select closes on selection and calls `onChange(agent.name)`.
   - Multi select stays open, calls `onToggle(nextNames)`, and marks selected items with `data-checked`.

6. Replace current agent selection call sites:
   - `web/src/systems/session/components/session-create-dialog.tsx`
     - Replace agent `NativeSelect` only.
     - Preserve trigger/test ID `session-create-agent-select`.
     - Keep provider select as-is.
   - `web/src/routes/_app/settings/skills.tsx`
     - Replace agent scope `NativeSelect` only.
     - Preserve existing surrounding field test IDs and put the trigger on `settings-page-skills-agent-select-input`.
   - `web/src/systems/network/components/network-create-channel-dialog.tsx`
     - Replace the ad-hoc agent button list with `AgentCommandMultiSelect`.
     - Preserve per-agent item test IDs `network-agent-option-${agent.name}`.

7. Update tests and fixtures:
   - Unit tests for category utilities.
   - Component tests for `AgentCategoryTree`.
   - Component tests for single and multi command selectors.
   - Update existing session/settings/network tests that relied on `selectOptions`.
   - Update web e2e fixture types to include optional `category_path?: string[]` where needed.
   - Add/adjust Storybook stories only if the existing web story pattern already covers these components.

Validation to run before finishing:

- `make web-lint`
- `make web-typecheck`
- `make web-test`

If one of these commands fails, fix the root cause instead of weakening tests. If an unrelated pre-existing failure blocks completion, report it precisely with command output and the file/test responsible.

Return a concise final summary with:

- Files changed.
- Tests/commands run and whether each passed.
- Any blockers or follow-up risks.
