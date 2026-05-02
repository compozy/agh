# Agent Authored Context - Task List

**GREENFIELD (alpha):** do not preserve legacy behavior for old local state. These tasks implement the approved Soul plus Heartbeat aggregate TechSpec by building authored context foundations first, then runtime integration, agent-manageable surfaces, extensibility surfaces, docs, and final QA evidence.

The child specs remain normative for behavior: `_techspec_soul.md` governs `SOUL.md`, `_techspec_heartbeat.md` governs `HEARTBEAT.md` and session health, and `_techspec.md` governs sequencing, shared boundaries, and cross-feature verification.

## MVP Boundary

MVP includes optional managed `SOUL.md` persona authoring, optional managed `HEARTBEAT.md` wake-policy authoring, metadata-only session health, deterministic session/task provenance, CLI/HTTP/UDS/Host API manageability, extension/tool/resource visibility, config lifecycle, docs, and real QA. MVP excludes UI editors, independent heartbeat queues, session liveness implemented through authored Markdown, network greet changes, `ClaimNextRun` replacement, implicit parent soul inheritance, and legacy compatibility bridges.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Add Soul Config and Resolver Foundation | pending | high | - |
| 02 | Persist Soul Snapshots and Authoring Revisions | pending | high | task_01 |
| 03 | Implement Managed Soul Authoring Service | pending | high | task_01, task_02 |
| 04 | Integrate Soul With Sessions, Prompt Context, and Task Provenance | completed | critical | task_01, task_02, task_03 |
| 05 | Add Heartbeat Config and Policy Resolver Foundation | pending | high | - |
| 06 | Persist Heartbeat Snapshots, Revisions, Session Health, and Wake Audit | pending | critical | task_02, task_05 |
| 07 | Implement Metadata-Only Session Health | pending | critical | task_06 |
| 08 | Implement Managed Heartbeat Authoring and Status Services | pending | high | task_05, task_06, task_07 |
| 09 | Implement Heartbeat Wake Service and Scheduler Integration | pending | critical | task_05, task_07, task_08 |
| 10 | Add Shared API Contracts and Codegen Surface | pending | high | task_04, task_09 |
| 11 | Expose HTTP and UDS Routes Through Shared Core Handlers | pending | high | task_10 |
| 12 | Add Agent-Operable CLI Commands | pending | high | task_11 |
| 13 | Add Extension Host API, Hooks, Tools, Resources, and SDK Support | pending | high | task_10, task_11 |
| 14 | Regenerate Web and SDK Contract Consumers | pending | high | task_10, task_11, task_13 |
| 15 | Update Runtime, Config, Extension, and CLI Documentation | pending | high | task_12, task_13, task_14 |
| 16 | QA Plan and Test Coverage | pending | high | task_15 |
| 17 | Real-Scenario QA Execution | pending | critical | task_16 |
