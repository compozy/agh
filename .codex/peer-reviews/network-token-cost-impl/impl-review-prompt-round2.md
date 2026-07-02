You are a senior code reviewer pressure-testing an implementation in the AGH greenfield-alpha
codebase. Zero production users exist; bias toward simpler, deletable solutions over compatibility
shims. Your job is to find what's wrong, not to be polite.

SCOPE OF THIS REVIEW:
AGH Network token optimization megaround implementation after round-1 remediation: channel fanout policies/coordinator enforcement, mentions, subscriptions/digest/mute delivery, digest coalescing and cost allocation, durable guidance suppression, response register, thread-to-task promotion, runtime status-back via agh.runtime, designated task fan-out runs, API/CLI/UDS/native tool surfaces, web controls, docs/official skill updates, migrations v44-v47, generated contract artifacts, and round-1 blocker/nit fixes. The diff is intentionally larger than the normal peer-review threshold because the task contract explicitly requires full megaround implementation peer review until SHIP.

USER-PROVIDED CONTEXT FILES (read fully before reasoning, skip if `none`):
.compozy/tasks/network-token-optimization/codex-megaround-packet.txt
.codex/plans/20260701T200023-0300\*network-token-optimization-megaround.md
.codex/peer-reviews/network-token-cost-impl/impl-review-findings-round1.md

REPO-LEVEL CONTEXT (read any that exist; ignore the ones that don't):

- /CLAUDE.md, /internal/CLAUDE.md, /web/CLAUDE.md, /packages/site/CLAUDE.md
- /docs/\_memory/standing_directives.md
- /docs/\_memory/lessons/

CHANGED FILES:
internal/acp/types.go
internal/api/contract/contract.go
internal/api/contract/responses.go
internal/api/contract/settings.go
internal/api/contract/tasks.go
internal/api/core/agent_channels_internal_test.go
internal/api/core/agent_channels.go
internal/api/core/conversions.go
internal/api/core/interfaces.go
internal/api/core/memory.go
internal/api/core/network_conversations.go
internal/api/core/network_details.go
internal/api/core/network_test.go
internal/api/core/network.go
internal/api/core/prompt_stream.go
internal/api/core/settings_internal_test.go
internal/api/core/settings_test.go
internal/api/core/settings.go
internal/api/core/tasks_test.go
internal/api/core/tasks.go
internal/api/core/test_helpers_test.go
internal/api/core/tools.go
internal/api/httpapi/handlers_test.go
internal/api/httpapi/routes.go
internal/api/spec/spec.go
internal/api/testutil/network_stub.go
internal/api/udsapi/handlers_test.go
internal/api/udsapi/routes.go
internal/cli/agent_kernel.go
internal/cli/client.go
internal/cli/helpers_test.go
internal/cli/network.go
internal/cli/session.go
internal/cli/task.go
internal/config/config.go
internal/config/merge.go
internal/config/task_orchestration.go
internal/config/tool_surface.go
internal/daemon/boot.go
internal/daemon/composed_assembler_test.go
internal/daemon/daemon_test.go
internal/daemon/daemon.go
internal/daemon/harness_context.go
internal/daemon/native_create_tools.go
internal/daemon/native_tools.go
internal/daemon/network_response_register_prompt.go
internal/daemon/network_task_status_observer_test.go
internal/daemon/network_task_status_observer.go
internal/daemon/prompt_input_composite.go
internal/daemon/prompt_sections.go
internal/daemon/task_event_bridge_notifier.go
internal/daemon/task_role_runtime_test.go
internal/daemon/task_role_runtime.go
internal/daemon/task_runtime.go
internal/daemon/truncate_utf8.go
internal/network/audit.go
internal/network/delivery_test.go
internal/network/delivery.go
internal/network/envelope.go
internal/network/hooks_test.go
internal/network/manager_test.go
internal/network/manager.go
internal/network/router_test.go
internal/network/router.go
internal/network/transport_test.go
internal/network/validate.go
internal/observe/tasks.go
internal/situation/task_context.go
internal/store/globaldb/global_db_network_channels_test.go
internal/store/globaldb/global_db_network_channels.go
internal/store/globaldb/global_db_network_conversations.go
internal/store/globaldb/global_db_network_messages_test.go
internal/store/globaldb/global_db_network_messages.go
internal/store/globaldb/global_db_network_preferences.go
internal/store/globaldb/global_db_task_aux.go
internal/store/globaldb/global_db_task_claim.go
internal/store/globaldb/global_db_task_force.go
internal/store/globaldb/global_db_task_review.go
internal/store/globaldb/global_db_task_test.go
internal/store/globaldb/global_db_task.go
internal/store/globaldb/global_db_test.go
internal/store/globaldb/global_db.go
internal/store/globaldb/migrate_task_run_status.go
internal/store/store.go
internal/store/types.go
internal/task/designation.go
internal/task/interfaces_integration_test.go
internal/task/interfaces.go
internal/task/live.go
internal/task/manager_test.go
internal/task/manager.go
internal/task/types.go
internal/tools/builtin_ids.go
internal/tools/builtin/builtin_test.go
internal/tools/builtin/descriptors.go
internal/tools/builtin/extensions.go
internal/tools/builtin/network.go
internal/tools/builtin/tasks.go
internal/tools/builtin/testdata/native-tool-catalog.json
internal/tools/builtin/toolsets.go
openapi/agh.json
packages/site/content/protocol/delivery.mdx
packages/site/content/runtime/cli-reference/network/channels/meta.json
packages/site/content/runtime/cli-reference/network/channels/update.mdx
packages/site/content/runtime/cli-reference/network/digest-mode.mdx
packages/site/content/runtime/cli-reference/network/meta.json
packages/site/content/runtime/cli-reference/network/mute.mdx
packages/site/content/runtime/cli-reference/network/send.mdx
packages/site/content/runtime/cli-reference/network/subscribe.mdx
packages/site/content/runtime/cli-reference/network/subscriptions.mdx
packages/site/content/runtime/cli-reference/network/threads/meta.json
packages/site/content/runtime/cli-reference/network/threads/promote.mdx
packages/site/content/runtime/cli-reference/network/unmute.mdx
packages/site/content/runtime/cli-reference/task/fan-out.mdx
packages/site/content/runtime/cli-reference/task/meta.json
packages/site/content/runtime/cli-reference/task/promote.mdx
packages/site/content/runtime/core/network/channels-and-peers.mdx
packages/site/content/runtime/core/network/delivery-and-safety.mdx
packages/site/content/runtime/core/network/task-ingress.mdx
packages/site/content/runtime/core/network/threads.mdx
sdk/typescript/src/generated/contracts.ts
skills/agh/references/native-tools.md
skills/agh/references/network.md
skills/agh/references/tasks-and-orchestration.md
web/src/generated/agh-openapi.d.ts
web/src/hooks/routes/**tests**/use-settings-network-page.test.tsx
web/src/hooks/routes/use-task-detail-orchestration-tab.ts
web/src/routes/\_app/settings/**tests**/-network.test.tsx
web/src/routes/\_app/settings/network.tsx
web/src/routes/\_app/tasks.$id.tsx
web/src/systems/network/adapters/network-api.ts
web/src/systems/network/components/composer/**tests**/channel-thread-composer.test.tsx
web/src/systems/network/components/composer/**tests**/composer.test.tsx
web/src/systems/network/components/composer/channel-thread-composer.tsx
web/src/systems/network/components/composer/composer.tsx
web/src/systems/network/components/composer/detail-composer.tsx
web/src/systems/network/components/composer/use-composer-state.ts
web/src/systems/network/components/shell/**tests**/channel-policy-dialog.test.tsx
web/src/systems/network/components/shell/channel-header.tsx
web/src/systems/network/components/shell/channel-policy-dialog.tsx
web/src/systems/network/components/thread-overlay/**tests**/thread-overlay-header.test.tsx
web/src/systems/network/components/thread-overlay/**tests**/thread-subscription-control.test.tsx
web/src/systems/network/components/thread-overlay/thread-overlay-header.tsx
web/src/systems/network/components/thread-overlay/thread-overlay.tsx
web/src/systems/network/components/thread-overlay/thread-subscription-control.tsx
web/src/systems/network/components/thread-overlay/use-thread-overlay-view.ts
web/src/systems/network/hooks/use-network-actions.ts
web/src/systems/network/index.ts
web/src/systems/network/lib/query-keys.ts
web/src/systems/network/lib/query-options.ts
web/src/systems/network/mocks/**tests**/network-mocks.test.ts
web/src/systems/network/mocks/fixtures.ts
web/src/systems/network/mocks/handlers.ts
web/src/systems/network/types.ts
web/src/systems/settings/mocks/fixtures.ts
web/src/systems/tasks/adapters/tasks-api.ts
web/src/systems/tasks/components/**tests**/tasks-detail-orchestration-panel.test.tsx
web/src/systems/tasks/components/**tests**/tasks-detail-runs-panel.test.tsx
web/src/systems/tasks/components/**tests**/tasks-fan-out-runs-card.test.tsx
web/src/systems/tasks/components/stories/task-overview-components.stories.tsx
web/src/systems/tasks/components/tasks-detail-orchestration-panel.tsx
web/src/systems/tasks/components/tasks-detail-runs-panel.tsx
web/src/systems/tasks/components/tasks-fan-out-runs-card.tsx
web/src/systems/tasks/hooks/use-task-actions.ts
web/src/systems/tasks/hooks/use-tasks-fan-out-runs-card.ts
web/src/systems/tasks/index.ts
web/src/systems/tasks/mocks/handlers.ts
web/src/systems/tasks/types.ts

DIFF (raw patch):
.codex/peer-reviews/network-token-cost-impl/impl-review-diff-round2.patch

COMMIT LIST (or `none` for staged-only review):
669f6b503 savepoint

TARGET FINDINGS FILE:
/Users/pedronauck/Dev/compozy/agh/.codex/peer-reviews/network-token-cost-impl/impl-review-findings-round2.md

SCOPED-WRITE CONTRACT:

1. You may write exactly one file: the target findings file above.
2. Do not edit source code, tests, configs, docs, specs, ledgers, prompts, summaries, or any other file.
3. Do not create sibling artifacts, temp files, backups, or alternate output files.
4. If you cannot write the exact target file, stop and report the failure briefly. Do not print the review findings to stdout as a fallback.
5. After writing the file, your final chat response must be one sentence: `Wrote {findings_path}`.

YOUR JOB:

1. Read every context file fully. Then read every changed file in full (not just the hunks) — diffs
   hide surrounding state.
2. Cross-check the implementation against any user-provided context (specs, ADRs, RFCs, design
   docs) when present. Flag any requirement, acceptance criterion, or architectural decision that
   is missing, partially implemented, or implemented differently than specified.
3. Identify BLOCKERS — issues that must be fixed before this change ships:
   - Security regressions: raw `claim_token` leaving its boundary, unverified-format identity
     classification, secrets in logs, command/SQL injection, missing authn/authz on a new surface.
   - Concurrency bugs: races, goroutine leaks, missing context cancellation, peer claimer pattern,
     parallel queue alongside `task_runs`, hooks tailing event tables, lock ordering hazards.
   - Correctness bugs: nil deref on hot path, off-by-one on lease/heartbeat math, swallowed errors
     (`_` discard) in production code, panic/log.Fatal in library/handler code.
   - Persistence hazards: schema change without a numbered migration, side-table-vs-JSON inversion,
     `EnsureSchema`-style boot reconciliation for a column change, missing `BEGIN IMMEDIATE` on a
     state-mutating tx, `ORDER BY 0` shape errors.
   - Surface incompleteness: CLI/HTTP shipped without UDS, codegen drift (openapi/agh.json vs
     web/src/generated/agh-openapi.d.ts), backend change without web/docs impact analysis.
   - Test-shape violations: missing `t.Run("Should ...")` subtests, missing `t.Parallel`, mocks
     replacing behavior assertions, status-code-only assertions on HTTP responses, integration
     suite that never touches a real DB when the change is persistence-sensitive.
   - Greenfield violations: compat shims, dual fields, alias renames, "removed/" comment graveyards,
     migration code defending against state that never existed.
   - Truthful-UI violations: web/site rendering controls or metrics the runtime does not actually
     support.
   - Extensibility/agent-manageability gaps: feature reachable only via internal Go calls or web UI
     with no CLI/HTTP/UDS path for agents, no extension/skill/tool/bridge integration where the
     spec required one.
4. Identify RISKS — latent or non-blocking concerns the team should know about: observability gaps
   (missing slog fields, no metrics on a new hot path), test-density holes, doc co-ship missing,
   tight coupling that will hurt the next refactor, performance smells that are fine today but will
   bite at scale.
5. Identify NITS — clarity, naming, dead code, comment policy violations, godoc gaps.
6. Issue a VERDICT: SHIP / FIX_BEFORE_SHIP / REWORK.
   - SHIP — no blockers; risks/nits acceptable as follow-ups.
   - FIX_BEFORE_SHIP — at least one blocker, but the change shape is right; remediation is local.
   - REWORK — structural problems require redesign or a new TechSpec (e.g., two-touch rule fired,
     parallel queue created, abstraction inverted).

CONSTRAINTS:

- Greenfield: prefer "delete the old thing" over "preserve compat".
- Hard cuts only: any rename touches code, storage, APIs, CLI, extensions, specs, RFCs, and
  .compozy/tasks/\* artifacts in the same change.
- task_runs is the single durable queue. Reject any parallel queue.
- ClaimNextRun is the only authoritative claim primitive. Reject any peer claimer.
- Manual operator paths converge with autonomous on the same primitives.
- Hooks dispatch at the call site; never tail event tables.
- claim_token (raw) never crosses transport, channel, log, or memory.
- Generated artifacts co-ship with source change in same PR (openapi + web typings).
- Subagents are read-only; only the paired agent commits code.
- Every error wrapped with `%w`; `errors.Is` / `errors.As` only.
- No `_`-discarded errors in production code or tests without a written justification.

FINDINGS FILE FORMAT:
Write `{findings_path}` as Markdown with this exact frontmatter and headings:

---

schema_version: 1
review_kind: implementation
round: 2
verdict: SHIP|FIX_BEFORE_SHIP|REWORK
reviewer_runtime: claude
reviewer_model: opus
generated_at: <ISO-8601 timestamp>

---

# Summary

Two sentences explaining the verdict.

# Blockers

Use `None.` when there are no blockers. Otherwise, use one item per blocker:

## B-NNN — <short title>

- File: <repo-root path>
- Line: <line number or null>
- Issue: <one paragraph>
- Rationale: <why this blocks shipment, with project rule/lesson reference>
- Suggested fix: <concrete change>

# Risks

Use `None.` when there are no risks. Otherwise, use one item per risk:

## R-NNN — <short title>

- File: <repo-root path>
- Line: <line number or null>
- Issue: <one paragraph>
- Suggested fix: <concrete change>

# Nits

Use `None.` when there are no nits. Otherwise, use one item per nit:

## N-NNN — <short title>

- File: <repo-root path>
- Line: <line number or null>
- Issue: <one line>
- Suggested fix: <one line>

# Evidence

List files read, tests/build evidence observed, and any limitations. Do not invent evidence.

# Deferred Or Follow-Up

List non-blocking follow-ups, or `None.`.
