# Web Redesign — Task List

## Tasks

| #  | Title                                                                 | Status  | Complexity | Dependencies                          |
|----|-----------------------------------------------------------------------|---------|------------|---------------------------------------|
| 01 | Extend tokens, install motion, add UIProvider                         | completed | low        | —                                     |
| 02 | Migrate shadcn batch 1 (Dialog, Popover, Sheet, Tooltip) to @agh/ui   | completed | medium     | task_01                               |
| 03 | Migrate shadcn batch 2 (Combobox, Command, Select, ScrollArea, Tabs) to @agh/ui | completed | medium     | task_01                               |
| 04 | Migrate shadcn batch 3 (DropdownMenu, Switch, Toggle, ToggleGroup, Accordion, Collapsible) to @agh/ui | completed | medium     | task_01                               |
| 05 | Add Sidebar + SplitPane primitives to @agh/ui                         | completed | high       | task_01                               |
| 06 | Add PageHeader, Pills, SearchInput, Empty, Section, Toolbar and migrate design-system primitives | completed | critical   | task_01                               |
| 07 | Add Metric, MonoBadge, KindChip, StatusDot, ConnectionIndicator to @agh/ui | completed | high       | task_01                               |
| 08 | Close web/src/components/ui/: migrate remaining shadcn + delete folder | completed | high       | task_01, task_02, task_03, task_04, task_05, task_06 |
| 09 | Add CodeBlock primitive                                               | completed | low        | task_01                               |
| 10 | Add ChatMessageBubble + ToolCallCard shells                           | completed | medium     | task_01                               |
| 11 | Wire Playwright visual snapshot harness for @agh/ui                   | completed | high       | task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10 |
| 12 | Write packages/ui contributor guide (README.md)                       | completed | low        | task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10, task_11 |
| 13 | Rewrite app-sidebar on @agh/ui Sidebar                                | completed | high       | task_05, task_08                      |
| 14 | Rewrite root layout + route-level motion                              | completed | high       | task_05, task_06                      |
| 15 | Rewrite /design-system showcase and delete design-system folder       | completed | medium     | task_06, task_07, task_14             |
| 16 | Wire Playwright visual snapshot baseline for web/                     | completed | medium     | task_11, task_13, task_14             |
| 17 | Rewrite Tasks domain list + detail panel                              | completed | high       | task_13, task_14                      |
| 18 | Rewrite Tasks domain Kanban, Dashboard, Inbox views                   | completed | high       | task_17                               |
| 19 | Rewrite Tasks domain forms and run detail route                       | completed | medium     | task_17                               |
| 20 | Rewrite Session domain message thread                                 | completed | high       | task_10, task_13, task_14             |
| 21 | Rewrite Session domain composer                                       | completed | medium     | task_20                               |
| 22 | Rewrite Session domain inspector panel                                | completed | medium     | task_20                               |
| 23 | Rewrite Network domain (channels + peers)                             | pending | high       | task_13, task_14                      |
| 24 | Rewrite Automation domain (jobs + triggers)                           | pending | high       | task_13, task_14                      |
| 25 | Rewrite Bridges domain (list + detail)                                | pending | medium     | task_13, task_14                      |
| 26 | Rewrite Knowledge domain (list + detail)                              | pending | medium     | task_13, task_14                      |
| 27 | Rewrite Skills domain (installed + marketplace)                       | pending | medium     | task_13, task_14                      |
| 28 | Rewrite Workspace onboarding and Agent sidebar integrations           | pending | medium     | task_13, task_14                      |
| 29 | Rewrite Daemon status and home dashboard                              | pending | low        | task_13, task_14                      |
| 30 | Rewrite Settings shell (save-bar, page-actions, restart-banner)       | pending | medium     | task_13, task_14                      |
| 31 | Rewrite Settings pages batch 1 (general, memory, skills, providers, automation) | pending | high       | task_30                               |
| 32 | Rewrite Settings pages batch 2 (mcp-servers, hooks-extensions, observability, environments, network) | pending | high       | task_30                               |
