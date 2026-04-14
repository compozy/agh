# AGH Extension Research: Data, Database, Search, AI/ML, and Analytics Integrations

> Research date: 2026-04-11
> Scope: Third-party integrations buildable as AGH extensions (subprocess JSON-RPC with Host API)

---

## Summary

The MCP ecosystem has exploded to 21,000+ servers (Glama.ai, April 2026). Nearly every major data, search, and AI platform now ships an official or community MCP server. AGH can leverage these existing servers directly as extensions (they already communicate via JSON-RPC over stdio) or wrap their APIs into AGH-native extensions that tap into AGH's session/memory/skills/observe Host API for deeper integration.

This document catalogs 50+ integrations across 9 categories with concrete AGH agent use cases.

---

## Master Integration Table

| #   | Integration        | Category       | MCP Server Exists | Server Source         | Priority | AGH Value                |
| --- | ------------------ | -------------- | ----------------- | --------------------- | -------- | ------------------------ |
| 1   | PostgreSQL         | Database       | Yes               | Official (Anthropic)  | HIGH     | Core data access         |
| 2   | MySQL              | Database       | Yes               | Community             | MED      | Legacy system access     |
| 3   | MongoDB            | Database       | Yes               | Official (MongoDB)    | MED      | Document store ops       |
| 4   | Redis              | Database       | Yes               | Official (Redis)      | HIGH     | Cache/state mgmt         |
| 5   | Supabase           | Database       | Yes               | Official (hosted+OSS) | HIGH     | Full BaaS agent          |
| 6   | Neon               | Database       | Yes               | Official              | HIGH     | Branch-safe migrations   |
| 7   | PlanetScale        | Database       | Yes               | Official              | MED      | MySQL branching          |
| 8   | Turso              | Database       | Yes               | Official (`--mcp`)    | MED      | Edge SQLite              |
| 9   | CockroachDB        | Database       | Yes               | Community             | LOW      | Distributed SQL          |
| 10  | DynamoDB           | Database       | Yes               | Official (AWS Labs)   | MED      | NoSQL modeling           |
| 11  | Pinecone           | Vector Store   | Via unified       | weave-mcp / MindsDB   | MED      | Managed vector search    |
| 12  | Qdrant             | Vector Store   | Yes               | Official              | HIGH     | OSS vector search        |
| 13  | Weaviate           | Vector Store   | Via unified       | weave-mcp / MindsDB   | MED      | Hybrid search            |
| 14  | Chroma             | Vector Store   | Via unified       | weave-mcp             | LOW      | Prototyping              |
| 15  | Milvus             | Vector Store   | Yes               | Official              | MED      | Industrial-scale vectors |
| 16  | LanceDB            | Vector Store   | Yes               | Community (multiple)  | HIGH     | Embedded, zero-config    |
| 17  | Elasticsearch      | Search         | Yes               | Community             | MED      | Full-text + analytics    |
| 18  | Algolia            | Search         | Yes               | Official (hosted+OSS) | HIGH     | Enterprise search        |
| 19  | Typesense          | Search         | Yes               | Community             | LOW      | Lightweight search       |
| 20  | Meilisearch        | Search         | Yes               | Official              | MED      | Dev-friendly search      |
| 21  | BigQuery           | Data Warehouse | Yes               | Via Dot/Alkemi        | MED      | Google analytics DW      |
| 22  | Snowflake          | Data Warehouse | Yes               | Via Dot/Alkemi        | MED      | Enterprise DW            |
| 23  | ClickHouse         | Data Warehouse | Yes               | Official              | MED      | Real-time analytics      |
| 24  | Databricks         | Data Warehouse | Yes               | Community             | MED      | Lakehouse + ML           |
| 25  | Hugging Face       | AI/ML          | Yes               | Community             | HIGH     | Model discovery          |
| 26  | Replicate          | AI/ML          | Yes               | Community             | MED      | Hosted model inference   |
| 27  | Modal              | AI/ML          | Indirect          | Host MCP on Modal     | MED      | Serverless GPU           |
| 28  | Together AI        | AI/ML          | Via OpenRouter    | OpenRouter proxy      | LOW      | OSS model inference      |
| 29  | Groq               | AI/ML          | Via OpenRouter    | OpenRouter proxy      | MED      | Ultra-fast inference     |
| 30  | OpenRouter         | AI/ML          | Yes               | Community             | HIGH     | Multi-model gateway      |
| 31  | Ollama             | AI/ML          | Yes               | Community             | HIGH     | Local model inference    |
| 32  | Firecrawl          | Web/Scraping   | Yes               | Official              | HIGH     | Web-to-markdown          |
| 33  | Tavily             | Web/Search     | Yes               | Community             | MED      | RAG-optimized search     |
| 34  | Exa                | Web/Search     | Yes               | Community             | HIGH     | Neural search            |
| 35  | Brave Search       | Web/Search     | Yes               | Official              | HIGH     | Privacy-first search     |
| 36  | SerpAPI            | Web/Search     | Yes               | Community             | LOW      | Google SERP scraping     |
| 37  | Crawl4AI           | Web/Scraping   | Yes               | Community             | MED      | Async web crawling       |
| 38  | Jina               | Web/Search     | Yes               | Community             | MED      | Multimodal search        |
| 39  | Amplitude          | Analytics      | Yes               | Community             | LOW      | Behavioral analytics     |
| 40  | Mixpanel           | Analytics      | Yes               | Community (unified)   | LOW      | Product analytics        |
| 41  | PostHog            | Analytics      | Yes               | Official              | MED      | OSS product analytics    |
| 42  | Plausible          | Analytics      | Yes               | Community             | LOW      | Privacy analytics        |
| 43  | Zapier             | Automation     | Yes               | Official              | HIGH     | 8000+ app connectors     |
| 44  | n8n                | Automation     | Yes               | Community             | HIGH     | Self-hosted workflows    |
| 45  | Make               | Automation     | Partial           | Via API               | MED      | Visual workflows         |
| 46  | Pipedream          | Automation     | Yes               | Community             | MED      | Code-first automation    |
| 47  | Temporal           | Automation     | Yes               | Community             | MED      | Durable execution        |
| 48  | Inngest            | Automation     | Yes               | Official              | HIGH     | Durable serverless       |
| 49  | Google MCP Toolbox | Multi-DB       | Yes               | Official (Google)     | HIGH     | 40+ data sources         |
| 50  | DBHub              | Multi-DB       | Yes               | Community             | MED      | Multi-engine gateway     |

---

## Detailed Integration Profiles

### 1. Databases

#### PostgreSQL

- **What it does**: Read-only SQL access, schema inspection, query execution against Postgres databases.
- **MCP server**: Official Anthropic server (`@modelcontextprotocol/server-postgres`, deprecated July 2025). Replaced by Supabase MCP and Neon MCP for managed Postgres. Generic Postgres MCP Pro (CrystalDBA) for self-hosted.
- **AGH use case**: Agent investigating a production bug queries the Postgres database to find anomalous rows, correlates with session logs from AGH's observe system, then writes a migration to fix the data issue and a code patch to prevent recurrence. The agent stores findings in AGH memory for future reference.
- **Status**: Multiple servers exist. Google's MCP Toolbox also covers Postgres.

#### MySQL

- **What it does**: MySQL query execution, schema browsing, data analysis through controlled interface.
- **MCP server**: `designcomputer/mysql_mcp_server` (community). Also covered by DBHub and Google MCP Toolbox.
- **AGH use case**: Agent connects to a legacy MySQL database to understand schema for a migration project, generates an entity-relationship map, and creates Go structs matching the schema.
- **Status**: Community server available. PlanetScale MCP covers MySQL-compatible databases.

#### MongoDB

- **What it does**: Document CRUD, aggregation pipelines, Atlas cloud management, collection operations.
- **MCP server**: Official (`mongodb-js/mongodb-mcp-server`). Supports both local MongoDB and Atlas.
- **AGH use case**: Agent analyzes MongoDB collections to identify slow aggregation pipelines, proposes index optimizations, and validates them by running `explain()` through the MCP server. Results are recorded in AGH observe events.
- **Status**: Official server available.

#### Redis

- **What it does**: Data management, search operations, cache inspection, pub/sub monitoring.
- **MCP server**: Official Redis MCP Server (released Dec 2025). Also: Redis Cloud API MCP, Upstash MCP.
- **AGH use case**: Agent debugging a caching issue inspects Redis keys and TTLs, identifies a cache stampede pattern, then implements a fix with jittered expiration. Uses AGH skills to apply the fix and AGH memory to document the pattern.
- **Status**: Official server available.

#### Supabase

- **What it does**: Full BaaS management -- database design, queries, edge functions, storage, auth, branching. 20+ tools exposed.
- **MCP server**: Official hosted MCP (`https://mcp.supabase.com/mcp`). OAuth 2.1 auth, zero installation. Also: `supabase-community/supabase-mcp` (OSS).
- **AGH use case**: Agent bootstraps an entire backend: creates tables, sets up RLS policies, deploys edge functions, configures storage buckets -- all through natural language within an AGH session. Each step is recorded as an AGH observe event for auditability.
- **Status**: Official, production-ready, most mature BaaS MCP.

#### Neon

- **What it does**: Serverless Postgres management with branch-based migration safety, query tuning, project management. 29 tools with scope-based permissions.
- **MCP server**: Official (remote-first, OAuth + API key). Unique branch-based migration safety.
- **AGH use case**: Agent creates a Neon branch, tests a schema migration on the branch, validates query performance with `explain`, then merges. If the migration causes regressions, the agent rolls back the branch without affecting production. AGH memory stores the migration history.
- **Status**: Official. Rated 4/5 as "best cloud database MCP experience". Neon was acquired by Databricks but operates independently.

#### PlanetScale

- **What it does**: MySQL-compatible serverless database with branching. List orgs, databases, branches, run SQL.
- **MCP server**: Official with OAuth authentication.
- **AGH use case**: Agent creates a database branch for a feature, applies migrations, runs integration tests, then opens a deploy request -- all within an AGH session.
- **Status**: Official. No free tier ($39/mo minimum).

#### Turso

- **What it does**: Edge-deployed SQLite (libSQL). Schema design, data operations, 9 tools.
- **MCP server**: Official via `--mcp` flag. Claude Code integration built in.
- **AGH use case**: Agent sets up an edge database for a mobile app, designs the schema through conversation, generates the libSQL client code, and deploys replicas to multiple regions.
- **Status**: Official. Free tier (5GB, 100 databases).

#### CockroachDB

- **What it does**: Distributed SQL cluster management, monitoring, schema operations, query execution.
- **MCP server**: Community (`bpamiri/cockroachdb-mcp`, `dhartunian/cockroachdb-mcp-server`). Also exposes CockroachDB docs.
- **AGH use case**: Agent monitors cluster health during a deployment, detects node imbalance, and recommends rebalancing operations.
- **Status**: Community servers available.

#### DynamoDB

- **What it does**: Data modeling guidance, design pattern recommendations, cost analysis, code generation. 8 tools.
- **MCP server**: Official (AWS Labs, `awslabs.dynamodb-mcp-server`). Also: community server by Iman Kamyabi for operational management.
- **AGH use case**: Agent analyzes an existing MySQL schema, designs an equivalent DynamoDB single-table model with access patterns, generates cost projections, and produces Go SDK code for the new model. Uses AGH memory to track the migration plan.
- **Status**: Official (AWS Labs). Active development through 2026 (v2.0.8).

---

### 2. Vector Stores

#### Pinecone

- **What it does**: Managed vector similarity search at billion-vector scale.
- **MCP server**: Via unified servers (weave-mcp supports 11 DBs, MindsDB unified MCP).
- **AGH use case**: Agent indexes codebase documentation into Pinecone, then uses semantic search during debugging sessions to find relevant code patterns. AGH memory stores the index metadata.
- **Status**: No first-party MCP server. Available through weave-mcp and MindsDB.

#### Qdrant

- **What it does**: Vector similarity search, code search, semantic memory. ACORN algorithm for filtered HNSW.
- **MCP server**: Official (`qdrant/mcp-server-qdrant`). Supports SSE transport. Can be specialized as code search tool.
- **AGH use case**: Agent stores code snippets and documentation in Qdrant, then during code review, semantically searches for similar patterns, known bugs, or relevant implementations. AGH skills catalog is indexed for semantic discovery.
- **Status**: Official. 1GB free tier forever.

#### Weaviate

- **What it does**: Hybrid search (BlockMax WAND + RSF), vector + keyword search combined.
- **MCP server**: Via weave-mcp and MindsDB unified MCP.
- **AGH use case**: Agent builds a hybrid search index over project documentation and code comments, enabling natural language queries like "find all error handling patterns for database timeouts."
- **Status**: No first-party MCP server. Available through unified servers.

#### Chroma

- **What it does**: Lightweight, developer-friendly vector database for prototyping and small/medium apps.
- **MCP server**: Via weave-mcp.
- **AGH use case**: Agent uses Chroma as a local-first semantic memory store for session context, indexing conversation history for retrieval during long-running tasks.
- **Status**: Community only. Best for prototyping, not production scale.

#### Milvus

- **What it does**: Industrial-scale vector search (40K+ GitHub stars). Full-text, vector, hybrid, and multi-vector search.
- **MCP server**: Official with 5 search tools: `milvus-text-search`, `milvus-vector-search`, `milvus-hybrid-search`, `milvus-multi-vector-search`, `milvus-query`.
- **AGH use case**: Agent indexes a large codebase (millions of functions) into Milvus for semantic code search at scale. During incident response, the agent searches for similar past incidents using vector similarity on error signatures stored in AGH memory.
- **Status**: Official. Best for billion-vector scale.

#### LanceDB

- **What it does**: Embedded vector database with zero configuration. Stores text with vector embeddings for semantic memory. Scales to millions of vectors on disk.
- **MCP server**: Multiple community implementations (Python: `RyanLisse/lancedb_mcp`, Node.js: `vurtnec`, PyPI: `mcp-lance-db`).
- **AGH use case**: AGH uses LanceDB as its local semantic memory backend -- zero external dependencies, embedded in the daemon process. Agent memories, skill descriptions, and session summaries are all indexed for semantic retrieval. This is the most natural fit for AGH's "single-binary, local-first" architecture.
- **Status**: Community servers. MIT licensed. Netflix uses the underlying Lance format in production. HNSW indexing cuts processing time by 50% (July 2025).

---

### 3. Search Engines

#### Elasticsearch

- **What it does**: Full-text search, analytics, index management, document operations. Also "Elasticsearch Memory" variant with hierarchical categorization and semantic search.
- **MCP server**: Community implementation. Also covered by Google MCP Toolbox and weave-mcp.
- **AGH use case**: Agent searches application logs indexed in Elasticsearch to diagnose a production error, correlates with metrics, and generates a root cause analysis document. Findings are stored in AGH memory for the team.
- **Status**: Community server. OpenSearch (fork) also has an official MCP server.

#### Algolia

- **What it does**: Enterprise search and recommendations. Go and Node.js MCP servers plus a hosted MCP server.
- **MCP server**: Official. Multiple implementations: Go (`algolia/mcp`), Node.js (`algolia/mcp-node`), Hosted (fully managed, OAuth).
- **AGH use case**: Agent configures Algolia search indexes for a product catalog, sets up synonyms and ranking rules, tests search quality with sample queries, and deploys. All actions are tracked in AGH observe events for rollback.
- **Status**: Official. Most mature search MCP offering. Usage counts toward existing Algolia plan.

#### Typesense

- **What it does**: In-memory search with Raft clustering. Collection management, document operations, search.
- **MCP server**: Community (`suhail-ak-s/typesense`).
- **AGH use case**: Agent indexes project documentation into Typesense for instant typo-tolerant search, then builds a search UI component.
- **Status**: Community server on PulseMCP.

#### Meilisearch

- **What it does**: Developer-friendly, fast, typo-tolerant search. Indexing and querying via natural language.
- **MCP server**: Official Meilisearch MCP server.
- **AGH use case**: Agent sets up Meilisearch for a knowledge base, configures filterable attributes and ranking, imports documents, and validates search quality.
- **Status**: Official.

---

### 4. Data Warehouses

#### BigQuery

- **What it does**: Google's serverless data warehouse. Query execution, dataset management.
- **MCP server**: Via Dot (GetDot.ai) and Alkemi.ai. Google MCP Toolbox also supports it.
- **AGH use case**: Agent queries BigQuery to analyze user behavior data, identifies a conversion funnel drop-off, generates a report, and creates a Jira ticket with recommendations. Uses AGH memory to track analytics findings over time.
- **Status**: Accessible via unified MCP servers (Dot, Alkemi, Google MCP Toolbox).

#### Snowflake

- **What it does**: Enterprise data warehouse with semi-structured data support.
- **MCP server**: Via Dot and Alkemi. Also MindsDB unified MCP.
- **AGH use case**: Agent runs cost analysis queries on Snowflake, identifies expensive queries consuming credits, proposes optimizations (clustering keys, materialized views), and validates improvements.
- **Status**: Via unified servers. Requires clustering key alignment for optimal MCP use.

#### ClickHouse

- **What it does**: Real-time analytics database. 100+ GB/s scan per node. Column-oriented.
- **MCP server**: Dedicated ClickHouse MCP server listed on official MCP servers repo. ClickHouse acquired Langfuse for LLM observability.
- **AGH use case**: Agent queries ClickHouse for real-time application metrics, builds an anomaly detection query, and sets up an alerting pipeline. Integrates with AGH observe for unified monitoring.
- **Status**: Dedicated MCP server available.

#### Databricks

- **What it does**: Lakehouse platform with Unity Catalog. Workspace interaction, notebook execution, SQL queries.
- **MCP server**: Community (`characat0/databricks`). Covered in official MCP servers repo.
- **AGH use case**: Agent accesses Databricks workspace to run feature engineering notebooks, evaluates ML model performance, and generates a deployment plan.
- **Status**: Community server. Databricks acquired Neon in 2025.

---

### 5. AI/ML Platforms

#### Hugging Face

- **What it does**: Model and dataset discovery, Hub access, repository browsing. Paired with Ollama for local inference.
- **MCP server**: Community MCP for Hub access.
- **AGH use case**: Agent searches Hugging Face for a suitable text embedding model, downloads it via Ollama, benchmarks it against the project's domain data, and configures it as AGH's embedding backend for memory consolidation.
- **Status**: Community server. Can be combined with Ollama for fully local operation.

#### Replicate

- **What it does**: Hosted model inference. Run any open-source model via API.
- **MCP server**: Community MCP bridging Replicate's platform.
- **AGH use case**: Agent uses Replicate to run image generation models for UI mockups, or to run specialized NLP models for code analysis that aren't available locally.
- **Status**: Community server available.

#### Modal

- **What it does**: Serverless GPU infrastructure. Python-first SDK, sub-second cold starts, instant autoscaling.
- **MCP server**: No dedicated MCP server, but Modal is used to host and scale MCP servers. Python SDK defines everything in code.
- **AGH use case**: Agent deploys a custom fine-tuned model on Modal for specialized code review, scales it up during active development hours, and scales to zero overnight. AGH extension wraps Modal's API for on-demand GPU access.
- **Status**: No MCP server -- would need to be built as an AGH extension wrapping Modal's Python SDK.

#### Together AI

- **What it does**: Open-source model inference at scale.
- **MCP server**: Accessible via OpenRouter MCP server (proxy).
- **AGH use case**: Agent switches between different open-source models (Llama, Mistral) via Together AI for cost-optimized inference on different task types.
- **Status**: Via OpenRouter. No dedicated MCP server.

#### Groq

- **What it does**: Ultra-fast LLM inference on custom LPU hardware. Lowest latency provider.
- **MCP server**: Accessible via OpenRouter. Also supported by LibreChat MCP.
- **AGH use case**: Agent uses Groq for rapid code analysis where latency matters -- e.g., real-time code review suggestions during pair programming sessions within AGH.
- **Status**: Via OpenRouter. No dedicated MCP server.

#### OpenRouter

- **What it does**: Unified API gateway to 100+ models from OpenAI, Anthropic, Together AI, Groq, and more.
- **MCP server**: Multiple community implementations (`stabgan/openrouter-multimodal`, `heltonteixeira/openrouterai`).
- **AGH use case**: AGH extension that lets agents dynamically select the best model for each sub-task -- fast model for quick questions, reasoning model for complex analysis, cheap model for bulk operations. Model selection becomes an AGH skill.
- **Status**: Community servers available. High-value integration for AGH's multi-agent architecture.

#### Ollama

- **What it does**: Run LLMs locally. Download and run Llama, Mistral, and hundreds of other models.
- **MCP server**: Community MCP server. Dolphin MCP provides multi-provider support including Ollama.
- **AGH use case**: AGH uses Ollama as a local inference backend for privacy-sensitive operations, embedding generation, and offline operation. Agent can run code analysis models locally without sending code to external APIs.
- **Status**: Community server. Critical for AGH's "local-first" principle.

---

### 6. Web Search & Scraping

#### Firecrawl

- **What it does**: Converts websites to LLM-ready markdown. Crawling, scraping, media parsing, actions (click/scroll/write). 98.3K GitHub stars.
- **MCP server**: Official. Fastest MCP in benchmarks (7s avg, 83% accuracy).
- **AGH use case**: Agent crawls a competitor's documentation site, converts to markdown, indexes in LanceDB for semantic search, and uses the knowledge to implement a similar feature. AGH memory stores the crawled content for team reference.
- **Status**: Official. Best-in-class web scraping MCP.

#### Tavily

- **What it does**: Search engine designed for LLMs and RAG. Concise, fact-checked results to reduce hallucinations.
- **MCP server**: Community. npm and Python SDKs available.
- **AGH use case**: Agent uses Tavily for research tasks -- finding API documentation, checking for known issues, discovering best practices -- and stores findings in AGH memory for the workspace.
- **Status**: Community server. Acquired by Nebius. 1M+ downloads.

#### Exa

- **What it does**: Neural search engine built for AI. Semantic understanding, natural language queries.
- **MCP server**: Community. Part of MCP Omnisearch unified server.
- **AGH use case**: Agent searches for code examples and implementations using semantic queries like "Go implementation of event sourcing with SQLite" and gets highly relevant results. Integrates with AGH skills for automated research workflows.
- **Status**: Community server. Top-tier search quality.

#### Brave Search

- **What it does**: Privacy-focused search with independent index. No tracking. $5/1K queries.
- **MCP server**: Official. Listed in Anthropic's official MCP servers repo.
- **AGH use case**: Agent performs web research for debugging (searching error messages, Stack Overflow, GitHub issues) without leaking user data to third parties. Privacy alignment with AGH's local-first philosophy.
- **Status**: Official. Recommended for privacy-sensitive deployments.

#### SerpAPI

- **What it does**: Google SERP scraping. Returns search metadata (titles, URLs, snippets).
- **MCP server**: Community.
- **AGH use case**: Agent scrapes Google search results for competitive analysis or SEO research tasks.
- **Status**: Community. Reliability risk due to Google anti-scraping measures.

#### Crawl4AI

- **What it does**: Async web crawling with content extraction, metadata retrieval, Google search.
- **MCP server**: Community (`ritvij14/crawl4ai`).
- **AGH use case**: Agent crawls a set of documentation pages to build a local knowledge base for a new library the team is adopting.
- **Status**: Community server.

#### Jina

- **What it does**: Multimodal search (text + images). `r.jina.ai/` URL prefix for instant markdown conversion. Custom neural search pipelines.
- **MCP server**: Community.
- **AGH use case**: Agent uses Jina Reader to quickly convert any URL to markdown for context, and Jina's search for finding relevant documentation across multiple sources.
- **Status**: Community server. Open source with managed cloud option.

---

### 7. Analytics

#### Amplitude

- **What it does**: Deep behavioral analytics, cohort analysis, retention studies, predictive models.
- **MCP server**: Community. Integrates with Amplitude analytics platform.
- **AGH use case**: Agent queries Amplitude to analyze feature adoption metrics before and after a deployment, generates a report, and flags regressions for the team.
- **Status**: Community server. Free tier up to 10M events/month.

#### Mixpanel

- **What it does**: Product analytics with session replay, heatmaps, experiments, feature flags.
- **MCP server**: Community unified analytics MCP (bridges GA4, Mixpanel, PostHog with natural language queries). Released March 2026.
- **AGH use case**: Agent queries Mixpanel funnel data to identify where users drop off, correlates with recent code changes in AGH session history, and proposes UX fixes.
- **Status**: Via unified analytics MCP server.

#### PostHog

- **What it does**: All-in-one product analytics: event tracking, session replay, feature flags, A/B testing, error tracking, LLM analytics, surveys. Open source.
- **MCP server**: Official. Supports latest MCP spec including Streamable HTTP. Free tier available.
- **AGH use case**: Agent queries PostHog to check feature flag states, analyzes experiment results, and toggles flags based on performance data. LLM analytics track AGH's own AI usage costs and model performance. AGH observe events can be forwarded to PostHog for unified observability.
- **Status**: Official. Best OSS analytics MCP. Free up to 1M events/month.

#### Plausible

- **What it does**: Privacy-friendly, cookieless web analytics. Traffic stats, referrers, UTMs. No consent banners.
- **MCP server**: Community. Queries traffic metrics, conversions, time-period comparisons.
- **AGH use case**: Agent monitors website traffic after a deployment to detect anomalies (traffic drops, error spikes) and alerts the team.
- **Status**: Community server.

---

### 8. Automation & Workflow

#### Zapier

- **What it does**: Connects 8,000+ apps. Handles auth, rate limiting, parameter mapping. AI Agents and MCP support.
- **MCP server**: Official. Agent can trigger any Zapier action (Slack message, Jira ticket, Google Sheet update).
- **AGH use case**: Agent completing a code review automatically creates a Jira ticket for found issues, sends a Slack summary to the team channel, and updates a Google Sheet tracking code quality metrics -- all through a single AGH session.
- **Status**: Official. Free tier (100 tasks/month).

#### n8n

- **What it does**: Fair-code workflow automation. Self-hostable. 1,396 nodes (812 core + 584 community). Native MCP support.
- **MCP server**: Community (`czlonkowski/n8n-mcp`). Provides AI assistants with node documentation and properties.
- **AGH use case**: Agent designs and deploys an n8n workflow that monitors a GitHub repo for new issues, classifies them with AI, assigns to the right team member, and updates project tracking -- all orchestrated from an AGH session. Self-hosted n8n means data never leaves infrastructure.
- **Status**: Community MCP. Self-hostable aligns with AGH's local-first philosophy.

#### Make (formerly Integromat)

- **What it does**: Visual workflow builder with branching and parallel processing. 1,000+ integrations.
- **MCP server**: Partial (via API). No dedicated MCP server found.
- **AGH use case**: Agent creates a Make scenario to automate deployment notifications, connecting GitHub Actions to Slack and email.
- **Status**: Would need to be built as an AGH extension wrapping Make's API.

#### Pipedream

- **What it does**: Event-driven serverless automation. Run Python/Node.js/Go/Bash code. 2,700+ apps.
- **MCP server**: Community.
- **AGH use case**: Agent creates a Pipedream workflow that listens for webhooks from the production monitoring system, processes alerts with custom logic, and triggers remediation actions.
- **Status**: Community server.

#### Temporal

- **What it does**: Durable execution for mission-critical workflows. Workflow history, retry logic, long-running processes.
- **MCP server**: Community (`mocksi/temporal-workflows`).
- **AGH use case**: Agent designs and deploys Temporal workflows for data pipeline orchestration, implementing retry logic, compensation handlers, and monitoring. AGH tracks the workflow design decisions in memory.
- **Status**: Community server on PulseMCP.

#### Inngest

- **What it does**: Event-driven durable execution. Serverless + serverful. Retries, concurrency, throttling, rate limiting. Checkpointing for near-zero inter-step latency. AgentKit for multi-agent networks.
- **MCP server**: Official. Dev Server MCP integration for AI-assisted development. Pre-built skills: `inngest-durable-functions`, `inngest-steps`, `inngest-flow-control`, `inngest-middleware`.
- **AGH use case**: Agent uses Inngest to orchestrate a complex deployment pipeline: run tests, build artifacts, deploy to staging, run smoke tests, promote to production, send notifications. Each step is durable and retriable. AGH skills map to Inngest skills for a unified development experience. Inngest's AgentKit could power AGH's own multi-agent orchestration.
- **Status**: Official with MCP + agent skills. Strong alignment with AGH's architecture (event-driven, durable, Go SDK available).

---

### 9. Multi-Database & Unified Servers

#### Google MCP Toolbox for Databases

- **What it does**: Single MCP server supporting 40+ data sources: PostgreSQL, MySQL, SQL Server, Oracle, MongoDB, Redis, Elasticsearch, CockroachDB, ClickHouse, Couchbase, Neo4j, Snowflake, Trino, and more.
- **MCP server**: Official (Google, `googleapis/mcp-toolbox`). Configuration via `tools.yaml`.
- **AGH use case**: Single AGH extension that gives agents access to any database in the infrastructure. Agent can join data across PostgreSQL and MongoDB, or migrate between database engines, without needing separate extensions per database.
- **Status**: Official. Highest coverage of any single MCP server.

#### DBHub

- **What it does**: Unified gateway for PostgreSQL, MySQL, SQLite, DuckDB. Consistent table browsing, schema inspection, safe read-only SQL.
- **MCP server**: Community.
- **AGH use case**: Agent uses DBHub as a universal database explorer for development environments with multiple database types.
- **Status**: Community. Good for development/staging environments.

#### MCP Omnisearch

- **What it does**: Unified access to Tavily, Brave, Kagi, Exa, GitHub, Linkup, and Firecrawl through a single MCP interface.
- **MCP server**: Community (`spences10/mcp-omnisearch`).
- **AGH use case**: Agent performs multi-source research by querying multiple search providers in parallel, deduplicating and ranking results. Single extension replaces 7 individual search integrations.
- **Status**: Community. High-value aggregation.

#### weave-mcp

- **What it does**: Universal CLI for 11 vector databases: Weaviate, Supabase, MongoDB, Milvus, Chroma, Qdrant, Neo4j, Pinecone, OpenSearch, Elasticsearch. Dual transport (HTTP + stdio).
- **MCP server**: Community (`maximilien/weave-mcp`, v0.4.0).
- **AGH use case**: Agent manages vector stores across providers -- migrating embeddings from Chroma (prototyping) to Qdrant (production) or benchmarking retrieval quality across multiple vector databases.
- **Status**: Community. Most comprehensive vector database MCP.

---

## Priority Recommendations for AGH

### Tier 1: Build First (High-value, strong alignment with AGH architecture)

| Integration            | Rationale                                                                                                                                     |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| **LanceDB**            | Embedded, zero-config, local-first -- perfect fit for AGH's memory/consolidation system. Could replace or augment SQLite for semantic search. |
| **Ollama**             | Local model inference. Enables AGH to run without external API dependencies. Critical for air-gapped or privacy-sensitive deployments.        |
| **Supabase**           | Most mature BaaS MCP. Covers database, auth, storage, functions. High developer demand.                                                       |
| **Neon**               | Branch-safe migrations. Best cloud Postgres MCP. Already has MCP tools for AGH's use case.                                                    |
| **Firecrawl**          | Best web scraping MCP. Enables agents to ingest external knowledge. High benchmark scores.                                                    |
| **Inngest**            | Durable workflow orchestration. Go SDK. Event-driven architecture aligns with AGH. AgentKit for multi-agent.                                  |
| **n8n**                | Self-hosted workflow automation. 1,396 nodes. Privacy alignment with AGH.                                                                     |
| **Google MCP Toolbox** | 40+ data sources from one extension. Maximum coverage, minimum effort.                                                                        |

### Tier 2: Build Next (Strong value, good ecosystem)

| Integration      | Rationale                                                                          |
| ---------------- | ---------------------------------------------------------------------------------- |
| **Qdrant**       | Official MCP. Best OSS vector search with free tier. Good for AGH semantic memory. |
| **OpenRouter**   | Multi-model gateway. Lets agents pick optimal model per task.                      |
| **Brave Search** | Official MCP. Privacy-first search. Aligns with AGH values.                        |
| **Exa**          | Neural search. Best semantic search quality for research tasks.                    |
| **PostHog**      | Official MCP. OSS analytics. Could power AGH's own usage analytics.                |
| **Redis**        | Official MCP. Cache/state management for distributed AGH deployments.              |
| **Zapier**       | 8,000+ app connectors. Broadest automation reach.                                  |
| **ClickHouse**   | Real-time analytics. Could power AGH's observe/metrics backend.                    |

### Tier 3: Consider Later (Niche or lower priority)

Everything else -- Amplitude, Mixpanel, Plausible, Typesense, SerpAPI, Make, Chroma, Together AI, Modal, etc.

---

## Architecture Notes for AGH Extensions

### MCP Server Compatibility

Most existing MCP servers communicate via JSON-RPC over stdio -- the same protocol AGH uses for its extensions. This means AGH can potentially wrap existing MCP servers as extensions with minimal glue code:

1. **Direct proxy**: AGH spawns the MCP server as a subprocess, proxies JSON-RPC messages, and adds Host API context (session ID, memory access, observe events).
2. **Native extension**: For high-priority integrations, build a Go-native AGH extension that directly calls the service's API and exposes it through AGH's Host API for deeper integration (memory persistence, skill registration, observe events).
3. **Unified gateway**: For categories with many providers (databases, vector stores, search), build a single AGH extension that supports multiple backends via configuration.

### Key Design Considerations

- **Read-only by default**: Most database MCP servers enforce read-only mode. AGH should follow this pattern and require explicit opt-in for write operations.
- **Credential management**: MCP servers require database credentials. AGH extensions should integrate with AGH's config system (TOML) for credential management, never hardcoding secrets.
- **Latency awareness**: Data warehouses (Snowflake, BigQuery) have 1-30s query latency. AGH extensions should handle this with async patterns and progress reporting via observe events.
- **Local-first preference**: For vector stores and search, prefer embedded solutions (LanceDB, SQLite FTS5) that align with AGH's single-binary architecture. Use cloud services only when scale requires it.

---

## Sources

- [Model Context Protocol Servers (GitHub)](https://github.com/modelcontextprotocol/servers)
- [Glama.ai MCP Directory (21,173 servers)](https://glama.ai/mcp/servers)
- [PulseMCP Server Directory](https://www.pulsemcp.com)
- [FastMCP Top 10 Most Popular MCP Servers](https://fastmcp.me/blog/top-10-most-popular-mcp-servers)
- [50 Most Popular MCP Servers in 2026](https://mcpmanager.ai/blog/most-popular-mcp-servers/)
- [Google MCP Toolbox for Databases](https://github.com/googleapis/mcp-toolbox)
- [Supabase MCP Server](https://supabase.com/features/mcp-server)
- [Neon MCP Server](https://chatforest.com/reviews/neon-mcp-server/)
- [PlanetScale MCP Server](https://planetscale.com/changelog/mcp-server)
- [Turso MCP Server](https://turso.tech/blog/introducing-the-turso-database-mcp-server)
- [AWS DynamoDB MCP Server](https://awslabs.github.io/mcp/servers/dynamodb-mcp-server)
- [Qdrant MCP Server (GitHub)](https://github.com/qdrant/mcp-server-qdrant)
- [Milvus MCP Documentation](https://milvus.io/docs/milvus_and_mcp.md)
- [MindsDB Unified MCP Server for Vector Stores](https://mindsdb.com/unified-model-context-protocol-mcp-server-for-vector-stores)
- [weave-mcp Universal Vector DB CLI](https://github.com/maximilien/weave-mcp)
- [Algolia MCP Server](https://www.algolia.com/developers/lp-mcp)
- [Algolia MCP (Go)](https://github.com/algolia/mcp)
- [Official Meilisearch MCP Server](https://www.pulsemcp.com/servers/meilisearch)
- [ClickHouse MCP and Data Warehouses](https://clickhouse.com/resources/engineering/mcp-data-warehouse-everthing-you-need-to-know)
- [Databricks MCP Server](https://www.pulsemcp.com/servers/characat0-databricks)
- [Firecrawl Web Data API](https://github.com/firecrawl/firecrawl)
- [MCP Omnisearch](https://github.com/spences10/mcp-omnisearch)
- [Web Search for Agents in 2026](https://michaellivs.com/blog/web-search-for-agents-2026/)
- [Best Web Search APIs for AI 2026](https://www.firecrawl.dev/blog/best-web-search-apis)
- [PostHog MCP Server](https://www.pulsemcp.com/servers/posthog)
- [Mixpanel MCP Server](https://www.pulsemcp.com/servers/mixpanel)
- [Inngest MCP Integration](https://www.inngest.com/)
- [n8n MCP Server](https://github.com/czlonkowski/n8n-mcp)
- [Temporal Workflows MCP](https://www.pulsemcp.com/servers/mocksi-temporal-workflows)
- [Zapier MCP (8000+ apps)](https://www.pulsemcp.com)
- [LanceDB MCP Server](https://github.com/RyanLisse/lancedb_mcp)
- [Official Redis MCP Server](https://www.pulsemcp.com/servers/redis)
- [MongoDB MCP Server](https://github.com/mongodb-js/mongodb-mcp-server)
- [CockroachDB MCP Server](https://www.mdskills.ai/mcp-servers/mcp-cockroachdb)
- [OpenRouter MCP Server](https://www.pulsemcp.com/servers/stabgan-openrouter-multimodal)
- [Best MCP Servers for AI and ML 2026](https://fastmcp.me/Blog/best-mcp-servers-for-ai-machine-learning)
- [MCP Benchmark: Top MCP Servers for Web Access](https://aimultiple.com/browser-mcp)
- [Best MCP Servers for Database Management 2026](https://www.dbvis.com/thetable/best-mcp-servers-for-database-management-of-2025/)
