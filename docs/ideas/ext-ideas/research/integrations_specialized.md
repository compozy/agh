# Specialized Third-Party Integration Extensions for AGH

> Research date: 2026-04-11
> Focus: Browser automation, Media, Finance, IoT, Cloud storage, Security, Social media, Email marketing, Maps/Location, and unique/niche integrations that could be built as AGH extensions via subprocess JSON-RPC with Host API access.

---

## Summary Table

| #   | Integration                        | Category            | MCP Server Exists?           | Effort to Build Extension     | Priority |
| --- | ---------------------------------- | ------------------- | ---------------------------- | ----------------------------- | -------- |
| 1   | Playwright                         | Browser Automation  | Yes (official, Microsoft)    | Low -- wrap existing MCP      | High     |
| 2   | Browserbase + Stagehand            | Browser Automation  | Yes (official)               | Low -- wrap existing MCP      | High     |
| 3   | Puppeteer                          | Browser Automation  | Yes (community)              | Low -- wrap existing MCP      | Medium   |
| 4   | BrowserUse                         | Browser Automation  | Yes (open source)            | Low -- wrap existing MCP      | Medium   |
| 5   | YouTube (transcripts + analytics)  | Media / Content     | Yes (multiple)               | Low -- wrap existing MCP      | High     |
| 6   | Spotify                            | Media / Content     | Yes (multiple)               | Low -- wrap existing MCP      | Medium   |
| 7   | ElevenLabs (TTS)                   | Media / Content     | Yes                          | Low -- wrap existing MCP      | Medium   |
| 8   | DALL-E 3 / GPT Image               | Media / Content     | Yes                          | Low -- wrap existing MCP      | High     |
| 9   | Flux (image gen)                   | Media / Content     | Yes (Replicate-based)        | Low -- wrap existing MCP      | Medium   |
| 10  | Luma AI (video gen)                | Media / Content     | Yes                          | Low -- wrap existing MCP      | Medium   |
| 11  | Remotion (programmatic video)      | Media / Content     | No                           | Medium -- build from API      | Low      |
| 12  | Pod Engine (podcast intel)         | Media / Content     | Yes (official)               | Low -- wrap existing MCP      | Low      |
| 13  | Stripe                             | Finance             | Yes (official, 25 tools)     | Low -- wrap existing MCP      | High     |
| 14  | Plaid                              | Finance             | No dedicated MCP             | Medium -- build from REST API | Medium   |
| 15  | Coinbase                           | Finance             | Yes (official)               | Low -- wrap existing MCP      | Medium   |
| 16  | Stock Market (Yahoo Finance, etc.) | Finance             | Yes (multiple)               | Low -- wrap existing MCP      | Medium   |
| 17  | CoinGecko / CoinMarketCap          | Finance             | Yes                          | Low -- wrap existing MCP      | Low      |
| 18  | Home Assistant                     | IoT / Smart Home    | Yes (official + community)   | Low -- wrap existing MCP      | High     |
| 19  | MQTT (via Home Assistant)          | IoT / Smart Home    | Indirect (via HA)            | Medium -- build or compose    | Medium   |
| 20  | AWS S3                             | Cloud Storage       | Yes (official + community)   | Low -- wrap existing MCP      | High     |
| 21  | Cloudflare R2                      | Cloud Storage       | Yes (S3-compat MCP)          | Low -- wrap existing MCP      | Medium   |
| 22  | Backblaze B2                       | Cloud Storage       | Partial (S3-compat)          | Low -- use S3-compat MCP      | Low      |
| 23  | Google Cloud Storage               | Cloud Storage       | Partial (S3-compat interop)  | Medium -- build or adapt      | Low      |
| 24  | Snyk                               | Security            | Yes (official, 11 tools)     | Low -- wrap existing MCP      | High     |
| 25  | SonarQube                          | Security            | Yes (official, 423 stars)    | Low -- wrap existing MCP      | High     |
| 26  | OWASP ZAP (DAST)                   | Security            | Partial (via DevSecOps-MCP)  | Medium -- bundle aggregator   | Medium   |
| 27  | Semgrep                            | Security            | Yes (official)               | Low -- wrap existing MCP      | Medium   |
| 28  | Twitter/X                          | Social Media        | Yes (multiple, fragmented)   | Low -- wrap existing MCP      | High     |
| 29  | Bluesky (AT Protocol)              | Social Media        | Yes (57 tools)               | Low -- wrap existing MCP      | Medium   |
| 30  | LinkedIn                           | Social Media        | Partial (via aggregators)    | Medium -- aggregator or build | Medium   |
| 31  | Reddit                             | Social Media        | Partial (via aggregators)    | Medium -- aggregator or build | Low      |
| 32  | Mastodon                           | Social Media        | Partial (via multi-platform) | Low -- wrap existing          | Low      |
| 33  | SendGrid                           | Email Marketing     | Yes (official, 14+ tools)    | Low -- wrap existing MCP      | Medium   |
| 34  | Resend                             | Email Marketing     | Yes (community)              | Low -- wrap existing MCP      | Medium   |
| 35  | Mailchimp                          | Email Marketing     | Yes (community)              | Low -- wrap existing MCP      | Low      |
| 36  | ConvertKit (Kit)                   | Email Marketing     | No                           | Medium -- build from API      | Low      |
| 37  | Google Maps                        | Maps / Location     | Yes (official + community)   | Low -- wrap existing MCP      | High     |
| 38  | Mapbox                             | Maps / Location     | Yes (official)               | Low -- wrap existing MCP      | Medium   |
| 39  | OpenStreetMap                      | Maps / Location     | Yes (multiple)               | Low -- wrap existing MCP      | Medium   |
| 40  | Blender 3D                         | Niche / Creative    | Yes                          | Low -- wrap existing MCP      | Low      |
| 41  | Unity / Unreal Engine              | Niche / Game Dev    | Yes (both)                   | Low -- wrap existing MCP      | Low      |
| 42  | Minecraft                          | Niche / Game Dev    | Yes                          | Low -- wrap existing MCP      | Low      |
| 43  | ROS (Robot OS)                     | Niche / Robotics    | Yes                          | Low -- wrap existing MCP      | Low      |
| 44  | OctoEverywhere (3D Printer)        | Niche / Hardware    | Yes                          | Low -- wrap existing MCP      | Low      |
| 45  | KiCAD (PCB design)                 | Niche / Engineering | Yes                          | Low -- wrap existing MCP      | Low      |
| 46  | Ableton Live                       | Niche / Music       | Yes                          | Low -- wrap existing MCP      | Low      |
| 47  | Salesforce CRM                     | Niche / Business    | Yes                          | Low -- wrap existing MCP      | Medium   |
| 48  | HubSpot CRM                        | Niche / Business    | Yes                          | Low -- wrap existing MCP      | Medium   |
| 49  | Odoo ERP                           | Niche / Business    | Yes                          | Low -- wrap existing MCP      | Low      |
| 50  | Meta-MCP (Magg)                    | Niche / Agent Infra | Yes                          | Low -- wrap existing MCP      | Medium   |

---

## 1. Browser Automation

### 1.1 Playwright MCP (Microsoft Official)

**What it does:** Full browser automation via the accessibility tree. Agents get structured DOM representation and can click, fill forms, navigate, take screenshots, and extract content across Chromium, Firefox, and WebKit.

**MCP server:** Official -- `@anthropic/mcp-server-playwright` and Microsoft's own Playwright MCP server. Mature, actively maintained.

**AGH use case:** An AGH agent performing end-to-end testing of a web application. The agent spawns a session, uses Playwright MCP to navigate pages, fill forms, assert content, and report test results back via the observe Host API. Combined with session memory, the agent can track regression history across runs.

**Status:** MCP server exists. Extension wraps the existing server as an AGH subprocess.

### 1.2 Browserbase + Stagehand

**What it does:** Cloud-managed browser sessions with AI-native primitives. Stagehand provides three atomic actions -- `act` (perform an action), `extract` (pull data), `observe` (inspect page state) -- bridging brittle selectors with natural language intent. Browserbase manages browser infra (sessions, proxies, bot evasion).

**MCP server:** Official -- `mcp-server-browserbase` (GitHub: browserbase/mcp-server-browserbase). Supports Playwright, Puppeteer, and Patchright as underlying drivers. Pioneer in the MCP space since November 2024.

**AGH use case:** An AGH agent performing competitive intelligence. The agent uses Browserbase to scrape competitor pricing pages that require JavaScript rendering and bot evasion. Stagehand's `extract` primitive pulls structured pricing data without brittle CSS selectors. Results are stored in session memory for trend analysis across runs.

**Status:** MCP server exists. Extension wraps the cloud service.

### 1.3 Puppeteer MCP

**What it does:** Node.js headless browser control. JavaScript-native, best for teams already in the Node ecosystem.

**MCP server:** Community -- multiple implementations available. Lower adoption than Playwright MCP.

**AGH use case:** JavaScript-heavy site interaction where the AGH agent needs to execute JS in-page and extract results.

**Status:** MCP server exists.

### 1.4 BrowserUse

**What it does:** Open-source, self-hosted browser automation for AI agents. Full control over the browser with no cloud dependency.

**MCP server:** Yes -- browser-use MCP server enables browsing from MCP-compatible environments.

**AGH use case:** Privacy-sensitive web automation where all browsing must stay on local infrastructure. An AGH agent performs form submissions and data extraction without sending data to third-party cloud services.

**Status:** MCP server exists.

---

## 2. Media / Content

### 2.1 YouTube (Transcripts + Analytics)

**What it does:** Extract video transcripts, search video content, access channel analytics, manage playlists. Two architectural approaches: yt-dlp-based (no API key needed, transcript-only) and YouTube Data API v3-based (full metadata, 10K daily quota).

**MCP servers:**

- `anaisbetts/mcp-youtube` -- most popular (490+ stars), yt-dlp-based, transcript extraction
- `kimtaeyoon83/mcp-server-youtube-transcript` -- Python-based, pagination for long transcripts, proxy support
- YouTube Data API servers for full metadata access

**AGH use case:** A research agent that takes a YouTube playlist URL, extracts all transcripts, summarizes each video, identifies key themes, and writes a knowledge synthesis document. Uses AGH session memory to build a persistent knowledge base of video content over time.

**Status:** Multiple MCP servers exist. Choose based on need (transcript-only vs. full API access).

### 2.2 Spotify

**What it does:** Control playback, manage playlists, search catalog, analyze listening patterns. Premium required for queue operations.

**MCP servers:**

- `gupta-kush/spotify-mcp` -- 93 tools including smart shuffle, vibe analysis, artist network mapping
- `allensy/spotify-mcp` -- Dockerized, basic playback and search
- Composio Spotify MCP -- via CLI integration
- Zapier Spotify MCP -- automation-focused

**AGH use case:** A personal assistant agent that curates workout playlists based on natural language preferences ("high energy electronic, 130+ BPM, no vocals"). The agent searches the catalog, analyzes track features, builds and saves playlists. Uses AGH memory to learn user preferences over time.

**Status:** Multiple MCP servers exist.

### 2.3 ElevenLabs (Text-to-Speech)

**What it does:** High-quality AI voice generation from text. Useful for podcasters, video creators, audiobook producers.

**MCP server:** Yes -- connects AI assistants to ElevenLabs TTS technology.

**AGH use case:** A content production agent that takes written blog posts and converts them to podcast-style audio narration, selecting appropriate voice profiles and pacing.

**Status:** MCP server exists.

### 2.4 DALL-E 3 / GPT Image Generation

**What it does:** Generate images from text prompts via OpenAI's API. Strongest at understanding complex instructions and compositional prompts.

**MCP server:** Yes -- dedicated DALL-E 3 MCP server (`sammyl720/dall-e-image-generator`), plus Azure OpenAI variants.

**AGH use case:** A documentation agent that generates diagrams, illustrations, and hero images for technical documentation based on content descriptions. Images are stored as session artifacts and linked to the generated docs.

**Status:** MCP server exists.

### 2.5 Flux (Black Forest Labs)

**What it does:** High-quality image generation with excellent text rendering in images, camera-accurate optical characteristics, and fast iteration via Schnell variant.

**MCP server:** Yes -- `awkoy/replicate-flux-mcp` via the Replicate API.

**AGH use case:** A design agent that generates UI mockup screenshots with readable text, proper typography, and realistic device frames for product presentations.

**Status:** MCP server exists (Replicate-based).

### 2.6 Luma AI (Video Generation)

**What it does:** AI-driven video generation from text descriptions or parameters. Creates short-form video content programmatically.

**MCP server:** Yes -- listed on PulseMCP, integrates with Luma AI's API.

**AGH use case:** A marketing agent that generates product demo videos from text descriptions, creating short clips showcasing features with camera movements and transitions.

**Status:** MCP server exists.

### 2.7 Remotion (Programmatic Video in React)

**What it does:** Create videos using React components. Compose, render, and export MP4/WebM videos programmatically with full control over every frame.

**MCP server:** No dedicated MCP server found.

**AGH use case:** An agent that generates data visualization videos -- takes a dataset, builds animated React charts using Remotion, and renders a narrated video walkthrough of key metrics. Requires building a custom extension.

**Status:** No MCP server. Would need to be built as a custom AGH extension wrapping Remotion's CLI/API.

### 2.8 Pod Engine (Podcast Intelligence)

**What it does:** Tracks every active podcast, aggregates data from Apple Podcasts, Spotify, YouTube, Instagram, Twitter, and social networks. Monitors rankings across all major charts with historical data.

**MCP server:** Yes -- world's first podcast intelligence MCP server integration (official from Pod Engine).

**AGH use case:** A podcast analytics agent that monitors chart positions, tracks competitor shows, and generates weekly performance reports with trend analysis.

**Status:** MCP server exists (official).

---

## 3. Finance

### 3.1 Stripe (Payments + Billing)

**What it does:** 25 tools covering the core payment lifecycle: customers, products, prices, invoices, subscriptions, refunds, and documentation search. Part of Stripe's Agentic Commerce Suite (March 2026).

**MCP server:** Official -- `stripe/ai` monorepo (1.4K stars, v0.3.1). Supports OAuth and API key auth. Also includes `@stripe/agent-toolkit` for framework integrations and `@stripe/token-meter` for usage metering.

**AGH use case:** A billing operations agent that monitors subscription churn, generates invoices for overdue accounts, creates discount codes for retention campaigns, and produces revenue reports. Uses AGH memory to track customer interaction history and billing patterns.

**Notable:** Stripe also launched the Machine Payments Protocol (MPP) in March 2026 for agent-to-agent micropayments -- complementary to MCP (MCP lets agents use Stripe, MPP lets agents pay through Stripe).

**Status:** Official MCP server exists with 25 tools.

### 3.2 Plaid (Banking Data)

**What it does:** Connects consumer bank accounts to financial applications. Provides account balances, transaction history, identity verification, credit reporting, and anti-fraud services. Customers include Robinhood, Coinbase, Block, and Affirm.

**MCP server:** No dedicated MCP server found.

**AGH use case:** A personal finance agent that connects to user bank accounts via Plaid, categorizes transactions, identifies spending patterns, and generates monthly budget reports. Could trigger alerts when unusual spending is detected.

**Status:** No MCP server. Would need to be built wrapping Plaid's REST API. Plaid's Link flow (browser-based auth) adds complexity.

### 3.3 Coinbase

**What it does:** Crypto wallet management, onramps, stablecoin transactions. Released Payments MCP through Developer Platform in late 2025. Also developed x402 protocol for per-request micropayments.

**MCP server:** Yes (official) -- part of Coinbase Developer Platform.

**AGH use case:** A crypto portfolio management agent that monitors wallet balances, executes trades based on predefined strategies, and generates tax reporting data. Uses AGH memory to track portfolio history.

**Status:** Official MCP server exists.

### 3.4 Stock Market Data

**What it does:** Real-time prices, fundamentals, earnings data, stock screening, and historical charts.

**MCP servers:**

- Yahoo Finance MCP -- real-time and historical market data
- Financial Datasets MCP -- comprehensive financial analyst toolkit
- 11+ servers ranked by Lambda Finance for different capabilities

**AGH use case:** A financial research agent that screens stocks by fundamental criteria, analyzes earnings reports, compares peer companies, and generates investment research notes. Uses AGH session memory to maintain a watchlist and track thesis evolution.

**Status:** Multiple MCP servers exist.

### 3.5 CoinGecko / CoinMarketCap

**What it does:** Crypto market data, token prices, market caps, trading volumes, exchange data.

**MCP servers:**

- CoinMarketCap MCP (`szcharlesji/coinmarketcap`)
- CoinGecko official MCP server

**AGH use case:** A crypto market monitoring agent that tracks token prices, identifies significant movements, and generates daily market briefs.

**Status:** MCP servers exist.

---

## 4. IoT / Smart Home

### 4.1 Home Assistant

**What it does:** Central smart home control hub. The MCP integration (introduced in HA 2025.2) exposes the Assist API, allowing AI agents to control devices and entities. Supports OAuth authentication and Streamable HTTP protocol.

**MCP server:** Yes -- official Home Assistant MCP Server integration (built-in since 2025.2). Also community servers: `homeassistant-ai/ha-mcp` and `tevonsb/homeassistant-mcp`.

**AGH use case:** A home automation agent that manages daily routines -- adjusts lighting based on time of day, controls HVAC based on weather forecasts and occupancy, monitors security cameras, and generates energy usage reports. Uses AGH memory to learn household patterns and optimize automations over time.

**Status:** Official MCP server exists (built into Home Assistant). Multiple community alternatives.

### 4.2 MQTT

**What it does:** Lightweight messaging protocol for IoT device communication. Gold standard for Zigbee2MQTT, ESPHome, and custom sensors. Mosquitto is the dominant broker implementation.

**MCP server:** No standalone MQTT MCP server found. MQTT devices are typically accessed through Home Assistant's MQTT integration.

**AGH use case:** An industrial monitoring agent that subscribes to MQTT topics from factory sensors, detects anomalies in temperature/pressure readings, and triggers alerts. Would require a custom extension bridging MQTT subscriptions to AGH's event system.

**Status:** No standalone MCP server. Accessed indirectly via Home Assistant or would need custom extension.

---

## 5. Cloud Storage

### 5.1 AWS S3

**What it does:** Browse buckets, read/write objects, generate presigned URLs, run SQL queries against S3 Tables, CSV-to-table conversion.

**MCP servers:**

- Official AWS: S3 Tables MCP Server (`awslabs/mcp/servers/s3-tables-mcp-server`)
- `txn2/mcp-s3` -- S3 and S3-compatible storage, multi-account support
- `gangadharrr/aws-s3-mcp` -- full bucket and object management
- `aws-samples/sample-mcp-server-s3` -- official AWS sample

**AGH use case:** A data pipeline agent that monitors S3 buckets for new files, processes incoming CSVs into structured tables, generates presigned URLs for sharing, and manages lifecycle policies. Uses AGH memory to track processing history and data lineage.

**Status:** Multiple MCP servers exist, including official AWS implementations.

### 5.2 Cloudflare R2

**What it does:** S3-compatible object storage with zero egress fees. Endpoint: `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`.

**MCP server:** Yes -- via S3-compatible MCP servers (e.g., `txn2/mcp-s3` configured with R2 endpoint). Also `am1010101/s3-mcp-server` on LobeHub.

**AGH use case:** A static asset management agent that uploads, organizes, and manages website assets on R2, generating cache-busted URLs and cleaning up unused files.

**Status:** Works via S3-compatible MCP servers.

### 5.3 Backblaze B2

**What it does:** Low-cost object storage ($0.006/GB). S3-compatible API.

**MCP server:** Works via S3-compatible MCP servers with appropriate endpoint configuration.

**AGH use case:** A backup management agent that handles archival storage, verifies backup integrity, and manages retention policies.

**Status:** Works via S3-compatible MCP servers. No dedicated server needed.

### 5.4 Google Cloud Storage

**What it does:** Google's object storage with S3-compatible interoperability mode.

**MCP server:** No dedicated MCP server found. GCS's S3-compatible interop mode may allow some S3 MCP servers to work.

**AGH use case:** A multi-cloud storage agent that manages assets across GCS, S3, and R2, optimizing placement by cost and access patterns.

**Status:** No dedicated MCP server. Partial support via S3 interop. May need custom extension for full GCS API access.

---

## 6. Security Tools

### 6.1 Snyk

**What it does:** 11 tools covering SAST, SCA, IaC scanning, container scanning, SBOM, and AI-BOM from a single integration. The most comprehensive security scanning MCP server available.

**MCP server:** Official -- `snyk/mcp-server-snyk` (v1.6.1, Apache 2.0). Vendor-controlled, closed to contributions.

**Additionally:** `snyk/agent-scan` (1.9K stars) is a meta-security tool that scans MCP servers themselves for prompt injection, tool poisoning, and credential handling issues.

**AGH use case:** A security review agent that scans project dependencies for known vulnerabilities, checks IaC configurations for misconfigurations, generates SBOM reports, and creates remediation PRs. Uses AGH memory to track vulnerability history and remediation progress.

**Status:** Official MCP server exists with 11 tools.

### 6.2 SonarQube

**What it does:** Static analysis for code quality and security. 6,500+ rules across 35+ languages. AI CodeFix generates LLM-powered fix suggestions. Compliance reporting for OWASP Top 10, CWE Top 25, PCI DSS.

**MCP server:** Official -- `sonarqube-mcp-server` (423 stars, 321 commits). Supports SonarQube Cloud (zero-config) and self-managed Docker deployment.

**AGH use case:** A code quality agent that runs SonarQube analysis on every PR, categorizes issues by severity, generates fix suggestions using AI CodeFix, and tracks technical debt trends over time. Uses AGH memory to maintain quality baselines.

**Status:** Official MCP server exists. Largest community in the code security MCP category.

### 6.3 OWASP ZAP (DAST)

**What it does:** Dynamic application security testing -- runtime vulnerability scanning of web applications.

**MCP server:** No standalone MCP server. Available via `DevSecOps-MCP` which bundles Semgrep + Bandit + SonarQube + OWASP ZAP + Trivy (6 tools covering SAST, DAST, IAST, and SCA).

**AGH use case:** A penetration testing agent that runs DAST scans against staging environments, identifies runtime vulnerabilities, and generates security reports with remediation guidance.

**Status:** Available via DevSecOps-MCP aggregator. No standalone server.

### 6.4 Semgrep

**What it does:** Fast, lightweight static analysis with custom rule support. Covers SAST and secrets detection.

**MCP server:** Official -- Semgrep MCP server.

**AGH use case:** A pre-commit security agent that scans code changes for security patterns, enforces custom security rules, and blocks merges with critical findings.

**Status:** Official MCP server exists.

---

## 7. Social Media

### 7.1 Twitter/X

**What it does:** Post tweets, search content, manage drafts, publish threads.

**MCP servers:**

- `EnesCinr/twitter-mcp` (375 stars, TypeScript, MIT) -- most popular, posting and searching
- `vidhupv/x-mcp` (61 stars, Python, MIT) -- draft management and thread publishing
- 8+ competing servers (most fragmented category)

**AGH use case:** A social media manager agent that drafts tweets based on product updates, schedules thread publications, monitors mentions and engagement, and generates weekly analytics reports. Uses AGH memory to track content performance and audience growth.

**Status:** Multiple MCP servers exist. Category is fragmented.

### 7.2 Bluesky (AT Protocol)

**What it does:** Full AT Protocol integration with 57 tools covering streaming, batch operations, analytics, posting, and social graph management.

**MCP server:** `cameronrye/atproto-mcp` (TypeScript, MIT) -- 57 tools, the most comprehensive.

**AGH use case:** A decentralized social agent that cross-posts content, monitors the firehose for brand mentions, and builds audience analytics dashboards.

**Status:** MCP server exists with 57 tools.

### 7.3 LinkedIn

**What it does:** Professional network posting, engagement tracking, lead generation.

**MCP server:** Partial -- via aggregators like Anysite, Pluggo AI, and Publora (multi-platform). No standalone LinkedIn MCP with full API access.

**AGH use case:** A professional branding agent that drafts and publishes LinkedIn articles from internal knowledge bases, optimizes posting times, and tracks engagement metrics.

**Status:** Available via aggregators. Full standalone server would need LinkedIn API access (restricted).

### 7.4 Reddit

**What it does:** Post and comment, monitor subreddits, track engagement.

**MCP server:** Partial -- via aggregators (Anysite, Pluggo AI, recast-mcp). No prominent standalone Reddit MCP.

**AGH use case:** A community engagement agent that monitors relevant subreddits for questions, drafts helpful responses, and tracks karma and post performance.

**Status:** Available via aggregators. Standalone server possible via Reddit API.

### 7.5 Mastodon

**What it does:** Fediverse social networking via ActivityPub protocol.

**MCP server:** Partial -- via multi-platform servers like Publora and simplex CLI tool.

**AGH use case:** A fediverse presence agent that cross-posts content across Mastodon instances and monitors federated timelines.

**Status:** Available via multi-platform tools.

### 7.6 Multi-Platform (recast-mcp)

**What it does:** Turns any URL into platform-specific content: LinkedIn posts, Twitter threads, Reddit posts, and newsletter content.

**MCP server:** `DrewDawson2027/recast-mcp` -- content repurposing across platforms.

**AGH use case:** A content distribution agent that takes a blog post URL and generates platform-optimized versions for each social network, then schedules publication.

**Status:** MCP server exists.

---

## 8. Email Marketing

### 8.1 SendGrid

**What it does:** Email delivery, marketing campaigns, contact list management, template management, deliverability monitoring.

**MCP servers:**

- `garethcurl/sendgrid-mcp` -- open source, Flask/Python, stats and template management
- MCPBundles SendGrid -- 20 tools, remote hosted
- Multiple community alternatives

**AGH use case:** A marketing operations agent that creates email campaigns from content briefs, manages contact segments, monitors deliverability metrics, and A/B tests subject lines. Uses AGH memory to track campaign performance history.

**Status:** Multiple MCP servers exist.

### 8.2 Resend

**What it does:** Modern email API for developers. Plain text and HTML email sending with scheduling, configurable sender/reply-to addresses.

**MCP server:** Yes -- community MCP integration with Resend API.

**AGH use case:** A transactional email agent that sends order confirmations, password resets, and notification emails as part of automated workflows.

**Status:** MCP server exists (community).

### 8.3 Mailchimp

**What it does:** Email campaign management, creation, sending, automation workflows, and analytics.

**MCP server:** `bryangsmith/mailchimp` -- FastMCP framework, campaign management and analytics.

**AGH use case:** A newsletter agent that drafts weekly digests from curated content, segments audiences, and automates drip campaigns.

**Status:** MCP server exists (community).

### 8.4 ConvertKit (Kit)

**What it does:** Creator-focused email platform with visual automations, subscriber management, and integrated commerce for digital products.

**MCP server:** No dedicated MCP server found.

**AGH use case:** A creator economy agent that manages subscriber segments, triggers automation sequences based on behavior, and tracks digital product sales.

**Status:** No MCP server. Would need to be built from Kit's REST API.

---

## 9. Maps / Location

### 9.1 Google Maps

**What it does:** 18+ tools: geocoding, reverse geocoding, nearby search, place details, directions, distance matrix, elevation, timezone, weather, air quality, static maps, batch geocoding (50 addresses), route optimization (25 stops), local rank tracking.

**MCP servers:**

- Official Anthropic-listed Google Maps MCP (`modelcontextprotocol/google-maps`)
- `cablate/mcp-google-map` -- 18 tools, advanced features like route planning and competitor rank tracking

**AGH use case:** A logistics agent that optimizes multi-stop delivery routes, calculates ETAs with real-time traffic, geocodes customer addresses in batch, and generates static map visualizations for reports.

**Status:** Official and community MCP servers exist.

### 9.2 Mapbox

**What it does:** Geocoding, POI search across millions of businesses, multi-modal routing with real-time traffic, travel time matrices, route optimization, map matching, isochrone generation, static map rendering.

**MCP server:** Official -- `mapbox/mcp-server` (GitHub). Includes DevKit with documentation tools and Developer Playgrounds. API token stays local.

**AGH use case:** A real estate analysis agent that generates isochrone maps showing walkable areas, calculates commute times to offices, searches nearby amenities, and produces static map images for property reports.

**Status:** Official MCP server exists.

### 9.3 OpenStreetMap

**What it does:** Open geospatial data, geocoding, POI search, SQL queries against OSM data with PostGIS.

**MCP servers:**

- `wiseman/osm-mcp` -- PostgreSQL/PostGIS integration, web-based map viewing
- `jagan-shanmugam/open-streetmap-mcp` -- geocoding and location services

**AGH use case:** A geospatial research agent that queries OpenStreetMap for infrastructure data (roads, buildings, parks), generates custom maps, and performs spatial analysis without API costs.

**Status:** Multiple MCP servers exist.

---

## 10. Unique / Novel / Niche Integrations

### 10.1 Ableton Live (Music Production)

**What it does:** AI-assisted music production with tools like `create_midi_track`, `load_instrument_or_effect`, `fire_clip`. Generate musical ideas, set up tracks, control playback.

**MCP server:** Yes -- Ableton MCP.

**AGH use case:** A music production assistant agent that generates MIDI drum patterns, loads instruments, and arranges track sections based on natural language descriptions of desired sounds.

### 10.2 ROS / ROS2 (Robot Operating System)

**What it does:** Converts natural language commands into ROS/ROS2 control commands for robot manipulation.

**MCP server:** `lpigeon/ros-mcp-server` for general ROS, `lpigeon/unitree-go2-mcp-server` for Unitree Go2 robot.

**AGH use case:** A robotics control agent that translates high-level task descriptions ("pick up the red box and place it on the shelf") into ROS command sequences, monitors execution, and handles errors.

### 10.3 NVIDIA Isaac Sim

**What it does:** Natural language control of NVIDIA Isaac Sim for robotics simulation, OpenUSD scene manipulation, and physics simulation.

**MCP server:** `omni-mcp/isaac-sim-mcp`.

**AGH use case:** A simulation engineer agent that sets up robot training environments, configures physics parameters, and runs simulation batches for reinforcement learning.

### 10.4 OctoEverywhere (3D Printing)

**What it does:** Live 3D printer state, webcam snapshots, printer control via AI agents.

**MCP server:** Yes -- OctoEverywhere 3D Printing MCP.

**AGH use case:** A print farm management agent that monitors multiple 3D printers, detects print failures from webcam snapshots, pauses failed prints, and queues new jobs.

### 10.5 KiCAD (PCB Design)

**What it does:** Direct interaction with KiCAD for printed circuit board design via LLMs.

**MCP server:** Yes -- KiCAD MCP.

**AGH use case:** An electronics design agent that generates PCB layouts from circuit descriptions, checks design rules, and exports manufacturing files.

### 10.6 Minecraft (Mineflayer)

**What it does:** Controls a Minecraft character in real-time. Build structures, explore, interact with game environment through natural language.

**MCP server:** Yes -- Minecraft MCP via Mineflayer API.

**AGH use case:** An AI game companion that builds structures from architectural descriptions, automates resource gathering, and navigates complex environments.

### 10.7 Unity / Unreal Engine

**What it does:** Bridge between AI assistants and game engines. Manage assets, control scenes, edit scripts, automate tasks. Unreal version supports automatic scene generation.

**MCP servers:** Unity MCP and Unreal Engine MCP (both exist).

**AGH use case:** A game development assistant agent that generates game levels, populates scenes with assets, writes interaction scripts, and automates build processes.

### 10.8 Meta-MCP / Magg (Self-Extending Agents)

**What it does:** A meta-MCP server that acts as a universal hub. LLMs can autonomously discover, install, and orchestrate multiple MCP servers -- giving AI agents the power to extend their own capabilities on-demand.

**MCP server:** `sitbon/magg`.

**AGH use case:** An AGH agent that can discover and install new capabilities at runtime. When asked to perform a task it lacks tools for, it queries Magg to find and install the appropriate MCP server, then uses it. This is particularly powerful with AGH's skills system -- the agent could permanently learn new tool integrations.

### 10.9 Agoragentic (Agent Marketplace)

**What it does:** Agent-to-agent marketplace where AI agents discover, invoke, and pay for services from other agents using USDC on Base L2.

**MCP server:** `rhein1/agoragentic-integrations`.

**AGH use case:** An AGH agent that can hire other specialized agents for subtasks -- e.g., paying a specialized data analysis agent for a report, or a design agent for image creation -- settling payments autonomously via crypto.

### 10.10 ImmoStage (Real Estate Virtual Staging)

**What it does:** AI virtual staging for real estate: stage empty rooms, beautify floor plans into 3D renders, classify room images, generate property descriptions, style recommendations.

**MCP server:** `LarryWalkerDEV/mcp-immostage`.

**AGH use case:** A real estate marketing agent that takes empty room photos, virtually stages them with appropriate furniture styles, generates listing descriptions, and produces marketing materials.

### 10.11 Metropolitan Museum / Smithsonian (Cultural Archives)

**What it does:** Access museum collections -- search artworks, retrieve metadata, browse exhibitions.

**MCP servers:** `mikechao/metmuseum-mcp` (Met Museum), `molanojustin/smithsonian-mcp` (Smithsonian).

**AGH use case:** A cultural research agent that explores museum collections by theme, period, or artist, compiles curated exhibitions, and generates educational materials.

### 10.12 Excalidraw (Diagramming)

**What it does:** Programmatic canvas toolkit to create, edit, and export diagrams via AI agents with real-time canvas sync.

**MCP server:** Yes -- Excalidraw MCP.

**AGH use case:** An architecture documentation agent that generates system diagrams from code analysis, maintains living architecture documents, and exports diagrams for presentations.

### 10.13 Salesforce / HubSpot / Attio CRM

**What it does:** CRM operations -- SOQL queries, record management, contact search, deal management, engagement timelines, dynamic schema discovery.

**MCP servers:** Salesforce MCP, HubSpot MCP, Attio MCP -- all exist.

**AGH use case:** A sales operations agent that syncs meeting notes to CRM records, updates deal stages, generates pipeline reports, and drafts follow-up emails based on conversation history.

### 10.14 Cisco pyATS (Network Management)

**What it does:** Structured, model-driven interaction with Cisco network devices via natural language.

**MCP server:** `automateyournetwork/pyATS_MCP`.

**AGH use case:** A network operations agent that monitors device health, diagnoses connectivity issues, generates network topology maps, and automates configuration changes.

### 10.15 Discogs (Music Database)

**What it does:** Interact with the Discogs music database API for vinyl collection management, pricing, and discography research.

**MCP server:** `cswkim/discogs-mcp-server`.

**AGH use case:** A music collection agent that catalogs vinyl records, tracks market values, and identifies rare pressings.

---

## Recommended Priority Bundles for AGH

### Tier 1 -- High Impact, Low Effort (wrap existing MCP servers)

These integrations have mature, official MCP servers and address the most common agent use cases:

| Extension            | Why                                                                             |
| -------------------- | ------------------------------------------------------------------------------- |
| **Playwright**       | Web automation is foundational for any agent system                             |
| **Stripe**           | Payment operations are critical for business agents                             |
| **S3**               | Cloud storage access is a basic infrastructure need                             |
| **Snyk + SonarQube** | Security scanning is essential for code-focused agents                          |
| **Google Maps**      | Location intelligence enables logistics, real estate, and local business agents |
| **YouTube**          | Content analysis and research is high-demand                                    |
| **Home Assistant**   | Smart home control is the killer IoT use case                                   |
| **Twitter/X**        | Social media management is a top agent use case                                 |

### Tier 2 -- Medium Impact, Low-Medium Effort

| Extension                   | Why                                                                                |
| --------------------------- | ---------------------------------------------------------------------------------- |
| **Browserbase + Stagehand** | Cloud browsers with bot evasion for scraping                                       |
| **DALL-E / Flux**           | Image generation for content and design agents                                     |
| **Coinbase**                | Crypto operations and agent-to-agent payments                                      |
| **Mapbox**                  | Advanced geospatial features beyond Google Maps                                    |
| **Bluesky**                 | Growing decentralized social platform                                              |
| **SendGrid / Resend**       | Email operations for marketing and transactional workflows                         |
| **Spotify**                 | Music/audio control for personal assistant agents                                  |
| **Stock Market**            | Financial analysis and monitoring                                                  |
| **Meta-MCP (Magg)**         | Self-extending agent capabilities -- deeply aligned with AGH's extensibility model |

### Tier 3 -- Niche but Differentiated

| Extension                    | Why                                                       |
| ---------------------------- | --------------------------------------------------------- |
| **Ableton Live**             | Creative AI for music production                          |
| **ROS / Isaac Sim**          | Robotics control opens entirely new agent domains         |
| **3D Printing**              | Physical-world manufacturing automation                   |
| **KiCAD**                    | Hardware design automation                                |
| **CRM (Salesforce/HubSpot)** | Enterprise sales automation                               |
| **Agoragentic**              | Agent marketplace enables AGH agents to hire other agents |

---

## Architecture Notes for AGH Integration

All of these integrations follow the same pattern for AGH:

1. **Extension subprocess** wraps the MCP server (or API client) as a JSON-RPC subprocess
2. **Host API access** allows the extension to read/write session memory, emit observe events, and query skills
3. **Configuration** via TOML agent definitions in AGH config, specifying the MCP server command and environment variables (API keys, endpoints)
4. **Lifecycle** managed by AGH's ACP layer -- spawn on session start, communicate via stdio JSON-RPC, clean shutdown on session end

For MCP-server-based integrations, the extension is essentially a thin adapter: launch the MCP server subprocess, translate between AGH's Host API protocol and MCP's tool/resource protocol, and handle authentication credential management.

For integrations without existing MCP servers (Plaid, ConvertKit, Remotion, standalone MQTT), the extension would implement the tool interface directly against the service's REST API or SDK.
