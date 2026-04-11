# Automation System - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Introduce automation config and core domain types | pending | high | - |
| 02 | Add automation persistence and overlay storage in globaldb | pending | high | task_01 |
| 03 | Implement dispatcher, run recording, and execution governance | pending | high | task_02 |
| 04 | Build scheduler runtime for scheduled jobs | pending | high | task_03 |
| 05 | Build trigger engine with normalized ingress and webhook auth | pending | critical | task_03 |
| 06 | Compose automation manager and wire daemon boot lifecycle | pending | high | task_04, task_05 |
| 07 | Expose automation over core API, HTTP/UDS routes, and OpenAPI | pending | high | task_06 |
| 08 | Add CLI automation command group | pending | medium | task_07 |
| 09 | Add extension Host API automation methods and automation hook events | pending | high | task_06 |
| 10 | Build web automation management UI | pending | high | task_07 |
