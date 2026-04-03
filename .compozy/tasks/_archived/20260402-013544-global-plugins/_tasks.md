# Global Plugins and Zero Workdir Pollution — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | [Plugin assets and embed package](task_01.md) | completed | medium | none |
| 02 | [AGH_* env vars and remove BuildHookConfig from interface](task_02.md) | completed | medium | none |
| 03 | [All drivers: zero workdir writes](task_03.md) | completed | high | task_01, task_02 |
| 04 | [Install/Uninstall plugin lifecycle](task_04.md) | completed | high | task_01, task_03 |

## Dependency Graph

```
task_01 (plugin assets + embed)     task_02 (env vars + interface cleanup)
    │                                   │
    └───────────────┬───────────────────┘
                    ▼
                task_03 (all 4 drivers: zero workdir writes)
                    │
                    ▼
                task_04 (install/uninstall CLI)
```

## References

- [TechSpec](_techspec.md)
- [ADR-001: Global Plugins Over Per-Session File Generation](adrs/adr-001.md)
- [ADR-002: Hybrid Plugin Strategy](adrs/adr-002.md)
- [ADR-003: AGH_* Environment Variable Prefix Standardization](adrs/adr-003.md)
- [ADR-004: Remove BuildHookConfig from AgentDriver Interface](adrs/adr-004.md)
- [ADR-005: Zero Workdir Pollution via CLI Flags and Environment Variables](adrs/adr-005.md)
