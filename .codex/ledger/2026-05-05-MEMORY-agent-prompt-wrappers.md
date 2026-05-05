Goal (incl. success criteria):

- Complete network-threads task_12 in /Users/pedronauck/Dev/compozy/agh2: update agent prompt wrappers, structured prompt metadata, daemon/harness guidance, bundled agh-network skill text, tests, tracking, verification, and one local commit if clean.

Constraints/Assumptions:

- Follow user/system/developer instructions, repo AGENTS.md/CLAUDE.md/internal guidance, task_12.md, \_techspec.md, ADRs, dependent tasks 06/09/10, workflow memory, and required skills.
- No destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- Must read workflow memory before code edits and update task memory as decisions/learnings/touched surfaces change.
- Must use cy-workflow-memory, cy-execute-task, cy-final-verify, agh-code-guidelines, golang-pro, documentation-writer, agh-test-conventions, and testing-anti-patterns.
- Automatic commit is enabled only after clean verification, self-review, and tracking updates; unrelated dirty tracking/QA files already exist and must not be reverted.

Key decisions:

- Scope stays on prompt contracts and agent guidance; runtime routing and tool schemas are already finalized by tasks 06, 09, and 10.
- Structured network prompt metadata should carry the rendered wrapper trust marker.

State:

- Complete: implementation, verification, tracking/memory updates, task-scoped commit, and post-commit verification are done.

Done:

- Scanned existing ledgers for network-thread overlap and read relevant task_04/task_05/task_06 ledgers plus current agh2 worktree state.
- Loaded required workflow, execution, verification, Go, documentation, test, no-workarounds, and debugging skill instructions.
- Read workflow memory, task_12 memory, root/internal guidance, COPY.md, `_techspec.md`, `_tasks.md`, `_design.md`, ADRs, and dependent tasks/memory for 06/09/10.
- Captured pre-change signals: `PromptNetworkMeta` lacks trust; bundled `agh-network` lacks final thread/direct/work native tool and CLI read guidance; daemon/acpmock fixtures still contain old `--thread-id`/`--direct-id`/`--work-id` and `kind:"direct"` expectations.
- Baseline focused tests passed before implementation.
- Added `PromptNetworkMeta.Trust`, wrapper/metadata trust parity, lifecycle-only wrapper `work-id`, final native-tool-first reply guidance, acpmock trust matching, final CLI fixture flags, and bundled `agh-network` thread/direct/work guidance.
- Task-specific unit/package tests, network E2E integration tests, absence scan, `git diff --check`, and full `make verify` passed.
- Workflow memory and task tracking updated for task_12 completion.
- Created local commit `96996f7b` (`feat: align network prompt wrappers and skill`) with 12 task-scoped code/test/fixture files.
- Post-commit `make verify` passed with Go lint at 0 issues, 8,386 Go tests, package-boundary check, frontend checks, and build.

Now:

- Final response.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None yet.

Working set (files/ids/commands):

- Repo: /Users/pedronauck/Dev/compozy/agh2
- Task files: .compozy/tasks/network-threads/task_12.md, \_tasks.md, \_techspec.md, adrs/
- Workflow memory: .compozy/tasks/network-threads/memory/MEMORY.md, memory/task_12.md
- Task-scoped code/test files: internal/acp/types.go, internal/acp/client_test.go, internal/network/delivery.go, internal/network/delivery_test.go, internal/skills/bundled/skills/agh-network/SKILL.md, internal/skills/bundled/bundled_test.go, internal/testutil/acpmock/fixture.go, internal/testutil/acpmock/fixture_test.go, internal/testutil/acpmock/testdata/network_collaboration_fixture.json, internal/daemon/daemon_network_collaboration_integration_test.go, internal/daemon/network_e2e_assertions_test.go, internal/daemon/prompt_input_composite_integration_test.go
