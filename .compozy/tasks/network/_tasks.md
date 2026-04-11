# AGH Network - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Network protocol core | completed | high | - |
| 02 | Transport, config, and audit foundation | completed | high | task_01 |
| 03 | Presence registry and router | completed | high | task_01, task_02 |
| 04 | Session space opt-in and metadata | completed | medium | task_01 |
| 05 | Prompt provenance and ACP guardrails | completed | high | task_04 |
| 06 | Inbound delivery workers and turn-end handoff | completed | high | task_03, task_05 |
| 07 | Network manager and daemon boot integration | completed | high | task_02, task_03, task_04, task_06 |
| 08 | Network CLI/API surface and observability | completed | medium | task_07 |
| 09 | Bundled agh-network skill and prompt injection | completed | medium | task_04, task_08 |
| 10 | Operational hardening and reliability sweep | completed | high | task_05, task_06, task_07, task_08, task_09 |
