# Third-Party Integration Extensions for AGH

Research date: 2026-04-11

This document catalogs communication, productivity, and business tool integrations that could be built as AGH extensions. Each integration leverages AGH's subprocess JSON-RPC extension model with Host API access to session/memory/skills/observe operations.

---

## Master Integration Table

| Category           | Integration      | MCP Server Exists?         | Server Source                                          | Maturity   | AGH Priority |
| ------------------ | ---------------- | -------------------------- | ------------------------------------------------------ | ---------- | ------------ |
| **Communication**  | Slack            | Yes (Official + Community) | Slack official (47 tools), korotovsky/slack-mcp-server | Production | High         |
|                    | Discord          | Yes (Community)            | Multiple community servers                             | Stable     | Medium       |
|                    | Microsoft Teams  | Yes (Official)             | Microsoft Work IQ Teams MCP                            | Production | Medium       |
|                    | Telegram         | Yes (Community)            | overpod, sparfenyuk, kfastov servers                   | Stable     | Medium       |
|                    | WhatsApp         | Yes (Community)            | lharries/whatsapp-mcp, Sinch MCP                       | Beta       | Low          |
|                    | Email (Gmail)    | Yes (Multiple)             | taylorwilsdon/google_workspace_mcp                     | Production | High         |
|                    | Email (Outlook)  | Yes (Official + Community) | Microsoft Work IQ Mail, Softeria ms-365-mcp            | Production | High         |
|                    | Matrix           | No                         | Needs to be built                                      | N/A        | Low          |
| **Productivity**   | Google Workspace | Yes (Multiple)             | taylorwilsdon/google_workspace_mcp (100+ tools)        | Production | High         |
|                    | Microsoft 365    | Yes (Official + Community) | Work IQ servers, Softeria, PnP CLI                     | Production | High         |
|                    | Notion           | Yes (Official)             | Notion official MCP                                    | Production | High         |
|                    | Obsidian         | Yes (Community)            | mcpvault, cyanheads, MarkusPfundstein (60+ servers)    | Stable     | Medium       |
|                    | Roam Research    | No                         | Needs to be built                                      | N/A        | Low          |
| **Project Mgmt**   | Linear           | Yes (Community)            | jerhadf/linear-mcp-server, DX Heroes                   | Stable     | High         |
|                    | Jira             | Yes (Official + Community) | Atlassian Rovo MCP (OAuth 2.1), 74+ servers            | Production | High         |
|                    | Asana            | Yes (Official)             | Official at mcp.asana.com, roychri community           | Production | Medium       |
|                    | Monday.com       | Yes (Official)             | mondaycom/mcp (GraphQL API)                            | Production | Medium       |
|                    | Trello           | Yes (Community)            | delorenj/mcp-server-trello, Composio                   | Stable     | Low          |
|                    | ClickUp          | Yes (Community)            | taazkareem/clickup-mcp-server                          | Stable     | Medium       |
|                    | Basecamp         | Yes (Community)            | georgeantonopoulos/Basecamp-MCP-Server (75 tools)      | Stable     | Low          |
|                    | Shortcut         | Yes (Official)             | Official hosted server (OAuth)                         | Production | Medium       |
| **Knowledge Base** | Confluence       | Yes (Official)             | Atlassian Rovo MCP                                     | Production | High         |
|                    | GitBook          | Yes (Auto-generated)       | Auto-generated per docs site, MCPBook                  | Production | Medium       |
|                    | ReadMe           | Yes                        | MCP server + llms.txt                                  | Stable     | Low          |
|                    | Mintlify         | Yes (Auto-generated)       | Auto-generated per docs site (free)                    | Production | Low          |
| **CRM/Sales**      | Salesforce       | Yes (Official)             | Salesforce Agentforce MCP (60+ tools)                  | Production | Medium       |
|                    | HubSpot          | Yes (Community)            | Community server (116 stars, FAISS search)             | Stable     | Medium       |
|                    | Pipedrive        | Yes (Community)            | iamsamuelfraga/mcp-pipedrive                           | Stable     | Low          |
|                    | Attio            | Yes (Official)             | Official at mcp.attio.com + kesslerio community        | Production | Low          |
| **Design**         | Figma            | Yes (Official)             | Official at mcp.figma.com (Code Connect)               | Production | High         |
|                    | Miro             | Yes (Official)             | Official Miro MCP Server                               | Beta       | Medium       |
|                    | Excalidraw       | Yes (Official + Community) | excalidraw/excalidraw-mcp, yctimlin                    | Stable     | Medium       |

---

## Detailed Integration Descriptions

---

### 1. Communication Platforms

#### 1.1 Slack

**What it does:** Full Slack workspace integration -- read channels, search message history, send messages, manage canvases, thread conversations, and manage users.

**MCP servers available:**

- **Official Slack MCP Server** (GA February 2026): 47 tools, OAuth authentication, enterprise-grade security. Supports DMs, group DMs, channels, threads, and canvases.
- **korotovsky/slack-mcp-server** (Community): Supports Stdio/SSE/HTTP transports, proxy settings, DMs, Group DMs, smart history fetch. 9,000+ active users, 30,000+ monthly visitors.

**AGH agent use cases:**

- **Codebase Q&A bot:** Agent monitors a `#help-engineering` channel, detects questions about the codebase, researches the answer using workspace memory and file access, then posts a threaded reply with code references.
- **Standup summarizer:** Agent reads daily standup messages from a channel, consolidates blockers and progress across team members, writes a summary to Notion or Confluence.
- **Incident coordinator:** Agent detects urgent messages in `#incidents`, creates a Linear/Jira ticket, spins up a dedicated channel, posts runbook steps from the knowledge base.
- **PR review notifier:** Agent watches for merged PRs via GitHub events, summarizes changes, and posts a digest to `#releases` with links and context.

**Status:** Production-ready. Official server is the recommended path for enterprise use.

---

#### 1.2 Discord

**What it does:** Discord server management, channel operations, message sending/reading, forum post creation, and reaction handling.

**MCP servers available:**

- Multiple community servers providing channel management, message operations, and server administration.
- Unified notification servers that span Slack, Discord, and Telegram from a single interface.

**AGH agent use cases:**

- **Community support bot:** Agent monitors Discord support channels, searches documentation and past issues, provides answers to community questions.
- **Release announcer:** Agent publishes release notes to Discord announcement channels when a new version is tagged.
- **Meeting scheduler:** Agent coordinates availability across Discord threads and creates calendar events.

**Status:** Community-maintained. No official Discord MCP server yet -- multiple stable community implementations exist.

---

#### 1.3 Microsoft Teams

**What it does:** Search messages, manage chats/channels, send messages, create group chats, handle user/team operations. Uses Microsoft Graph API with device code authentication.

**MCP servers available:**

- **Microsoft Work IQ Teams MCP** (Official): Create, update, delete chats; add members; post messages; channel operations. Requires Microsoft 365 Copilot license for full features.
- **Community servers** on mcpservers.org (InditexTech).

**AGH agent use cases:**

- **Meeting prep agent:** Before a scheduled meeting, agent pulls relevant documents from SharePoint, recent email threads from Outlook, and Jira ticket updates, then posts a briefing to the Teams channel.
- **Cross-platform sync:** Agent mirrors important decisions from Teams channels to Slack or Linear for engineering teams that use different tools.

**Status:** Production-ready via Microsoft's official Work IQ servers.

---

#### 1.4 Telegram

**What it does:** Full Telegram access via MTProto protocol or Bot API -- messaging, chat management, media tools, file sending/downloading.

**MCP servers available:**

- **overpod/telegram** (MTProto): Full account access, messaging, chat management, media tools.
- **sparfenyuk/mcp-telegram**: User client API integration with AI assistants.
- **kfastov/telegram-mcp-server**: Bot API integration with 35 tools.
- **qpd-v/telegram-communicator**: Multi-account support, tiered permissions (March 2026).

**AGH agent use cases:**

- **Ops notification pipeline:** Agent sends deployment status, health alerts, and CI/CD results to a Telegram ops group.
- **Personal assistant:** Agent receives natural language commands via Telegram DM, executes tasks (file lookups, scheduling, code searches), and replies with results.

**Status:** Multiple stable community servers. Bot API path is simpler; MTProto path gives full account access.

---

#### 1.5 WhatsApp

**What it does:** Send and receive WhatsApp messages via WhatsApp Web multi-device API or business APIs.

**MCP servers available:**

- **lharries/whatsapp-mcp**: WhatsApp Web integration for personal accounts.
- **Sinch MCP** (February 2026): Unified messaging across SMS, WhatsApp, RCS, and email.
- **Unipile MCP**: Unified access across LinkedIn, WhatsApp, Instagram, Messenger.

**AGH agent use cases:**

- **Customer response agent:** Agent receives WhatsApp business messages, looks up customer info in CRM, drafts contextual replies.
- **Appointment reminder:** Agent sends appointment reminders via WhatsApp based on calendar events.

**Status:** Beta. WhatsApp's API restrictions make this more complex than other messaging platforms.

---

#### 1.6 Email (Gmail + Outlook)

**What it does:** Send, read, search, label, filter, forward, reply to emails. Manage attachments, calendar invites, and threading.

**MCP servers available:**

- **Gmail via Google Workspace MCP** (taylorwilsdon): OAuth 2.1, multi-user, 100+ tools across Gmail/Calendar/Drive/Docs/Sheets.
- **Microsoft Work IQ Mail** (Official): Create, update, delete messages; reply/reply-all; semantic search.
- **Softeria ms-365-mcp-server**: Graph API integration for Outlook mail, calendar, files.
- **Generic IMAP/SMTP MCP**: Works with any email provider -- Gmail, Outlook, Yahoo, Fastmail.

**AGH agent use cases:**

- **Email triage agent:** Agent reads incoming emails, categorizes by urgency and topic, drafts replies for routine queries, escalates important ones to Slack.
- **Follow-up tracker:** Agent monitors sent emails, detects when no reply is received after N days, creates follow-up tasks in Linear.
- **Report distributor:** Agent generates weekly status reports from project management data, formats as email, sends to stakeholder distribution list.

**Status:** Production-ready. Multiple mature options for both Gmail and Outlook.

---

#### 1.7 Matrix

**What it does:** Would provide integration with the Matrix decentralized communication protocol (Element, etc.).

**MCP servers available:** None found in current directories.

**AGH agent use cases:**

- **Bridge bot:** Agent monitors Matrix rooms and cross-posts to Slack/Discord channels.
- **Self-hosted comms agent:** For organizations using Matrix for privacy, agent provides the same Q&A and automation capabilities as Slack integrations.

**Status:** Needs to be built. Opportunity for AGH to be first-mover on Matrix MCP.

---

### 2. Productivity Suites

#### 2.1 Google Workspace (Docs, Sheets, Calendar, Drive)

**What it does:** Full CRUD across Gmail, Calendar, Docs, Sheets, Slides, Drive, Forms, Tasks, Chat, and Search. OAuth 2.1 authentication with multi-user support.

**MCP servers available:**

- **taylorwilsdon/google_workspace_mcp** (workspacemcp.com): The most complete server -- 12 services, 100+ tools, OAuth 2.1, remote multi-user auth, DXT installer. v1.15.0 on PyPI (March 2026).
- **aaronsb/google-workspace-mcp**: Gmail, Calendar, Drive with authenticated access.
- **goncaloreis/google-workspace-mcp**: Docs, Sheets, Gmail tools for Claude Desktop.
- **Google official MCP** (December 2025): Official support for Google and Google Cloud services.

**AGH agent use cases:**

- **Meeting notes agent:** Agent joins a Google Meet (via Calendar), records action items, creates a Google Doc summary, assigns tasks in Linear, and sends follow-up emails.
- **Spreadsheet analyst:** Agent reads data from Google Sheets, performs analysis, writes insights back, and creates a Slides presentation with charts.
- **Calendar optimizer:** Agent analyzes calendar for meeting overload, suggests consolidated meetings, blocks focus time, and manages RSVPs.
- **Document drafter:** Agent pulls context from codebase and memory, drafts technical docs in Google Docs, shares with reviewers.

**Status:** Production-ready. taylorwilsdon server is the community standard with official Google backing emerging.

---

#### 2.2 Microsoft 365 (Word, Excel, PowerPoint, OneDrive, SharePoint)

**What it does:** Full read/write across Outlook, Teams, SharePoint, OneDrive, Word, Excel, PowerPoint. Enterprise authentication via Microsoft Graph API.

**MCP servers available:**

- **Microsoft Work IQ** (Official): Mail, Calendar, SharePoint, OneDrive, Teams servers. Requires M365 Copilot license.
- **Softeria/ms-365-mcp-server**: Graph API integration for mail, files, calendar, Excel, OneNote, To Do, Planner, Contacts.
- **PnP/cli-microsoft365-mcp-server**: Natural language to CLI commands across Entra ID, OneDrive, Outlook, Planner, Power Apps, SharePoint, Teams.
- **Arcade.dev Office 365 MCP**: Five servers for Word, Excel, PowerPoint, OneDrive, SharePoint with full read/write.

**AGH agent use cases:**

- **Onboarding automator:** Agent sets up new hire in SharePoint, creates OneDrive folders, sends welcome email via Outlook, schedules intro meetings in Calendar.
- **Report generator:** Agent pulls data from multiple sources, generates formatted Excel report, creates PowerPoint summary, distributes via email.
- **Document reviewer:** Agent reads Word documents from SharePoint, provides feedback, tracks changes, and notifies authors.

**Status:** Production-ready. Multiple options from Microsoft official to community.

---

#### 2.3 Notion

**What it does:** Search pages, create/update notes, manage databases, organize knowledge bases, query structured data.

**MCP servers available:**

- **Notion official MCP**: Direct integration documented at developers.notion.com.
- **Grey-Iris community server**: Markdown-first integration using 92% fewer tokens than official server.
- **WayStation**: Unified server connecting Notion, Monday, Asana, Slack.
- **Pipedream**: Hosted MCP for 2,500+ APIs including Notion.

**AGH agent use cases:**

- **Knowledge base maintainer:** Agent monitors codebase changes, automatically updates relevant Notion docs with new API signatures, configuration changes, or architectural decisions.
- **Sprint planning assistant:** Agent reads Notion sprint board, cross-references with Linear tickets, identifies gaps, and creates missing tasks.
- **Research compiler:** Agent collects information from web searches, codebase analysis, and memory, then compiles structured research pages in Notion.

**Status:** Production-ready. Official and community servers both mature.

---

#### 2.4 Obsidian

**What it does:** Read, write, search, and manage notes in Obsidian vaults. Frontmatter extraction, tag management, full-text search, bidirectional link support.

**MCP servers available:**

- **bitbonsai/mcpvault** (v0.11.0, March 2026): Vault access with path traversal protection, tag scanning, symlink safety.
- **cyanheads/obsidian-mcp-server**: Comprehensive tools via Obsidian Local REST API plugin.
- **MarkusPfundstein/mcp-obsidian**: REST API integration.
- **boazy/notes-mcp**: Ripgrep-powered full-text search with YAML frontmatter extraction.
- 60+ servers tracked on PulseMCP.

**AGH agent use cases:**

- **Personal knowledge agent:** Agent maintains an Obsidian vault as a developer's second brain -- automatically creates notes from conversations, links related concepts, and surfaces relevant notes during coding sessions.
- **Daily journal generator:** Agent compiles daily activity (commits, PRs, meetings, messages) into a structured Obsidian daily note.
- **Research vault:** Agent stores research findings with proper tags and backlinks, making them searchable across future sessions.

**Status:** Very mature community ecosystem. 60+ servers, multiple stable options.

---

#### 2.5 Roam Research

**What it does:** Would provide integration with Roam Research's graph-based note-taking system.

**MCP servers available:** None found.

**AGH agent use cases:**

- **Graph-aware knowledge agent:** Agent creates and links blocks in Roam's graph structure based on coding sessions and research.

**Status:** Needs to be built. Roam's user base is smaller and more niche than Obsidian.

---

### 3. Project Management

#### 3.1 Linear

**What it does:** Create, update, search, and comment on issues. Manage projects, cycles, and team workflows.

**MCP servers available:**

- **jerhadf/linear-mcp-server**: Full Linear API integration for issue management.
- **DX Heroes**: Combined Jira + Linear integration.

**AGH agent use cases:**

- **Auto-ticket from code:** Agent detects TODOs and FIXMEs in codebase, creates corresponding Linear issues with proper labels and assignees.
- **Sprint reporter:** Agent generates sprint retrospective summaries from completed/moved issues, posts to Slack.
- **Bug-to-fix pipeline:** Agent receives a bug report (via Slack or email), creates a Linear issue, researches the codebase for likely root cause, and adds analysis as comments.
- **PR-to-ticket linker:** Agent automatically links merged PRs to their Linear tickets and updates ticket status.

**Status:** Stable community server. Linear is popular with engineering teams, making this a high-priority AGH integration.

---

#### 3.2 Jira

**What it does:** Execute JQL queries, create/update/transition tickets, manage epics and sprints, add comments and attachments.

**MCP servers available:**

- **Atlassian Rovo MCP** (Official, February 2026): OAuth 2.1 authorization, enterprise security. Covers Jira + Confluence.
- **Community servers**: 74+ Jira MCP servers tracked on PulseMCP. KS-GEN-AI/jira-mcp-server (TypeScript, JQL support).
- **DX Heroes**: Combined Jira + Linear integration.

**AGH agent use cases:**

- **Ticket auto-population:** Agent takes a brief description, enriches it with codebase context (affected files, related PRs, similar past issues), and creates a fully detailed Jira ticket.
- **Cross-system sync:** Agent keeps Jira tickets in sync with GitHub issues or Linear tickets.
- **Sprint velocity tracker:** Agent calculates velocity metrics from completed sprints, identifies trends, posts analysis to Confluence.

**Status:** Production-ready. Official Atlassian support with OAuth 2.1 makes this enterprise-grade.

---

#### 3.3 Asana

**What it does:** Manage tasks, projects, workspaces, sections, tags, and custom fields. Full CRUD with permission controls.

**MCP servers available:**

- **Official Asana MCP** (mcp.asana.com): Beta server with SSE endpoint. Note: v1 deprecated May 2026, migrating to v2.
- **roychri/mcp-server-asana** (Community): Broad tool coverage, extensible, supports disabling write operations for safe testing.

**AGH agent use cases:**

- **Task breakdown agent:** Agent takes a high-level project description, breaks it into subtasks with estimates, and creates them in Asana with proper dependencies.
- **Status reporter:** Agent reads Asana project status, compiles progress across multiple projects, generates executive summary.

**Status:** Production-ready (official server), though v2 migration is in progress.

---

#### 3.4 Monday.com

**What it does:** Full access to Monday.com's GraphQL API -- boards, items, sub-items, updates, documents, and automation rules.

**MCP servers available:**

- **Official Monday.com MCP** (mondaycom/mcp): Plug-and-play server with dynamic API tools for full GraphQL surface.
- **Jovan Sakovic** (Community, Python): Boards, items, updates, documents. 8,900+ downloads.

**AGH agent use cases:**

- **Board automator:** Agent monitors external events (deploys, alerts) and creates/updates Monday.com items automatically.
- **Cross-tool reporter:** Agent aggregates data from Monday.com boards and generates reports in Google Sheets or email.

**Status:** Production-ready with official server.

---

#### 3.5 Trello

**What it does:** Manage boards, lists, cards, comments, attachments, and labels.

**MCP servers available:**

- **delorenj/mcp-server-trello**: TypeScript, rate limiting, type-safe, in official MCP Registry.
- **Composio Trello MCP**: AI agent integration for card/list/board management.
- **mcp-trello** (PyPI): Python-based board management.

**AGH agent use cases:**

- **Kanban manager:** Agent moves cards based on PR status, adds comments with build results, archives completed cards.
- **Card creator:** Agent converts meeting notes or Slack messages into Trello cards with proper labels.

**Status:** Stable community servers. Lower priority given Trello's declining market share vs Linear/Jira.

---

#### 3.6 ClickUp

**What it does:** Tasks, checklists, sprints, comments, tags, spaces, lists, folders, files, docs, chat, time tracking, goals, and OKRs.

**MCP servers available:**

- **taazkareem/clickup-mcp-server**: High-performance server with document management, chat, goals/KRs, OAuth. Supports remote MCP connections.

**AGH agent use cases:**

- **OKR tracker:** Agent monitors key results progress, sends weekly updates to stakeholders, flags at-risk objectives.
- **Doc-to-task converter:** Agent reads ClickUp docs and creates structured task hierarchies from requirements.

**Status:** Stable community server with broad feature coverage.

---

#### 3.7 Basecamp

**What it does:** Manage projects, to-do lists, messages, schedules, and team members.

**MCP servers available:**

- **georgeantonopoulos/Basecamp-MCP-Server**: FastMCP-based, 75 tools, compatible with Cursor/Codex/Claude Desktop.
- **mcp-basecamp** (PyPI): Projects, to-dos, messages, schedules.

**AGH agent use cases:**

- **Bulk task creator:** Agent takes a pasted list of tasks, creates them in the correct Basecamp project and to-do list.
- **Daily digest:** Agent summarizes new messages, completed to-dos, and upcoming deadlines across all projects.

**Status:** Stable community servers.

---

#### 3.8 Shortcut

**What it does:** Stories, Epics, Docs, iteration operations, team workflows, objective tracking, and search.

**MCP servers available:**

- **Official Shortcut MCP Server**: Hosted server with OAuth authentication. Find, create, and update Stories, Epics, and Docs.

**AGH agent use cases:**

- **Sprint planner:** Agent analyzes velocity and backlog, recommends stories for the next iteration, and moves them into the sprint.
- **Story enricher:** Agent reads a brief story description, researches codebase for context, and adds technical details and acceptance criteria.

**Status:** Production-ready with official hosted server.

---

### 4. Knowledge Bases

#### 4.1 Confluence

**What it does:** Read, create, and update Confluence pages. Search across spaces. Integrated with Jira for issue-linked documentation.

**MCP servers available:**

- **Atlassian Rovo MCP** (Official): Unified Jira + Confluence MCP server with OAuth 2.1 and enterprise security.

**AGH agent use cases:**

- **Runbook maintainer:** Agent detects infrastructure changes and updates relevant Confluence runbooks automatically.
- **Architecture doc generator:** Agent analyzes codebase structure and generates/updates architecture documentation in Confluence.
- **Post-mortem writer:** After an incident, agent compiles timeline from Slack messages, PagerDuty alerts, and code changes into a structured Confluence post-mortem.

**Status:** Production-ready via Atlassian's official MCP server.

---

#### 4.2 GitBook

**What it does:** Access and search published documentation. GitBook auto-generates an MCP server for any published docs site.

**MCP servers available:**

- **Auto-generated per site**: Every GitBook published site automatically exposes an MCP server.
- **MCPBook** (Community): Scrapes and indexes GitBook docs for searchable MCP access.
- GitBook also auto-generates llms.txt and llms-full.txt files. Any page can be fetched as Markdown by appending .md to the URL.

**AGH agent use cases:**

- **Documentation search agent:** Agent queries GitBook docs for API references, configuration options, and guides during coding sessions.
- **Doc freshness checker:** Agent compares GitBook docs against current codebase and flags outdated content.

**Status:** Production-ready. Auto-generation means zero setup.

---

#### 4.3 ReadMe

**What it does:** API documentation with interactive references, changelogs, forums, and engagement analytics. MCP server + llms.txt support.

**MCP servers available:**

- ReadMe provides MCP server access, llms.txt support, and AI-powered doc linting via Agent Owlbert.

**AGH agent use cases:**

- **API integration helper:** Agent queries ReadMe docs for third-party API specifications, generates client code, and validates against live endpoints.

**Status:** Stable. Useful for consuming external API documentation.

---

#### 4.4 Mintlify

**What it does:** Developer documentation platform. Auto-generates MCP servers, llms.txt, llms-full.txt, and skill.md files for every docs site. Free on all tiers.

**MCP servers available:**

- **Auto-generated per site**: Every Mintlify docs site exposes an MCP server with zero configuration.
- AI traffic analytics show which agents visit docs and which MCP queries they run.

**AGH agent use cases:**

- **SDK documentation agent:** Agent queries Mintlify-hosted SDK docs during coding to get accurate API signatures and examples.

**Status:** Production-ready. Auto-generation with analytics is the most AI-native approach.

---

### 5. CRM / Sales

#### 5.1 Salesforce

**What it does:** Full CRM data access -- accounts, leads, contacts, opportunities, conversations. Create, update, and delete records.

**MCP servers available:**

- **Salesforce Agentforce MCP** (Official, July 2025 pilot): Native MCP client and server. Enterprise-grade policy enforcement, rate-limiting, access controls. 60+ tools.
- **Community servers**: 312 GitHub stars for the most popular third-party implementation.

**AGH agent use cases:**

- **Lead enrichment agent:** Agent takes a new lead, researches the company and contact using web search, enriches the Salesforce record with context.
- **Deal updater:** Agent monitors email threads and Slack channels for deal-related conversations, updates opportunity stages and notes in Salesforce.
- **Forecast reporter:** Agent pulls pipeline data, calculates forecasts, generates reports, and posts summaries to Slack.

**Status:** Production-ready. Official Salesforce backing with enterprise security.

---

#### 5.2 HubSpot

**What it does:** Read-only CRM data access for analysis, reporting, and insights. FAISS semantic search for intelligent querying.

**MCP servers available:**

- **Community server** (116 GitHub stars): Read-only CRM access with FAISS semantic search.
- No official HubSpot MCP server yet -- community fills the gap.

**AGH agent use cases:**

- **Sales intelligence agent:** Agent queries HubSpot data to prepare for sales calls -- pulls contact history, recent interactions, and deal context.
- **Pipeline analyzer:** Agent analyzes deal pipeline, identifies bottlenecks, generates coaching recommendations for sales managers.

**Status:** Stable community server. Read-only limitation is a design choice (safety), not a technical limitation.

---

#### 5.3 Pipedrive

**What it does:** Full CRM data management -- deals, contacts, activities, organizations, pipelines.

**MCP servers available:**

- **iamsamuelfraga/mcp-pipedrive** (December 2025): Described as the most complete Pipedrive MCP implementation.
- **Pipedream MCP** (mcp.pipedream.com): Static URL with per-user authentication.
- **Coupler.io MCP**: AI-powered sales data analysis.

**AGH agent use cases:**

- **Deal progression agent:** Agent monitors deal stages, sends reminders for stale deals, suggests next actions based on historical patterns.
- **Activity logger:** Agent creates Pipedrive activities from email threads and meeting notes automatically.

**Status:** Stable community servers.

---

#### 5.4 Attio

**What it does:** Modern CRM with flexible data models. Full CRUD for companies, people, deals, tasks, lists, notes.

**MCP servers available:**

- **Official Attio MCP** (mcp.attio.com): OAuth authentication, natural language CRM management.
- **kesslerio/attio-mcp-server** (Community): 14 universal tools, batch ops, 10 MCP prompts, 3 Claude Skills, OAuth. 1,291 commits.

**AGH agent use cases:**

- **Startup CRM agent:** Agent manages the entire sales pipeline for small teams -- creates contacts from emails, tracks deals, sends follow-ups.
- **Investor tracking:** Agent maintains an Attio database of investor contacts, tracks communications, and prepares meeting briefings.

**Status:** Production-ready. Official hosted server with OAuth. Popular with AI-native companies.

---

### 6. Design Tools

#### 6.1 Figma

**What it does:** Read design data (node tree, auto-layout, variants, design tokens, component properties). Write to canvas (open beta). Code Connect maps design components to code components.

**MCP servers available:**

- **Official Figma MCP** (mcp.figma.com): First-party product built by Figma in partnership with Anthropic, Cursor, and VS Code. Open beta with write-to-canvas (March 2026). Local desktop server at 127.0.0.1:3845.
- **Framelink Figma Connector**: Layout information for AI coding agents.

**AGH agent use cases:**

- **Design-to-code agent:** Agent reads a Figma frame, uses Code Connect to map design components to actual codebase components, generates production React code using the team's real component library.
- **Design review agent:** Agent compares implemented UI (via screenshots) against Figma designs and identifies visual discrepancies.
- **Component library sync:** Agent monitors Figma component changes and creates PRs to update the codebase component library.

**Status:** Production-ready. Official first-party server with deep integration. Code Connect is a standout feature.

---

#### 6.2 Miro

**What it does:** AI tools connect with Miro boards for visual workspace interaction -- create/modify elements, manage boards.

**MCP servers available:**

- **Official Miro MCP Server** (February 2026): Gateway between AI tools and Miro boards. Documented at developers.miro.com.

**AGH agent use cases:**

- **Architecture diagrammer:** Agent creates system architecture diagrams on Miro boards based on codebase analysis.
- **Retrospective facilitator:** Agent sets up retro board templates, collects team input from Slack, and organizes sticky notes.
- **Brainstorm visualizer:** Agent takes brainstorming session notes and creates organized mind maps on Miro.

**Status:** Beta. Official server with more features in development.

---

#### 6.3 Excalidraw

**What it does:** Create and manipulate hand-drawn style diagrams. Real-time canvas with WebSocket sync, element creation/modification, and AI-assisted diagramming.

**MCP servers available:**

- **excalidraw/excalidraw-mcp** (Official, February 2026): Streams hand-drawn diagrams with viewport control. Works with Claude, ChatGPT, VS Code, Goose.
- **yctimlin/excalidraw** (Community, November 2025): Node.js API for element creation, modification, and organization.
- **excalidraw-mcp** (PyPI): Dual-language server with live canvas, real-time sync, WebSocket updates.

**AGH agent use cases:**

- **Architecture sketch agent:** Agent generates architecture diagrams from natural language descriptions using technology-aware styling for 50+ technologies.
- **Incident timeline visualizer:** Agent creates visual incident timelines from log data and alert sequences.
- **Data flow diagrammer:** Agent analyzes codebase data flows and generates Excalidraw diagrams showing system interactions.

**Status:** Stable. Official and community servers both available.

---

### 7. Ecosystem Directories and Meta-Platforms

#### 7.1 ClawHub (OpenClaw Skills Marketplace)

ClawHub hosts 5,700+ to 31,000+ community skills covering email management, browser automation, and more. However, security is a concern: 7.6% of publicly available skills contain dangerous patterns.

**Relevant patterns for AGH:**

- Skills follow the AgentSkills SKILL.md format, which has converged across OpenClaw and Hermes.
- Plugin examples include QQ messaging (OneBot 11), wallet tools, cognitive memory systems, and event visualization.
- AGH should monitor ClawHub for popular skill categories that indicate user demand.

#### 7.2 Hermes Agent Ecosystem

Hermes ships 48 built-in tools across 40 toolsets and supports MCP via `hermes mcp serve`. The ecosystem map (hermes-ecosystem.vercel.app) catalogs 80+ tools, skills, and integrations.

**Notable integrations:**

- Multi-platform gateway: Telegram, Discord, Slack, WhatsApp, Signal, Feishu/Lark, WeCom.
- Memory systems: hindsight (8,362 stars), autocontext (711 stars), ClawMem (86 stars).
- Multi-agent orchestration: mission-control (3,700+ stars).

**Relevant patterns for AGH:**

- Hermes's multi-platform gateway pattern (single agent, multiple communication channels) is directly applicable to AGH extensions.
- The `hermes mcp serve` pattern of exposing agent sessions to MCP clients is similar to AGH's architecture.

#### 7.3 awesome-mcp-servers (wong2/awesome-mcp-servers)

The primary community curation point. Categories most relevant to AGH:

- Communication (Slack, Discord, Telegram, Email)
- Project Management (Linear, Jira, Asana, Monday, Trello, ClickUp)
- Knowledge & Memory (Notion, Obsidian, Confluence)
- CRM (Salesforce, HubSpot, Pipedrive, Attio)

#### 7.4 Meta-Integration Platforms

These platforms bundle many integrations under a single MCP interface:

| Platform         | Integrations                              | Notes                                 |
| ---------------- | ----------------------------------------- | ------------------------------------- |
| **Composio**     | Unified MCP for all hosted apps           | Single setup, no per-app npx commands |
| **Pipedream**    | 2,500+ APIs (Slack, GitHub, Notion, etc.) | Hosted MCP servers or self-deploy     |
| **Knit MCP**     | 10,000+ tools (HRIS, ATS, CRM, etc.)      | Broadest coverage                     |
| **WayStation**   | Notion, Monday, AirTable, etc.            | Productivity-focused hub              |
| **Activepieces** | Calendar, Notion, advanced flows          | Dynamic server with app composition   |
| **Arcade.dev**   | Microsoft 365, Google Workspace           | 30+ tools per suite                   |

---

## Recommended AGH Extension Priorities

### Tier 1: High-Impact, Production-Ready MCP Servers Exist

These integrations have mature MCP servers and address the most common agentic workflows:

1. **Slack** -- The primary communication channel for engineering teams. Agent monitoring, Q&A, notifications, and coordination.
2. **Linear** -- The most popular issue tracker for modern engineering teams. Auto-ticketing, sprint management, PR linking.
3. **Notion** -- Dominant knowledge base. Documentation sync, research compilation, sprint planning.
4. **Google Workspace** -- Full productivity suite. Email, calendar, docs, sheets with 100+ tools.
5. **Figma** -- Official first-party MCP with Code Connect. Design-to-code is a killer use case.
6. **Jira + Confluence** -- Enterprise standard. Official Atlassian MCP with OAuth 2.1.

### Tier 2: Strong Use Cases, Stable Servers

7. **Microsoft 365** -- Enterprise productivity. Official Work IQ servers.
8. **Discord** -- Community support and open-source project management.
9. **Obsidian** -- Developer knowledge management. 60+ servers.
10. **Shortcut** -- Official hosted MCP. Popular with smaller engineering teams.
11. **Asana / Monday.com / ClickUp** -- Project management alternatives with official MCP support.

### Tier 3: Niche but Valuable

12. **Telegram / WhatsApp** -- Ops notifications, personal assistant patterns.
13. **Salesforce / HubSpot / Attio** -- CRM integration for sales-adjacent engineering.
14. **Miro / Excalidraw** -- Visual collaboration and diagramming.
15. **GitBook / Mintlify** -- Auto-generated MCP servers for documentation consumption.

### Tier 4: Build Opportunities

16. **Matrix** -- No MCP server exists. Opportunity for AGH to serve privacy-focused organizations.
17. **Roam Research** -- No MCP server exists. Niche user base.

---

## AGH Extension Architecture Considerations

Based on this research, AGH extensions for third-party integrations should consider:

1. **Thin wrapper pattern:** Most integrations already have MCP servers. AGH extensions can wrap existing MCP servers as subprocesses, adding session-aware context (workspace memory, agent history) via the Host API.

2. **Multi-channel gateway:** Following the Hermes pattern, a single AGH extension could provide a unified communication interface across Slack/Discord/Teams/Telegram, with routing rules stored in AGH config.

3. **Bidirectional sync:** The highest-value integrations are not just read or write -- they maintain bidirectional state between AGH sessions and external tools (e.g., Linear ticket status reflects PR status; Notion docs reflect codebase changes).

4. **Meta-platform integration:** Rather than building individual connectors, AGH could integrate with Composio or Pipedream as a single extension that provides access to thousands of tools.

5. **Security model:** The MCP ecosystem has known security issues (43% of public servers have command injection vulnerabilities, 7.6% of ClawHub skills contain dangerous patterns). AGH extensions should enforce permission boundaries, rate limiting, and audit logging through the observe Host API.

---

## Sources

- [awesome-mcp-servers (wong2)](https://github.com/wong2/awesome-mcp-servers)
- [mcp-awesome.com (1200+ servers)](https://mcp-awesome.com/)
- [PulseMCP Server Directory (11,870+)](https://www.pulsemcp.com/servers)
- [MCP Servers Directory](https://mcpservers.org)
- [Slack MCP Overview](https://docs.slack.dev/ai/slack-mcp-server/)
- [korotovsky/slack-mcp-server](https://github.com/korotovsky/slack-mcp-server)
- [Slack MCP Evolution (Flagship)](https://flagship.cc/en/blogs/columns/slack-mcp-server-official-release)
- [Best MCP Servers for Slack (FastMCP)](https://fastmcp.me/blog/best-mcp-servers-for-slack-communication)
- [Best MCP Server for Slack (Truto)](https://truto.one/blog/best-mcp-server-for-slack-in-2026)
- [Discord MCP Server (FastMCP)](https://fastmcp.me/mcp/details/1354/discord)
- [Microsoft Teams MCP](https://mcpservers.org/servers/github-com-inditextech-mcp-teams-server)
- [Project Management MCP Servers (Merge.dev)](https://www.merge.dev/blog/project-management-mcp-servers)
- [Asana MCP Server (Official)](https://developers.asana.com/docs/using-asanas-mcp-server)
- [awesome-mcp-servers Project Management](https://github.com/TensorBlock/awesome-mcp-servers/blob/main/docs/project--task-management.md)
- [Atlassian MCP Server Best Practices](https://mcpmanager.ai/blog/atlassian-mcp-server-installation-best-practices/)
- [Jira MCP Servers (PulseMCP)](https://www.pulsemcp.com/servers?q=jira)
- [Google Workspace MCP (taylorwilsdon)](https://github.com/taylorwilsdon/google_workspace_mcp)
- [Google Workspace MCP (workspacemcp.com)](https://workspacemcp.com/)
- [Google Official MCP Support](https://cloud.google.com/blog/products/ai-machine-learning/announcing-official-mcp-support-for-google-services)
- [Notion MCP (Official)](https://developers.notion.com/guides/mcp/mcp)
- [Notion MCP Servers (PulseMCP)](https://www.pulsemcp.com/servers?q=notion)
- [Salesforce MCP Support](https://developer.salesforce.com/blogs/2025/06/introducing-mcp-support-across-salesforce)
- [CRM MCP Servers Overview](https://dev.to/grove_chatforest/crm-mcp-servers-salesforce-hubspot-pipedrive-attio-and-beyond-off)
- [Figma MCP Server Demand](https://www.primepublishers.com/figma-rides-on-strong-mcp-server-demand-a-sign-for-more-upside/article_d114cc6d-f1c1-53ca-89f9-b558f62ea1b2.html)
- [Figma MCP Production Pipeline](https://www.francescatabor.com/articles/2026/3/31/building-a-figma-driven-mcp-production-pipeline)
- [Figma MCP Design-to-Code (Siemens)](https://blog.siemens.com/2025/11/revolutionizing-design-to-code-workflows-with-figma-mcp-server/)
- [MCP-Obsidian (mcpvault)](https://github.com/bitbonsai/mcpvault)
- [Obsidian MCP Server (cyanheads)](https://github.com/cyanheads/obsidian-mcp-server)
- [Obsidian MCP Servers (PulseMCP)](https://www.pulsemcp.com/servers?q=obsidian)
- [Email MCP Server](https://mcpservers.org/servers/Shy2593666979/mcp-server-email)
- [WhatsApp MCP Server](https://github.com/lharries/whatsapp-mcp)
- [Telegram MCP Server (overpod)](https://www.pulsemcp.com/servers/overpod-telegram)
- [Telegram MCP Server (sparfenyuk)](https://mcpservers.org/servers/sparfenyuk/mcp-telegram)
- [ClickUp MCP Server](https://github.com/taazkareem/clickup-mcp-server)
- [Monday.com MCP (Official)](https://github.com/mondaycom/mcp)
- [Trello MCP Server](https://github.com/delorenj/mcp-server-trello)
- [Basecamp MCP Server](https://github.com/georgeantonopoulos/Basecamp-MCP-Server)
- [Shortcut MCP Server](https://help.shortcut.com/hc/en-us/articles/36443434285844-MCP-Server)
- [GitBook MCP (September 2025)](https://www.gitbook.com/blog/new-in-gitbook-september-2025)
- [Mintlify Documentation Tools](https://www.mintlify.com/library/7-best-software-documentation-tools-in-2026)
- [Pipedrive MCP (iamsamuelfraga)](https://github.com/iamsamuelfraga/mcp-pipedrive)
- [Miro MCP Server](https://developers.miro.com/docs/miro-mcp)
- [Excalidraw MCP (Official)](https://github.com/excalidraw/excalidraw-mcp)
- [Attio MCP Server (Truto)](https://truto.one/blog/best-mcp-server-for-attio-in-2026)
- [Microsoft Work IQ MCP](https://learn.microsoft.com/en-us/microsoft-agent-365/tooling-servers-overview)
- [Microsoft 365 MCP Server (Softeria)](https://github.com/softeria/ms-365-mcp-server)
- [PnP CLI Microsoft 365 MCP](https://github.com/pnp/cli-microsoft365-mcp-server)
- [Arcade.dev Office 365 MCP](https://www.arcade.dev/blog/microsoft-office-365-mcp-servers-launch)
- [ClawHub Plugins](https://clawhub.ai/plugins)
- [Hermes Agent Ecosystem Map](https://hermes-ecosystem.vercel.app/)
- [awesome-hermes-agent](https://github.com/0xNyk/awesome-hermes-agent)
- [OpenClaw vs Hermes (The New Stack)](https://thenewstack.io/persistent-ai-agents-compared/)
- [Top 12 MCP Servers 2026 (Skyvia)](https://skyvia.com/blog/best-mcp-servers/)
- [Best MCP Servers for Productivity (Fast.io)](https://fast.io/resources/best-mcp-servers-productivity/)
- [Best MCP Servers for Developers 2026 (Builder.io)](https://www.builder.io/blog/best-mcp-servers-2026)
- [25 Best MCP Servers (PremAI)](https://blog.premai.io/25-best-mcp-servers-for-ai-agents-complete-setup-guide-2026/)
