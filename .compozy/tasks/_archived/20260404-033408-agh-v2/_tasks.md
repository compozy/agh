# AGH v2 — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|-----------|--------------|
| 00 | Project Scaffold — go.mod cleanup, magefile update, stubs | **completed** | low | — |
| 01 | Config Package | **completed** | medium | task_00 |
| 02 | Store Package | **completed** | medium | task_00 |
| 03 | ACP Package | **completed** | high | task_01 |
| 04 | Session Package | **completed** | high | task_01, task_02, task_03 |
| 05 | Observe Package | **completed** | medium | task_02, task_04 |
| 06 | Daemon Package | **completed** | medium | task_01–task_05 |
| 07 | UDS API Package | **completed** | medium | task_04, task_05, task_06 |
| 08 | CLI Package | **completed** | medium | task_07 |
| 09 | HTTP API Package | **completed** | medium | task_04, task_05, task_06 |
