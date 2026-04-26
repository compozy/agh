# Autonomous AGH - Task List

**GREENFIELD (alpha):** do not preserve legacy behavior for old local state. These tasks implement the local autonomy MVP from `_techspec.md` steps 1-10, then add the same QA planning/execution handoff pattern used by `.compozy/tasks/hermes`.

**MVP boundary:** tasks 01-16 implement the autonomy kernel. Tasks 17-18 prepare and execute QA. Post-MVP network evolution, broad memory scopes, self-correction telemetry, eval/replay, and broad web visibility remain follow-up TechSpecs unless explicitly pulled into scope later.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Autonomy Config Foundation | pending | medium | - |
| 02 | Agent Contract DTOs And OpenAPI Parity | pending | high | task_01 |
| 03 | Autonomy Hook Taxonomy And Task Hook Bridge | pending | high | task_01, task_02 |
| 04 | Situation Surface Providers | pending | high | task_01, task_02, task_03 |
| 05 | Agent Caller Identity Layer | pending | medium | task_01, task_02 |
| 06 | Agent Self And Channel Verbs | pending | high | task_04, task_05 |
| 07 | Task Claim Lease Schema | pending | critical | task_01, task_02, task_03 |
| 08 | ClaimNextRun And Lease Fencing Service | pending | critical | task_07 |
| 09 | Agent Task Lease API And CLI Verbs | pending | high | task_05, task_08 |
| 10 | Operator Start Publish Approval Execution Boundary | pending | high | task_08 |
| 11 | Mechanical Scheduler Sweep Notify | completed | high | task_03, task_04, task_08, task_09, task_10 |
| 12 | Session Lineage And Spawn Metadata | pending | high | task_01, task_02, task_03, task_05 |
| 13 | Safe Spawn API CLI And Reaper | pending | critical | task_12 |
| 14 | Coordinator Bootstrap And Restricted Orchestration | pending | critical | task_04, task_09, task_10, task_11, task_13 |
| 15 | Tasks UI Manual First Labels And E2E | pending | medium | task_10, task_14 |
| 16 | Runtime Autonomy Docs And CLI References | pending | medium | task_06, task_09, task_13, task_14, task_15 |
| 17 | Autonomy MVP QA Plan And Regression Artifacts | pending | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10, task_11, task_12, task_13, task_14, task_15, task_16 |
| 18 | Autonomy MVP QA Execution And End-to-End Validation | pending | critical | task_17 |
