# Agent Authored Context - Task List

**GREENFIELD (alpha):** do not preserve legacy behavior for old local state. These tasks implement the approved Soul plus Heartbeat aggregate TechSpec by building authored context foundations first, then runtime integration, agent-manageable surfaces, extensibility surfaces, docs, and final QA evidence.

The child specs remain normative for behavior: `_techspec_soul.md` governs `SOUL.md`, `_techspec_heartbeat.md` governs `HEARTBEAT.md` and session health, and `_techspec.md` governs sequencing, shared boundaries, and cross-feature verification.

## MVP Boundary

MVP includes optional managed `SOUL.md` persona authoring, optional managed `HEARTBEAT.md` wake-policy authoring, metadata-only session health, deterministic session/task provenance, CLI/HTTP/UDS/Host API manageability, extension/tool/resource visibility, config lifecycle, docs, and real QA. MVP excludes UI editors, independent heartbeat queues, session liveness implemented through authored Markdown, network greet changes, `ClaimNextRun` replacement, implicit parent soul inheritance, and legacy compatibility bridges.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Add Soul Config and Resolver Foundation | completed | high | - |
| 02 | Persist Soul Snapshots and Authoring Revisions | completed | high | task_01 |
| 03 | Implement Managed Soul Authoring Service | completed | high | task_01, task_02 |
| 04 | Integrate Soul With Sessions, Prompt Context, and Task Provenance | completed | critical | task_01, task_02, task_03 |
| 05 | Add Heartbeat Config and Policy Resolver Foundation | completed | high | - |
| 06 | Persist Heartbeat Snapshots, Revisions, Session Health, and Wake Audit | completed | critical | task_02, task_05 |
| 07 | Implement Metadata-Only Session Health | completed | critical | task_06 |
| 08 | Implement Managed Heartbeat Authoring and Status Services | completed | high | task_05, task_06, task_07 |
| 09 | Implement Heartbeat Wake Service and Scheduler Integration | completed | critical | task_05, task_07, task_08 |
| 10 | Add Shared API Contracts and Codegen Surface | completed | high | task_04, task_09 |
| 11 | Expose HTTP and UDS Routes Through Shared Core Handlers | completed | high | task_10 |
| 12 | Add Agent-Operable CLI Commands | completed | high | task_11 |
| 13 | Add Extension Host API, Hooks, Tools, Resources, and SDK Support | completed | high | task_10, task_11 |
| 14 | Regenerate Web and SDK Contract Consumers | completed | high | task_10, task_11, task_13 |
| 15 | Update Runtime, Config, Extension, and CLI Documentation | completed | high | task_12, task_13, task_14 |
| 16 | Agent Soul QA plan and regression artifacts | completed | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10, task_11, task_12, task_13, task_14, task_15 |
| 17 | Agent Soul QA execution and operator-flow validation | completed | critical | task_16 |
