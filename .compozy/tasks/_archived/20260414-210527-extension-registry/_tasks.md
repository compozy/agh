# Extension Registry — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Extract shared extraction logic and define registry interfaces | completed | high | — |
| 02 | Implement MultiRegistry and Installer pipeline | completed | high | task_01 |
| 03 | Implement ClawHub and GitHub adapters | completed | critical | task_01 |
| 04 | Add extension registry schema, config, and CLI commands | completed | critical | task_02, task_03 |
| 05 | Migrate skill CLI to MultiRegistry and remove marketplace package | completed | high | task_02, task_03 |
