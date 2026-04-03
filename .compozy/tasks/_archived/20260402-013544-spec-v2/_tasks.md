# AGH Agent Network Framework — M1 Task List

## Source Documents

- PRD/TechSpec: `docs/spec-v2/00-executive-summary.md` through `docs/spec-v2/16-web-dashboard.md`
- Development Sequence: `docs/spec-v2/12-development-sequence.md`
- Multi-Session Design: `docs/plans/2026-03-30-multi-session-design.md`
- POC Reference: `_poc-project/`

## Tasks

| #   | Title                                               | Status  | Complexity | Dependencies                                |
| --- | --------------------------------------------------- | ------- | ---------- | ------------------------------------------- |
| 01  | [Project Scaffolding & Core Types](task_01.md)      | completed | low        | none                                      |
| 02  | [Configuration System](task_02.md)                  | completed | medium     | task_01                                     |
| 03  | [SQLite State Layer](task_03.md)                    | completed | medium     | task_01                                     |
| 04  | [NATS Embedded Transport](task_04.md)               | completed | medium     | task_01                                     |
| 05  | [Registry](task_05.md)                              | completed | low        | task_01, task_02, task_03                   |
| 06  | [Ring Buffer & PTY Manager](task_06.md)             | completed | medium     | task_01                                     |
| 07  | [Claude Code Driver](task_07.md)                    | completed | medium     | task_01, task_06                            |
| 08  | [Multi-Session Rework](task_08.md)                  | completed | medium     | task_01–07                                  |
| 09  | [Prompt Assembler](task_09.md)                      | completed | medium     | task_01, task_05                            |
| 10  | [TOON Renderer](task_10.md)                         | completed | low        | task_03                                     |
| 11  | [Hook System](task_11.md)                           | completed | high       | task_04, task_06, task_07                   |
| 12  | [Workgroup Management](task_12.md)                  | completed | medium     | task_05, task_06, task_11                   |
| 13  | [Error Handling & Resilience](task_13.md)           | completed | medium     | task_05, task_06                            |
| 14  | [Kernel Boot & Shutdown Orchestration](task_14.md)  | completed | medium     | task_03, task_04, task_05, task_06, task_08, task_13 |
| 15  | [SessionManager](task_15.md)                        | completed | medium     | task_08, task_14                            |
| 16  | [Daemon & Session CLI](task_16.md)                  | completed | medium     | task_14, task_15                            |
| 17  | [CLI Commands — Session & Runtime](task_17.md)      | completed | medium     | task_04, task_05, task_06, task_07, task_14, task_15, task_16 |
| 18  | [CLI Commands — Messaging & State](task_18.md)      | completed | medium     | task_04, task_05, task_10, task_17          |
| 19  | [CLI Commands — Workgroups & Discovery](task_19.md) | completed | medium     | task_12, task_17                            |
| 20  | [CLI Commands — Lifecycle & Hooks](task_20.md)      | completed | low        | task_17, task_18                            |
| 21  | [Meta-Learning](task_21.md)                         | completed | medium     | task_02, task_19                            |
| 22  | [Web Server & WebSocket](task_22.md)                | completed | medium     | task_03, task_05, task_06                   |
| 23  | [Dashboard Frontend](task_23.md)                    | completed | high       | task_22                                     |
| 24  | [Codex Driver](task_24.md)                          | completed | medium     | task_01, task_06                            |
| 25  | [OpenCode Driver](task_25.md)                       | completed | medium     | task_01, task_06                            |
| 26  | [Pi Driver](task_26.md)                             | completed | medium     | task_01, task_06                            |

## Dependency Graph

```
01 ──→ 02 ──→ 05 ──→ 09
 │      │      │
 │      │      ├──→ 12 ──→ 19
 │      │      │
 │      ├──→ 21
 │      │
 ├──→ 03 ──→ 05
 │      ├──→ 10
 │      ├──→ 14
 │      └──→ 22
 │
 ├──→ 04 ──→ 11 ──→ 12
 │      ├──→ 14
 │      ├──→ 17 ──→ 18 ──→ 20
 │      │    │      │
 │      │    ├──→ 19
 │      │    └──→ 20
 │      └──→ 22
 │
 ├──→ 06 ──→ 07 ──→ 11
 │      ├──→ 11
 │      ├──→ 12
 │      ├──→ 13 ──→ 14
 │      ├──→ 14
 │      ├──→ 17
 │      ├──→ 22 ──→ 23
 │      ├──→ 24
 │      ├──→ 25
 │      └──→ 26
 │
 ├──→ 24 (Codex Driver)
 ├──→ 25 (OpenCode Driver)
 └──→ 26 (Pi Driver)

Multi-Session Chain:
01–07 ──→ 08 (Multi-Session Rework)
              ├──→ 14 (Kernel Boot — daemon architecture)
              │    ├──→ 15 (SessionManager)
              │    │    └──→ 16 (Daemon & Session CLI)
              │    ├──→ 16
              │    └──→ 17
              └──→ 15
```

## Parallelization Guide

| Phase | Tasks                                                         | Notes                                   |
| ----- | ------------------------------------------------------------- | --------------------------------------- |
| 1     | task_01                                                       | Foundation — everything depends on this |
| 2     | task_02, task_03, task_04, task_06                            | All independent, start after 01         |
| 3     | task_05, task_07, task_10, task_13, task_24, task_25, task_26 | Mix of kernel + drivers                 |
| 4     | task_08, task_09, task_11, task_12, task_22                   | Multi-session rework + kernel wiring    |
| 5     | task_14, task_23                                              | Kernel boot (daemon) + dashboard        |
| 6     | task_15, task_17                                              | SessionManager + CLI entry points       |
| 7     | task_16, task_18, task_19                                     | Daemon CLI + remaining CLI commands     |
| 8     | task_20, task_21                                              | Final CLI + meta-learning               |
