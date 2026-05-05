# Hermes Hardening - Task List

**GREENFIELD (alpha):** do not preserve legacy behavior for old local state. These tasks harden AGH against the selected Hermes gaps by building durable foundations first, then domain hardening tracks, then QA planning and execution.

Selected issues: 10, 11, 14, 15, 16, 17, 20, 21, 22, 25, 27, 28, 29, 30, 33, 34, 35, 36, 37, 39, 40, 41, 42, 43, 57, 59, 60.

Excluded issues: 6, 8, 9.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Persistence and Retry Foundations | completed | critical | - |
| 02 | Observability Retention and Health Base | completed | high | task_01 |
| 03 | ACP and Session Lifecycle Hardening | completed | critical | task_01, task_02 |
| 04 | Durable Automation Scheduler | completed | critical | task_01 |
| 05 | MCP Auth and Skill Security | completed | critical | task_01 |
| 06 | Tool Process Registry and Interrupts | completed | critical | task_01 |
| 07 | Memory Visibility and Future Interfaces | completed | high | task_01, task_02 |
| 08 | CLI Config and Setup Lifecycle | completed | high | task_01, task_05 |
| 09 | Environment, Extension, and Release Hardening | completed | high | task_08 |
| 10 | Hermes Hardening QA Plan and Regression Artifacts | completed | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09 |
| 11 | Hermes Hardening QA Execution and End-to-End Validation | completed | critical | task_10 |
