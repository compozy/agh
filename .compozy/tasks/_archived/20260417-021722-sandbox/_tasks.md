# Execution Sandbox Abstraction + Daytona Provider — Task List

## Tasks

| #   | Title                                                             | Status    | Complexity | Dependencies              |
| --- | ----------------------------------------------------------------- | --------- | ---------- | ------------------------- |
| 01  | Core sandbox types, config profiles, and workspace resolution | completed | high       | —                         |
| 02  | Extract ACP Launcher and ToolHost interfaces                      | completed | critical   | task_01                   |
| 03  | Local provider implementation                                     | completed | medium     | task_02                   |
| 04  | Session sandbox integration and daemon wiring                 | completed | high       | task_01, task_03          |
| 05  | Validate Daytona SSH non-PTY transport                            | pending   | low        | —                         |
| 06  | Daytona SSH transport and provider implementation                 | pending   | critical   | task_03, task_04, task_05 |
| 07  | Daemon restart sandbox cleanup                                | completed | medium     | task_04                   |
| 08  | Sandbox extension hooks and Host API                          | completed | high       | task_04                   |
