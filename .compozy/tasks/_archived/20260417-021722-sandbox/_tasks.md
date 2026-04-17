# Execution Environment Abstraction + Daytona Provider — Task List

## Tasks

| #   | Title                                                             | Status    | Complexity | Dependencies              |
| --- | ----------------------------------------------------------------- | --------- | ---------- | ------------------------- |
| 01  | Core environment types, config profiles, and workspace resolution | completed | high       | —                         |
| 02  | Extract ACP Launcher and ToolHost interfaces                      | completed | critical   | task_01                   |
| 03  | Local provider implementation                                     | completed | medium     | task_02                   |
| 04  | Session environment integration and daemon wiring                 | completed | high       | task_01, task_03          |
| 05  | Validate Daytona SSH non-PTY transport                            | pending   | low        | —                         |
| 06  | Daytona SSH transport and provider implementation                 | pending   | critical   | task_03, task_04, task_05 |
| 07  | Daemon restart environment cleanup                                | completed | medium     | task_04                   |
| 08  | Environment extension hooks and Host API                          | completed | high       | task_04                   |
