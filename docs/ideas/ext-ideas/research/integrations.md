# AGH Extension Ideas — Third-Party Integrations

**Date**: 2026-04-11
**Sources**: 4 parallel research agents covering DevOps/CI, Communication/Productivity, Data/AI/Search, and Browser/Media/Specialized integrations
**Purpose**: Catalog concrete third-party integrations that could be built as AGH extensions

---

## Executive Summary

Four parallel research agents surveyed the MCP ecosystem (21,000+ servers on Glama.ai) and mapped **120+ third-party integrations** across 12 categories. The key finding: **~80% of integrations have existing MCP servers** that AGH can wrap as subprocess extensions with minimal effort. The remaining 20% need custom extensions built from REST APIs.

AGH's differentiator over standalone MCP servers is the **Host API** — extensions can combine external tool access with session memory, skills, observe events, and cross-tool orchestration to create stateful, context-aware workflows.

---

## Priority Summary — Top 30 Integrations

### Tier 1: Ship First (highest impact, production-ready MCP servers)

| #   | Integration        | Category       | MCP Status           | Use Case                                                  |
| --- | ------------------ | -------------- | -------------------- | --------------------------------------------------------- |
| 1   | **GitHub**         | DevOps         | Official             | PR lifecycle, issue management, code review automation    |
| 2   | **Slack**          | Communication  | Official (47 tools)  | Team Q&A bot, incident coordination, deploy notifications |
| 3   | **Linear**         | Project Mgmt   | Community            | Ticket-to-PR automation, sprint ops, bug triage           |
| 4   | **Notion**         | Knowledge Base | Official             | Living docs, research compilation, sprint planning        |
| 5   | **Sentry**         | Monitoring     | Official             | Error alert → investigate → fix → PR pipeline             |
| 6   | **Playwright**     | Browser        | Official (Microsoft) | E2E testing, web scraping, form automation                |
| 7   | **Supabase**       | Database       | Official (20+ tools) | Full BaaS: DB, auth, storage, edge functions              |
| 8   | **Firecrawl**      | Web Scraping   | Official             | Web-to-markdown, site crawling, content extraction        |
| 9   | **GitHub Actions** | CI/CD          | Community            | CI monitoring, failure diagnosis, workflow optimization   |
| 10  | **Stripe**         | Finance        | Official (25 tools)  | Billing ops, subscription mgmt, revenue reports           |

### Tier 2: Build Next (strong value, mature ecosystem)

| #   | Integration           | Category       | MCP Status              | Use Case                                           |
| --- | --------------------- | -------------- | ----------------------- | -------------------------------------------------- |
| 11  | **Datadog**           | Monitoring     | Official (GA)           | Observability investigation, latency diagnosis     |
| 12  | **Google Workspace**  | Productivity   | Community (100+ tools)  | Email, calendar, docs, sheets automation           |
| 13  | **Figma**             | Design         | Official (Code Connect) | Design-to-code, component sync, design review      |
| 14  | **Jira + Confluence** | Project Mgmt   | Official (Atlassian)    | Enterprise issue tracking, knowledge management    |
| 15  | **Neon**              | Database       | Official                | Branch-safe migrations, query tuning               |
| 16  | **Terraform**         | Infrastructure | Official (HashiCorp)    | IaC provisioning, plan/apply workflows             |
| 17  | **Kubernetes**        | Infrastructure | Multiple                | Pod debugging, deployment management, log analysis |
| 18  | **Snyk**              | Security       | Official (11 tools)     | SAST, SCA, container scanning, SBOM                |
| 19  | **SonarQube**         | Security       | Official (423 stars)    | Code quality gates, tech debt tracking             |
| 20  | **Brave Search**      | Web Search     | Official                | Privacy-first research, error investigation        |

### Tier 3: Differentiate (strategic value, growing demand)

| #   | Integration        | Category      | MCP Status | Use Case                                       |
| --- | ------------------ | ------------- | ---------- | ---------------------------------------------- |
| 21  | **Grafana**        | Monitoring    | Official   | Dashboard-driven diagnosis, anomaly detection  |
| 22  | **PagerDuty**      | Monitoring    | Community  | On-call copilot, incident lifecycle management |
| 23  | **Ollama**         | AI/ML         | Community  | Local model inference, privacy-sensitive ops   |
| 24  | **OpenRouter**     | AI/ML         | Community  | Multi-model gateway, cost-optimized inference  |
| 25  | **n8n**            | Automation    | Community  | Self-hosted workflow automation (1,396 nodes)  |
| 26  | **Home Assistant** | IoT           | Official   | Smart home control, energy monitoring          |
| 27  | **AWS S3**         | Cloud Storage | Official   | File management, data pipeline triggers        |
| 28  | **Exa**            | Web Search    | Community  | Neural semantic search for research            |
| 29  | **PostHog**        | Analytics     | Official   | Product analytics, feature flags, experiments  |
| 30  | **Twitter/X**      | Social Media  | Community  | Social media management, brand monitoring      |

---

## Detailed Integration Catalog

### 1. DevOps & Developer Tools

| Integration             | MCP Server                    | Tools                 | Key Capability                                   | Example AGH Workflow                                                                                                       |
| ----------------------- | ----------------------------- | --------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| **GitHub**              | Official + community          | Full API              | PRs, issues, code search, branches, Actions      | Agent receives Linear ticket → researches codebase → implements fix → opens PR → monitors CI → responds to review comments |
| **GitLab**              | Official                      | MR, pipelines, issues | Merge requests, CI pipelines, code browsing      | Pipeline fails → agent reads logs → correlates with recent MR → auto-fixes or creates issue with root cause                |
| **GitHub Actions**      | Community                     | Workflow mgmt         | Trigger, cancel, rerun workflows, read logs      | Agent monitors builds → diagnoses flaky tests → detects failure patterns → suggests workflow optimizations                 |
| **CircleCI**            | Official                      | Failure diagnosis     | Error summaries, flaky test detection, rollbacks | Build fails → agent diagnoses → correlates with commits → creates fix PR or triggers rollback                              |
| **Jenkins**             | Official plugin               | CI/CD automation      | Job management, build logs, pipeline control     | Enterprise CI management with complex multi-stage pipelines                                                                |
| **ArgoCD**              | K8s MCP Toolkit               | GitOps                | App sync, deployment status, rollback            | Agent monitors sync status → detects drift → checks pod health → applies fix or rolls back                                 |
| **Vercel**              | Official handler              | Deployments           | Deploy, rollback, environment mgmt               | Agent deploys to staging → runs smoke tests → promotes to production → posts summary                                       |
| **Railway**             | Official                      | Service mgmt          | Deploy, scale, configure environments            | Agent manages Railway services lifecycle                                                                                   |
| **SonarQube**           | Official (423 stars)          | Quality gates         | Bugs, vulnerabilities, code smells, tech debt    | PR created → agent runs analysis → auto-fixes simple issues → blocks merge if quality gate fails                           |
| **Snyk**                | Official (11 tools)           | Security scanning     | SAST, SCA, IaC, container, SBOM, AI-BOM          | Nightly scans → triage by severity → auto-PR for critical vulns → SBOM for compliance                                      |
| **Semgrep**             | Built into binary             | Static analysis       | Custom rules, vulnerability detection            | Pre-commit scanning → inline PR comments with fix suggestions                                                              |
| **Terraform**           | Official (HashiCorp)          | IaC                   | Registry, plan, apply, workspace mgmt            | Agent generates HCL → runs plan → presents for approval → applies → updates docs                                           |
| **Pulumi**              | Official                      | IaC (code-based)      | Infrastructure in Go/TS/Python                   | Agent writes Pulumi Go code for infrastructure changes                                                                     |
| **Kubernetes**          | Multiple (kubectl, k8m, Lens) | Cluster mgmt          | Pods, logs, events, helm, istio                  | Alert → agent checks pods → reads logs → identifies OOM kill → scales up → creates post-mortem                             |
| **AWS**                 | Official (60+ servers)        | Cloud ops             | Lambda, ECS, S3, EC2, RDS, CloudWatch            | Agent monitors cloud costs → identifies unused resources → proposes cleanup with savings                                   |
| **Docker**              | Community                     | Container mgmt        | Build, run, manage containers                    | Agent builds images → runs tests in containers → pushes to registry                                                        |
| **Dependabot/Renovate** | **None (build opportunity)**  | —                     | Dependency update management                     | Agent monitors deps → creates grouped PRs → runs security scans → auto-merges safe patches                                 |

### 2. Communication & Messaging

| Integration         | MCP Server                       | Tools             | Key Capability                                | Example AGH Workflow                                                                          |
| ------------------- | -------------------------------- | ----------------- | --------------------------------------------- | --------------------------------------------------------------------------------------------- |
| **Slack**           | Official (47 tools, GA Feb 2026) | Full workspace    | Channels, messages, threads, canvases, search | Agent monitors #help-engineering → researches codebase → posts threaded answer with code refs |
| **Discord**         | Community (multiple)             | Server mgmt       | Channels, messages, forums, reactions         | Community support bot → searches docs + past issues → provides answers                        |
| **Microsoft Teams** | Official (Work IQ)               | Chat/channels     | Create chats, post messages, manage channels  | Meeting prep agent → pulls docs from SharePoint → posts briefing to Teams channel             |
| **Telegram**        | Community (multiple)             | Bot API + MTProto | Messaging, media, groups                      | Ops notification pipeline → deployment status, health alerts, CI results                      |
| **WhatsApp**        | Community (beta)                 | Web API           | Send/receive messages                         | Customer response agent → looks up CRM → drafts contextual replies                            |
| **Email (Gmail)**   | Community (100+ tools)           | Full Gmail API    | Send, read, search, label, filter             | Email triage → categorize by urgency → draft routine replies → escalate important ones        |
| **Email (Outlook)** | Official (Work IQ)               | Graph API         | Messages, calendar, files                     | Report distributor → generates status reports → formats as email → sends to stakeholders      |
| **Matrix**          | **None (build opportunity)**     | —                 | Decentralized messaging                       | Self-hosted comms agent for privacy-focused organizations                                     |

### 3. Productivity & Project Management

| Integration          | MCP Server                | Tools          | Key Capability                           | Example AGH Workflow                                                                        |
| -------------------- | ------------------------- | -------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------- |
| **Notion**           | Official                  | Full API       | Pages, databases, blocks, search         | Code changes → agent auto-updates Notion docs → cross-references existing pages             |
| **Obsidian**         | Community (60+ servers)   | Vault access   | Read, write, search, tags, backlinks     | Personal knowledge agent → auto-creates notes from conversations → links related concepts   |
| **Google Workspace** | Community (100+ tools)    | Full suite     | Gmail, Calendar, Docs, Sheets, Drive     | Meeting notes agent → records action items → creates Doc → assigns tasks → sends follow-ups |
| **Microsoft 365**    | Official + community      | Full suite     | Word, Excel, SharePoint, OneDrive, Teams | Onboarding automator → sets up accounts → creates folders → sends welcome email             |
| **Linear**           | Community                 | Full API       | Issues, projects, cycles, teams          | Auto-ticket from TODO/FIXME → sprint reporter → bug-to-fix pipeline → PR-to-ticket linker   |
| **Jira**             | Official (Atlassian Rovo) | OAuth 2.1      | JQL, epics, sprints, transitions         | Ticket auto-population → enriches with codebase context → cross-system sync                 |
| **Confluence**       | Official (Atlassian Rovo) | OAuth 2.1      | Pages, spaces, search                    | Runbook maintainer → architecture doc generator → post-mortem writer                        |
| **Asana**            | Official (mcp.asana.com)  | Full API       | Tasks, projects, sections, custom fields | Task breakdown agent → high-level description → subtasks with estimates and dependencies    |
| **Monday.com**       | Official                  | GraphQL API    | Boards, items, updates, documents        | Board automator → external events create/update items automatically                         |
| **ClickUp**          | Community                 | Broad coverage | Tasks, docs, goals, OKRs, chat           | OKR tracker → monitors key results → weekly updates → flags at-risk objectives              |
| **Shortcut**         | Official (hosted)         | OAuth          | Stories, Epics, Docs, iterations         | Story enricher → researches codebase → adds technical details and acceptance criteria       |
| **Figma**            | Official (Code Connect)   | Design data    | Nodes, auto-layout, variants, tokens     | Design-to-code → reads frame → maps to codebase components → generates production React     |
| **Miro**             | Official (beta)           | Board mgmt     | Elements, boards, collaboration          | Architecture diagrammer → creates system diagrams from codebase analysis                    |
| **Excalidraw**       | Official + community      | Canvas toolkit | Elements, real-time sync, WebSocket      | Architecture sketch → generates diagrams from natural language descriptions                 |

### 4. Databases & Data

| Integration            | MCP Server                          | Tools                | Key Capability                                       | Example AGH Workflow                                                                  |
| ---------------------- | ----------------------------------- | -------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------- |
| **PostgreSQL**         | Multiple (official, Google Toolbox) | SQL access           | Read-only queries, schema inspection                 | Bug investigation → query production DB → find anomaly → write migration + code patch |
| **MySQL**              | Community + Google Toolbox          | SQL access           | Query, schema browsing                               | Legacy system analysis → understand schema → generate Go structs                      |
| **MongoDB**            | Official                            | Document CRUD        | Aggregation pipelines, Atlas mgmt                    | Identify slow aggregations → propose index optimizations → validate with explain()    |
| **Redis**              | Official (Dec 2025)                 | Data mgmt            | Keys, TTLs, pub/sub, search                          | Debug cache stampede → inspect TTLs → implement jittered expiration fix               |
| **Supabase**           | Official (20+ tools, OAuth)         | Full BaaS            | DB, auth, storage, edge functions                    | Bootstrap entire backend through natural language in single session                   |
| **Neon**               | Official (29 tools)                 | Serverless Postgres  | Branch-based migrations, query tuning                | Create branch → test migration → validate queries → merge or rollback                 |
| **PlanetScale**        | Official                            | MySQL branching      | Branches, deploy requests                            | Feature branch → apply migrations → run integration tests → deploy request            |
| **Turso**              | Official (--mcp flag)               | Edge SQLite          | Schema design, data ops                              | Set up edge database → design schema → generate client code → deploy replicas         |
| **DynamoDB**           | Official (AWS Labs)                 | NoSQL modeling       | Design patterns, cost analysis, code gen             | Analyze MySQL schema → design DynamoDB single-table model → generate Go SDK code      |
| **Google MCP Toolbox** | Official (Google)                   | **40+ data sources** | Postgres, MySQL, MongoDB, Redis, Neo4j, Snowflake... | Single extension → access any database → join data across engines                     |

### 5. Vector Stores & Search

| Integration       | MCP Server                    | Tools                 | Key Capability                           | Example AGH Workflow                                                                |
| ----------------- | ----------------------------- | --------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------- |
| **LanceDB**       | Community (multiple)          | Embedded vectors      | Zero-config, disk-based, semantic search | AGH semantic memory backend → index memories + skills → retrieve during sessions    |
| **Qdrant**        | Official                      | Vector search         | HNSW, filtered search, code search       | Index codebase → semantic code search during debugging → find similar patterns      |
| **Milvus**        | Official (5 search tools)     | Industrial-scale      | Billion-vector, hybrid search            | Large codebase indexing → semantic function search → incident similarity matching   |
| **Pinecone**      | Via unified (weave-mcp)       | Managed vectors       | Billion-scale managed                    | Documentation indexing → semantic search during coding sessions                     |
| **Elasticsearch** | Community + Google Toolbox    | Full-text + analytics | Index mgmt, document ops                 | Search application logs → diagnose production errors → generate root cause analysis |
| **Algolia**       | Official (Go + Node + hosted) | Enterprise search     | Synonyms, ranking, analytics             | Configure search indexes → set up ranking rules → test quality → deploy             |
| **Meilisearch**   | Official                      | Dev-friendly search   | Typo-tolerant, fast                      | Index knowledge base → instant typo-tolerant search → build UI component            |
| **Firecrawl**     | Official (98K stars)          | Web-to-markdown       | Crawl, scrape, media parse               | Crawl competitor docs → convert to markdown → index for semantic search             |
| **Brave Search**  | Official                      | Privacy-first search  | Independent index, no tracking           | Web research for debugging → search error messages, Stack Overflow, GitHub issues   |
| **Exa**           | Community                     | Neural search         | Semantic understanding                   | Semantic code search → "Go implementation of event sourcing with SQLite"            |
| **Tavily**        | Community                     | RAG-optimized         | Fact-checked, concise results            | Research tasks → find API docs, known issues, best practices                        |

### 6. AI/ML Platforms

| Integration      | MCP Server     | Tools                | Key Capability                    | Example AGH Workflow                                                                                      |
| ---------------- | -------------- | -------------------- | --------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Ollama**       | Community      | Local inference      | Run Llama, Mistral, etc. locally  | Local embedding generation → privacy-sensitive code analysis → offline operation                          |
| **OpenRouter**   | Community      | 100+ models          | Multi-model gateway, cost routing | Dynamic model selection → fast model for quick questions → reasoning model for complex analysis           |
| **Hugging Face** | Community      | Model discovery      | Hub access, model search          | Search for embedding model → download via Ollama → benchmark on domain data → configure as memory backend |
| **Replicate**    | Community      | Hosted inference     | Run any OSS model via API         | Image generation for mockups → specialized NLP models for code analysis                                   |
| **Groq**         | Via OpenRouter | Ultra-fast inference | Custom LPU hardware               | Rapid code analysis → real-time review suggestions during pair programming                                |

### 7. Monitoring & Observability

| Integration   | MCP Server             | Tools              | Key Capability                            | Example AGH Workflow                                                                                    |
| ------------- | ---------------------- | ------------------ | ----------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Sentry**    | Official               | Error tracking     | Stack traces, error frequency, releases   | Critical error → query Sentry → search codebase → create fix PR → verify error rate drops               |
| **Datadog**   | Official (GA Mar 2026) | Full observability | Logs, metrics, traces, APM, SLOs          | Latency spike → query traces → correlate with deployment → identify commit → revert PR                  |
| **Grafana**   | Official               | Dashboard access   | Data sources, incidents, metrics          | Query dashboards for anomalies → correlate with deployments → generate incident summaries               |
| **PagerDuty** | Community              | Incident mgmt      | Acknowledge, resolve, reassign, analytics | Alert → acknowledge → gather context from Datadog/Sentry → diagnose → remediate → resolve → post-mortem |

### 8. Automation & Workflow

| Integration   | MCP Server | Tools             | Key Capability                      | Example AGH Workflow                                                                   |
| ------------- | ---------- | ----------------- | ----------------------------------- | -------------------------------------------------------------------------------------- |
| **Zapier**    | Official   | 8,000+ apps       | Cross-app automation, auth handling | Code review → create Jira ticket + Slack summary + Google Sheet update in one session  |
| **n8n**       | Community  | 1,396 nodes       | Self-hosted, privacy-first          | Design workflow → monitor GitHub for issues → classify with AI → assign → track        |
| **Inngest**   | Official   | Durable execution | Go SDK, event-driven, AgentKit      | Orchestrate deployment pipeline → test → build → stage → smoke test → promote → notify |
| **Temporal**  | Community  | Durable workflows | Retry logic, long-running processes | Data pipeline orchestration → retry handlers → compensation → monitoring               |
| **Pipedream** | Community  | 2,700+ apps       | Code-first (Python/Node/Go/Bash)    | Webhook listener → process alerts → trigger remediation                                |

### 9. Finance

| Integration       | MCP Server                   | Tools             | Key Capability                              | Example AGH Workflow                                                                 |
| ----------------- | ---------------------------- | ----------------- | ------------------------------------------- | ------------------------------------------------------------------------------------ |
| **Stripe**        | Official (25 tools)          | Payment lifecycle | Customers, subscriptions, invoices, refunds | Monitor churn → generate invoices → create discount codes → revenue reports          |
| **Coinbase**      | Official                     | Crypto ops        | Wallet mgmt, onramps, stablecoins           | Portfolio management → track balances → execute trades → tax reporting               |
| **Yahoo Finance** | Community                    | Market data       | Prices, fundamentals, earnings              | Stock screening → earnings analysis → peer comparison → research notes               |
| **Plaid**         | **None (build opportunity)** | —                 | Bank account aggregation                    | Connect bank accounts → categorize transactions → spending patterns → budget reports |

### 10. Browser & Media

| Integration       | MCP Server             | Tools            | Key Capability                      | Example AGH Workflow                                                               |
| ----------------- | ---------------------- | ---------------- | ----------------------------------- | ---------------------------------------------------------------------------------- |
| **Playwright**    | Official (Microsoft)   | Full browser     | Click, fill, navigate, screenshot   | E2E testing → navigate pages → assert content → report results via observe API     |
| **Browserbase**   | Official               | Cloud browsers   | Bot evasion, managed sessions       | Competitive intelligence → scrape JS-heavy pricing pages → extract structured data |
| **YouTube**       | Community (490+ stars) | Transcripts      | Transcript extraction, search       | Research playlist → extract transcripts → summarize → build knowledge base         |
| **DALL-E / Flux** | Community              | Image generation | Text-to-image                       | Generate diagrams, illustrations, hero images for documentation                    |
| **ElevenLabs**    | Community              | TTS              | Voice synthesis                     | Convert blog posts to podcast-style audio narration                                |
| **Spotify**       | Community (93 tools)   | Music control    | Playback, playlists, catalog search | Curate workout playlists → analyze track features → learn preferences over time    |

### 11. Social Media

| Integration    | MCP Server             | Tools            | Key Capability                       | Example AGH Workflow                                                                         |
| -------------- | ---------------------- | ---------------- | ------------------------------------ | -------------------------------------------------------------------------------------------- |
| **Twitter/X**  | Community (8+ servers) | Posting, search  | Tweets, threads, mentions, analytics | Draft tweets from product updates → schedule threads → monitor engagement → weekly analytics |
| **Bluesky**    | Community (57 tools)   | Full AT Protocol | Posting, firehose, social graph      | Cross-post content → monitor brand mentions → audience analytics                             |
| **LinkedIn**   | Via aggregators        | Posting          | Articles, engagement tracking        | Draft LinkedIn articles from internal knowledge → optimize posting times                     |
| **recast-mcp** | Community              | Multi-platform   | URL → platform-specific content      | Blog post → LinkedIn article + Twitter thread + Reddit post + newsletter                     |

### 12. Specialized & Niche

| Integration         | MCP Server            | Tools             | Key Capability                            | Example AGH Workflow                                                                              |
| ------------------- | --------------------- | ----------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **Home Assistant**  | Official (built-in)   | IoT control       | Devices, automations, energy              | Manage daily routines → lighting, HVAC, security → energy reports                                 |
| **AWS S3**          | Official (multiple)   | Object storage    | Buckets, objects, presigned URLs          | Monitor for new files → process CSVs → generate presigned URLs → manage lifecycle                 |
| **Google Maps**     | Official (18+ tools)  | Geolocation       | Geocoding, routing, POI search            | Optimize multi-stop delivery routes → calculate ETAs → generate static maps                       |
| **Mapbox**          | Official              | Geospatial        | Routing, isochrones, map matching         | Real estate analysis → isochrone maps → commute times → nearby amenities                          |
| **SendGrid**        | Community (14+ tools) | Email marketing   | Campaigns, templates, deliverability      | Create campaigns from briefs → manage segments → A/B test subjects                                |
| **Meta-MCP (Magg)** | Community             | Self-extending    | Discover + install MCP servers at runtime | Agent lacks a tool → discovers and installs appropriate MCP server → uses it → permanently learns |
| **Agoragentic**     | Community             | Agent marketplace | Agent-to-agent services + crypto payments | Agent hires specialized agents for subtasks → pays via USDC on Base L2                            |

---

## Build-From-Scratch Opportunities

These integrations have no existing MCP server and represent differentiation for AGH:

| Integration                | What to Build                                     | Why It Matters                                      |
| -------------------------- | ------------------------------------------------- | --------------------------------------------------- |
| **Dependabot/Renovate**    | Dependency update mgmt with security scanning     | Combining updates + security + auto-merge is unique |
| **GitHub Security Alerts** | Dependabot alerts, secret scanning, code scanning | Notable gap — GitHub's security features lack MCP   |
| **Plaid**                  | Banking data aggregation                          | Personal finance agent enabler                      |
| **Matrix**                 | Decentralized messaging                           | Serves privacy-focused organizations                |
| **MQTT (standalone)**      | IoT device communication                          | Industrial monitoring beyond Home Assistant         |
| **Remotion**               | Programmatic video in React                       | Data visualization videos                           |
| **ConvertKit**             | Creator email platform                            | Creator economy automation                          |
| **LaunchDarkly**           | Feature flag management                           | Agent-controlled progressive rollouts               |
| **Incident.io**            | Modern incident management                        | Growing platform with no MCP                        |

---

## Recommended Extension Bundles

### Bundle 1: Development Lifecycle

**Goal**: Ticket → Code → PR → Merged, fully autonomous

- GitHub MCP (version control, PRs)
- Linear or Jira MCP (issue tracking)
- GitHub Actions MCP (CI monitoring)
- SonarQube + Snyk MCP (quality + security gates)
- Slack MCP (team notifications)

### Bundle 2: Incident Response

**Goal**: Alert → Diagnose → Fix → Resolve, with cross-tool investigation

- Sentry MCP (error tracking)
- Datadog MCP (metrics, traces, logs)
- PagerDuty MCP (incident lifecycle)
- Grafana MCP (dashboards)
- Kubernetes MCP (infrastructure)
- Slack MCP (coordination)
- GitHub MCP (fix PRs)

### Bundle 3: Infrastructure Operations

**Goal**: Provision → Deploy → Monitor → Optimize

- Terraform MCP (IaC)
- AWS/GCP/Azure MCP (cloud resources)
- Kubernetes + ArgoCD MCP (orchestration)
- Datadog/Grafana MCP (monitoring)
- Slack MCP (notifications)

### Bundle 4: Knowledge Worker

**Goal**: Research → Document → Share → Keep Updated

- Notion/Confluence MCP (knowledge base)
- Google Workspace MCP (email, docs, sheets)
- Firecrawl MCP (web research)
- Brave/Exa MCP (search)
- Figma MCP (design context)
- Obsidian MCP (personal knowledge)

### Bundle 5: Data & Analytics

**Goal**: Query → Analyze → Report → Automate

- Google MCP Toolbox (40+ data sources)
- Supabase/Neon MCP (primary databases)
- LanceDB/Qdrant MCP (vector search)
- PostHog MCP (product analytics)
- n8n/Inngest MCP (workflow automation)

---

## Architecture Recommendations

### 1. Thin Wrapper Pattern (Default)

Most integrations have existing MCP servers. AGH extensions wrap them as subprocesses, adding:

- Session context (memory, workspace awareness)
- Observe event emission (audit trail)
- Credential management (TOML config)
- Cross-tool orchestration (compose multiple MCP servers in one workflow)

### 2. Unified Gateway Pattern (For Categories)

For categories with many providers (databases, search, vector stores), use a single extension that supports multiple backends:

- **Google MCP Toolbox** covers 40+ data sources
- **weave-mcp** covers 11 vector databases
- **MCP Omnisearch** covers 7 search providers
- **Composio** covers thousands of APIs

### 3. Security Boundaries

- 43% of public MCP servers have command injection vulnerabilities
- 7.6% of ClawHub skills contain dangerous patterns
- AGH extensions must enforce permission boundaries, rate limiting, and audit logging
- Read-only by default; write access requires explicit opt-in
- Credentials managed via TOML config, never hardcoded

### 4. AGH Differentiator

Unlike standalone MCP servers, AGH extensions can:

- **Remember** — Store findings in session memory for future reference
- **Learn** — Generate skills from successful workflows
- **Orchestrate** — Compose multiple tools across services in a single session
- **Observe** — Record full audit trail of cross-system operations

---

## Sources

Detailed per-category research files:

- [integrations_devops.md](research/integrations_devops.md) — 37 integrations across DevOps/CI/CD
- [integrations_communication.md](research/integrations_communication.md) — 35 integrations across comms/productivity
- [integrations_data_ai.md](research/integrations_data_ai.md) — 50+ integrations across data/AI/search
- [integrations_specialized.md](research/integrations_specialized.md) — 50 integrations across browser/media/finance/IoT/niche
