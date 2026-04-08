# Skills System v2 — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Extend types with MCPServerDecl, HookDecl, Provenance, and SourceMarketplace | completed | low | — |
| 02 | Extend config with MarketplaceConfig and merge plumbing | pending | medium | — |
| 03 | Parse metadata.agh fields in loader | completed | medium | task_01 |
| 04 | Implement MCPResolver with trust-tier filtering | completed | medium | task_01 |
| 05 | Implement HookRunner subprocess dispatch | completed | medium | task_01 |
| 06 | Implement Provenance hash verification and sidecar | completed | medium | task_01 |
| 07 | Extend Registry with marketplace source, provenance, and quarantine | completed | high | task_03, task_06 |
| 08 | Implement marketplace interface and ClawHub client | completed | medium | task_02 |
| 09 | Integrate MCP + hooks into daemon boot and session manager | completed | high | task_04, task_05, task_07 |
| 10 | Add marketplace CLI commands (search/install/remove/update) | completed | medium | task_07, task_08, task_02 |
