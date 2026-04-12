# Channel Adapters - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Introduce channel core domain types and globaldb schema | completed | high | - |
| 02 | Implement channel registry and policy-driven routing | completed | high | task_01 |
| 03 | Add typed delivery targets and outbound target resolution seam | completed | medium | task_02 |
| 04 | Extend extension protocol, capabilities, and instance-scoped launch negotiation | completed | high | task_02, task_03 |
| 05 | Implement channel Host API ingest and instance state reporting | completed | high | task_02, task_04 |
| 06 | Build delivery broker and session-to-channel projection | completed | critical | task_02, task_03, task_04 |
| 07 | Compose channel manager and wire daemon boot lifecycle | completed | high | task_03, task_05, task_06 |
| 08 | Expose channel management over shared API contract, HTTP/UDS routes, and OpenAPI | completed | high | task_07 |
| 09 | Add CLI channel management commands | completed | medium | task_08 |
| 10 | Add per-instance channel observability and health reporting | completed | high | task_06, task_07, task_08 |
| 11 | Implement the Telegram reference adapter and adapter conformance harness | completed | critical | task_05, task_06, task_08, task_10 |
