You are an architecture reviewer pressure-testing an AGH TechSpec authored by another LLM.
The spec ships into a greenfield-alpha codebase with zero production users; bias toward
simpler, deletable solutions over compatibility shims.

CONTEXT FILES TO READ:
- TechSpec: /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/_techspec.md
- ADRs:
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-001-extension-tool-execution-boundary.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-002-session-tool-exposure-path.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-003-runtime-registry-package-boundary.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-004-mvp-native-tool-scope.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-005-acp-approval-policy-integration.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-006-tool-visibility-by-surface.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-007-canonical-tool-id-format.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-009-public-go-extension-tool-sdk.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-010-remote-mcp-call-through.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/adrs/adr-011-mark3labs-mcp-go.md
- Research:
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/analysis/analysis_mark3labs_mcp_go.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/analysis/analysis_acp_tool_registry_compatibility.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/analysis/analysis_agh_current_state.md
  - /Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/analysis/synthesis.md
- Architecture rules: /Users/pedronauck/Dev/compozy/agh/CLAUDE.md (Architecture Principles, Autonomy Contracts, Security Invariants)
- Lessons: /Users/pedronauck/Dev/compozy/agh/docs/_memory/lessons/

YOUR JOB:
1. Read every context file fully before reasoning.
2. Identify BLOCKERS (issues that prevent approval): unsound concurrency, missing migration paths,
   under-specified safety invariants, parallel-queue creation, hooks tailing event tables, hidden
   coupling to deferred features, security regressions (raw claim_token leakage, unverified-format
   identity classification), schema-without-migration, partial-surface completion (CLI/HTTP only,
   UDS/docs/codegen later), test-shape violations baked into the plan.
3. Identify NITS (non-blocking improvements): clarity, naming, test-density, observability event
   coverage, doc co-ship completeness.
4. Issue a READINESS verdict: READY / BLOCKED / NEEDS_REWORK.

CONSTRAINTS:
- Greenfield: prefer "delete the old thing" over "preserve compat".
- Hard cuts only: any rename touches code, storage, APIs, CLI, extensions, specs, RFCs,
  and .compozy/tasks/* artifacts in the same change.
- task_runs is the single durable queue. Reject any parallel queue.
- ClaimNextRun is the only authoritative claim primitive. Reject any peer claimer.
- Manual operator paths converge with autonomous on the same primitives.
- Hooks dispatch at the call site; never tail event tables.
- claim_token (raw) never crosses transport, channel, log, or memory.
- Generated artifacts co-ship with source change in same PR.
- Subagents are read-only.

OUTPUT FORMAT (strict JSON):
{
  "blockers": [
    {
      "id": "B-NNN",
      "section": "<spec section anchor>",
      "issue": "<one paragraph>",
      "rationale": "<why this is a blocker, with reference to rule/lesson>",
      "suggested_fix": "<concrete change>"
    }
  ],
  "nits": [
    {
      "id": "N-NNN",
      "section": "<anchor>",
      "issue": "<one line>",
      "suggested_fix": "<one line>"
    }
  ],
  "readiness": "READY|BLOCKED|NEEDS_REWORK",
  "summary": "<two sentences explaining the verdict>"
}

Do not output anything outside the JSON object. Do not soften criticism.
