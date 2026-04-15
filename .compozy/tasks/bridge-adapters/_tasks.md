# Bridge Adapters - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Extend bridge core models, persistence, and provider manifests | pending | critical | - |
| 02 | Redesign provider-scoped bridge runtime handshake and daemon lifecycle | pending | critical | task_01 |
| 03 | Expand bridge v1 event and delivery contracts | pending | high | task_01 |
| 04 | Implement provider-scoped Host API instance management and authorization | pending | critical | task_01, task_02, task_03 |
| 05 | Build shared internal/bridgesdk runtime core and ingress hardening | pending | critical | task_02, task_03, task_04 |
| 06 | Expose provider metadata and provider_config through shared bridge APIs and OpenAPI | pending | high | task_01 |
| 07 | Update web bridge management for provider config, secret slots, and DM policy | pending | high | task_06 |
| 08 | Replace the Telegram reference path with a provider-scoped conformance harness | pending | high | task_02, task_04, task_05 |
| 09 | Implement the Telegram provider extension | pending | high | task_05, task_08 |
| 10 | Implement the Slack provider extension | pending | high | task_05, task_08 |
| 11 | Implement the Discord provider extension | pending | high | task_05, task_08 |
| 12 | Implement the WhatsApp provider extension | pending | high | task_05, task_08 |
| 13 | Implement the Microsoft Teams provider extension | pending | high | task_05, task_08 |
| 14 | Implement the Google Chat provider extension | pending | high | task_05, task_08 |
| 15 | Implement the GitHub provider extension | pending | high | task_05, task_08 |
| 16 | Implement the Linear provider extension | pending | high | task_05, task_08 |
| 17 | Add cross-provider multi-instance recovery and conformance coverage | pending | critical | task_09, task_10, task_11, task_12, task_13, task_14, task_15, task_16 |
