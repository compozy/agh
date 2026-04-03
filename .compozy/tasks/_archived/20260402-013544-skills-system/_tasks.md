# Skills System — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | [Skills package core](task_01.md) | completed | medium | none |
| 02 | [Registry, catalog builder and bundled skills](task_02.md) | completed | medium | task_01 |
| 03 | [Kernel integration](task_03.md) | completed | medium | task_02 |
| 04 | [CLI commands and ClawHub client](task_04.md) | completed | high | task_02 |

## Dependency Graph

```
task_01 (types + loader + verify + eligibility)
    │
    ▼
task_02 (registry + catalog + bundled)
    │
    ├──▶ task_03 (kernel boot + spawn + prompt)
    │
    └──▶ task_04 (CLI commands + ClawHub)
```

## References

- [TechSpec](_techspec.md)
- [ADR-001: Prompt-Only Runtime](adrs/adr-001.md)
- [ADR-002: SKILL.md Native Format](adrs/adr-002.md)
- [ADR-003: System Prompt + CLI Access](adrs/adr-003.md)
- [ADR-004: Four-Level Loading Hierarchy](adrs/adr-004.md)
