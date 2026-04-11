# Channel Adapters - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Introduce channel core domain types and globaldb schema | pending | high | - |
| 02 | Implement channel registry and policy-driven routing | pending | high | task_01 |
| 03 | Add typed delivery targets and outbound target resolution seam | pending | medium | task_02 |
| 04 | Extend extension protocol, capabilities, and instance-scoped launch negotiation | pending | high | task_02, task_03 |
| 05 | Implement channel Host API ingest and instance state reporting | pending | high | task_02, task_04 |
| 06 | Build delivery broker and session-to-channel projection | pending | critical | task_02, task_03, task_04 |
| 07 | Compose channel manager and wire daemon boot lifecycle | pending | high | task_03, task_05, task_06 |
| 08 | Expose channel management over shared API contract, HTTP/UDS routes, and OpenAPI | pending | high | task_07 |
| 09 | Add CLI channel management commands | pending | medium | task_08 |
| 10 | Add per-instance channel observability and health reporting | pending | high | task_06, task_07, task_08 |
| 11 | Implement the Telegram reference adapter and adapter conformance harness | pending | critical | task_05, task_06, task_08, task_10 |
