# DevOps, CI/CD, and Developer Tool Integrations for AGH Extensions

> Research date: 2026-04-11
> Purpose: Identify third-party integrations that could be built as AGH extensions (subprocess JSON-RPC with Host API access to session/memory/skills/observe).

---

## Summary

The MCP (Model Context Protocol) ecosystem has matured dramatically since Anthropic's initial release in late 2024. As of early 2026, PulseMCP indexes over 12,000 MCP servers, and the protocol is now governed by the Linux Foundation's Agentic AI Foundation. All three major cloud providers (AWS, Azure, GCP) have official MCP servers, and most major DevOps tools have either first-party or high-quality community MCP servers.

AGH extensions can leverage this ecosystem by wrapping existing MCP servers or building native integrations via tool APIs. The key AGH differentiator is the Host API -- extensions can combine external tool access with AGH's session memory, skills, and observability to create stateful, context-aware automation workflows that go beyond what standalone MCP servers offer.

---

## Integration Summary Table

| Category                     | Tool                | MCP Server Exists?                         | Maturity   | AGH Extension Priority |
| ---------------------------- | ------------------- | ------------------------------------------ | ---------- | ---------------------- |
| **Version Control**          | GitHub              | Yes (official + community)                 | Production | HIGH                   |
|                              | GitLab              | Yes (official)                             | Production | HIGH                   |
|                              | Bitbucket           | Yes (Atlassian remote MCP)                 | Beta       | MEDIUM                 |
| **Project Management**       | Linear              | Yes (community)                            | Production | HIGH                   |
|                              | Jira                | Yes (Atlassian remote MCP + mcp-atlassian) | Production | HIGH                   |
|                              | Shortcut            | No official server found                   | N/A        | LOW                    |
|                              | Notion              | Yes (official hosted + self-hosted)        | Production | HIGH                   |
| **CI/CD**                    | GitHub Actions      | Yes (community)                            | Production | HIGH                   |
|                              | CircleCI            | Yes (official)                             | Production | MEDIUM                 |
|                              | Jenkins             | Yes (official plugin)                      | Production | MEDIUM                 |
|                              | ArgoCD              | Yes (K8s MCP Toolkit)                      | Production | HIGH                   |
| **Deployment Platforms**     | Vercel              | Yes (official handler)                     | Production | MEDIUM                 |
|                              | Netlify             | Yes (community)                            | Community  | LOW                    |
|                              | Railway             | Yes (official)                             | Production | MEDIUM                 |
|                              | Fly.io              | Yes (unified deployment MCP)               | Community  | LOW                    |
| **Code Quality**             | SonarQube           | Yes (official by SonarSource)              | Production | HIGH                   |
|                              | Semgrep             | Yes (built into binary)                    | Production | HIGH                   |
|                              | Snyk                | Yes (official, 11 tools)                   | Production | HIGH                   |
|                              | Trivy               | Yes (plugin)                               | Production | MEDIUM                 |
|                              | Dependabot/Renovate | No MCP server                              | N/A        | MEDIUM (build)         |
| **Infrastructure**           | Terraform           | Yes (HashiCorp official)                   | Production | HIGH                   |
|                              | Pulumi              | Yes (official)                             | Production | MEDIUM                 |
|                              | Docker              | Yes (community)                            | Community  | MEDIUM                 |
|                              | Kubernetes          | Yes (multiple: kubectl, k8m, Lens)         | Production | HIGH                   |
|                              | AWS                 | Yes (official, 60+ servers)                | Production | HIGH                   |
|                              | GCP                 | Yes (official, preview)                    | Preview    | MEDIUM                 |
|                              | Azure               | Yes (official)                             | Production | MEDIUM                 |
| **Monitoring/Observability** | Sentry              | Yes (official + monitoring)                | Production | HIGH                   |
|                              | Datadog             | Yes (official, GA March 2026)              | Production | HIGH                   |
|                              | Grafana             | Yes (official)                             | Production | HIGH                   |
|                              | PagerDuty           | Yes (community)                            | Community  | HIGH                   |
| **Documentation**            | Notion              | Yes (official)                             | Production | HIGH                   |
|                              | Confluence          | Yes (Atlassian remote MCP)                 | Beta       | MEDIUM                 |
|                              | Mintlify            | Yes (auto-generated)                       | Production | LOW                    |
|                              | ReadMe              | Yes (auto-generated)                       | Production | LOW                    |
| **Communication**            | Slack               | Yes (official)                             | Production | HIGH                   |
|                              | Discord             | No official (5 community servers)          | Community  | LOW                    |

---

## Detailed Integration Descriptions

### 1. Version Control & Code Hosting

#### GitHub MCP

**What it does:** Full GitHub API access -- create/review PRs, manage issues, search code, read file contents, manage branches, configure Actions, query commit history and diffs.

**Existing MCP servers:** Multiple options exist. The official GitHub MCP server plus community alternatives like `github-mcp-server` and `ko1ynnky/github-actions-mcp-server` (Actions-specific). The GitHub MCP is the most widely adopted MCP server in the ecosystem.

**AGH use case -- Autonomous PR Lifecycle:**

1. Agent receives a task via session (e.g., from Slack or Linear ticket).
2. Agent clones repo, creates branch, implements changes.
3. Agent opens PR via GitHub MCP, filling description with context from AGH session memory.
4. Agent monitors CI status via GitHub Actions MCP, auto-fixes lint/test failures.
5. Agent responds to PR review comments, pushes fixes, and requests re-review.
6. AGH observe layer records the full lifecycle for future skill learning.

**Status:** MCP server exists. AGH extension wraps it with session context.

#### GitLab MCP

**What it does:** Interact with GitLab instances -- manage merge requests, read pipelines, query issues, browse repositories, check CI/CD pipeline status, read build logs.

**Existing MCP server:** Official GitLab MCP server (`gitlab-org/editor-extensions/gitlab-mcp-server`), plus community server by zereight. GitLab also acts as an MCP client via Duo, connecting to external MCP servers like Jira and Slack.

**AGH use case -- Pipeline Failure Triage:**

1. GitLab webhook triggers AGH session when a pipeline fails.
2. Agent queries GitLab MCP for pipeline logs and failed job details.
3. Agent reads the failing test/build output, cross-references with recent MR diffs.
4. Agent either auto-fixes the issue (pushes a commit) or creates an issue with root cause analysis.
5. Session memory retains the failure pattern for future similar incidents.

**Status:** MCP server exists. Build AGH extension for webhook-triggered sessions.

---

### 2. Project Management & Issue Tracking

#### Linear MCP

**What it does:** Query and manage Linear issues, projects, cycles, teams, and labels. Create/update issues, change status/priority/assignee, query sprint data, manage project workflows.

**Existing MCP server:** Community server (`tacticlaunch/mcp-linear`) with GraphQL API integration. Supports SSE transport.

**AGH use case -- Automated Sprint Ops:**

1. Agent monitors Linear for new issues assigned to it (or a team).
2. When a bug ticket arrives, agent reads the description, searches codebase, proposes a fix.
3. Agent creates a PR (via GitHub MCP), links it to the Linear issue, and moves the issue to "In Progress."
4. On PR merge, agent auto-transitions the Linear issue to "Done."
5. AGH memory stores the issue-to-fix mapping for future reference.

**Status:** MCP server exists. HIGH priority for AGH -- Linear is the default tracker for modern dev teams.

#### Jira + Confluence (Atlassian)

**What it does:** Jira: full issue CRUD, JQL queries, sprint management, custom field updates, workflow transitions. Confluence: space management, page CRUD, content search, label handling.

**Existing MCP servers:** Atlassian's official Remote MCP Server (beta, hosted on Cloudflare, first-party partnership with Anthropic). Also `sooperset/mcp-atlassian` open-source server covering both Jira and Confluence. Hainan Zhao's `mcp-gitlab-jira` for unified GitLab+Jira workflows.

**AGH use case -- Cross-Platform Knowledge Worker:**

1. Agent receives a Jira ticket about a production issue.
2. Agent searches Confluence for relevant runbooks and architecture docs.
3. Agent investigates the codebase, identifies the root cause.
4. Agent updates the Jira ticket with findings, creates a sub-task for the fix.
5. Agent implements the fix, opens a PR, and links everything back to the Jira epic.
6. AGH memory consolidates the runbook + fix into reusable knowledge.

**Status:** MCP servers exist (official + community). MEDIUM-HIGH priority -- critical for enterprise teams.

#### Notion MCP

**What it does:** Full Notion API access -- search pages/databases, create/update/delete pages and blocks, query databases with filters/sorts, manage workspace content. Optimized for AI agent token efficiency.

**Existing MCP server:** Official hosted server at `mcp.notion.com/sse` with OAuth, plus self-hosted option via `@notionhq/notion-mcp-server` (npm). Version 2.0.0 uses the 2025-09-03 API with data sources as primary abstraction.

**AGH use case -- Living Documentation Agent:**

1. Agent monitors code changes (via GitHub MCP) and automatically updates Notion docs.
2. When a new API endpoint is added, agent generates documentation and creates a Notion page.
3. Agent searches Notion for existing docs to avoid duplication, updates cross-references.
4. On architecture decisions, agent creates ADR pages in Notion with context from session memory.
5. AGH skills layer enables "document this change" as a reusable skill.

**Status:** MCP server exists (official, production). HIGH priority for teams using Notion as knowledge base.

---

### 3. CI/CD Platforms

#### GitHub Actions MCP

**What it does:** List, view, trigger, cancel, and rerun GitHub Actions workflows. Query workflow run status, read logs, analyze failures, manage secrets and variables.

**Existing MCP server:** Community server (`ko1ynnky/github-actions-mcp-server`) with complete workflow management. Compatible with Claude Desktop, Codeium, Windsurf.

**AGH use case -- CI Guardian Agent:**

1. Agent continuously monitors GitHub Actions for the team's repositories.
2. On build failure, agent reads logs, identifies the failing step, and diagnoses the issue.
3. Agent either pushes a fix (for known patterns stored in AGH memory) or creates an issue with diagnosis.
4. Agent detects flaky tests by analyzing failure patterns across runs.
5. Agent suggests workflow optimizations (caching, parallelism) based on observed run times.

**Status:** MCP server exists (community). HIGH priority -- most teams use GitHub Actions.

#### CircleCI MCP

**What it does:** Diagnose build failures, get structured error summaries, trace failures to commits, identify flaky tests, trigger rollbacks, and manage CI pipelines through natural language.

**Existing MCP server:** Official CircleCI MCP server (`CircleCI-Public/mcp-server-circleci`). Features include failure diagnosis, flaky test detection, and interactive rollback guidance.

**AGH use case -- Intelligent Build Doctor:**

1. CircleCI webhook notifies AGH of a failed build.
2. Agent queries CircleCI MCP for structured error logs and test results.
3. Agent correlates failure with recent commits (via GitHub MCP).
4. Agent identifies the root cause -- e.g., a dependency update broke a test.
5. Agent creates a fix PR or triggers a rollback via CircleCI MCP.
6. AGH observe layer tracks MTTR (mean time to resolution) metrics.

**Status:** MCP server exists (official). MEDIUM priority -- important for CircleCI shops.

#### Jenkins MCP

**What it does:** Jenkins MCP Server Plugin implements MCP server-side, enabling Jenkins to expose its build/deploy capabilities to LLM-powered applications and IDEs.

**Existing MCP server:** Official Jenkins MCP Server Plugin. Provides context, tools, and capabilities for CI/CD automation.

**AGH use case:** Similar to CircleCI -- build failure diagnosis, job triggering, log analysis. Especially valuable for enterprises with complex Jenkins pipelines.

**Status:** MCP server exists (official plugin). MEDIUM priority.

#### ArgoCD MCP (via K8s MCP Toolkit)

**What it does:** List clusters registered with ArgoCD, manage applications, sync deployments, handle resource management, query deployment status and history.

**Existing MCP server:** K8s MCP Server Toolkit provides unified kubectl, helm, istioctl, and argocd tools. Supports AWS EKS, Google GKE, and Azure AKS.

**AGH use case -- GitOps Deployment Agent:**

1. Agent monitors Git repository for merged PRs to the main branch.
2. Agent verifies ArgoCD sync status after deployment.
3. If sync fails, agent queries ArgoCD for error details and Kubernetes events.
4. Agent checks application health via kubectl, reads pod logs for crash loops.
5. Agent either fixes the manifest and re-syncs or triggers a rollback.
6. Agent posts deployment status to Slack (via Slack MCP).

**Status:** MCP server exists (K8s toolkit). HIGH priority for GitOps teams.

---

### 4. Deployment Platforms

#### Vercel / Railway / Fly.io / Netlify

**What it does:** Deploy applications, manage environments, view deployment logs, rollback deployments, manage environment variables, check health status.

**Existing MCP servers:**

- **Vercel:** Official MCP Handler adapter for Next.js/Nuxt/Svelte.
- **Railway:** Official MCP server (released Jan 2026) for deployment, service management, environment config.
- **Unified Deployment MCP:** Covers Vercel, Render, Railway, and Fly.io from a single endpoint with 9 tools.

**AGH use case -- Deploy & Monitor Agent:**

1. Agent receives "deploy to staging" command in session.
2. Agent triggers deployment via platform MCP (Railway/Vercel).
3. Agent monitors deployment progress and health checks.
4. Agent runs smoke tests against the deployed URL.
5. On failure, agent reads deployment logs, diagnoses the issue, and either fixes or rolls back.
6. Agent posts deployment summary to Slack with key metrics.

**Status:** Multiple MCP servers exist. MEDIUM priority -- wrap the Unified Deployment MCP for multi-platform support.

---

### 5. Code Quality & Security

#### SonarQube MCP

**What it does:** Query code quality issues, manage issue status (resolve, false-positive), search for bugs/vulnerabilities/code smells, track technical debt, access dependency risk analysis (Enterprise+).

**Existing MCP server:** Official by SonarSource (`SonarSource/sonarqube-mcp-server`). 423 stars, 321 commits. Requires JDK 21+, also available as Docker image. Integrates with Claude Code, Cursor, Windsurf. AI CodeFix generates LLM-powered fix suggestions.

**AGH use case -- Quality Gate Agent:**

1. After PR creation, agent triggers SonarQube analysis.
2. Agent queries SonarQube MCP for new issues introduced by the PR.
3. Agent auto-fixes simple issues (code smells, formatting) and pushes a commit.
4. For complex issues (security vulnerabilities), agent creates detailed comments on the PR.
5. Agent blocks merge if quality gate fails, posts summary to team channel.
6. AGH memory tracks recurring quality patterns per developer/module.

**Status:** MCP server exists (official). HIGH priority -- code quality gates are universal.

#### Snyk MCP

**What it does:** The most comprehensive security MCP -- 11 tools spanning SAST (code scanning), SCA (dependency scanning), IaC scanning, container image scanning, SBOM generation, and AI-BOM (AI Bill of Materials). Auth and trust management included.

**Existing MCP server:** Official Snyk MCP at v1.6.1 (March 2026). Also `snyk/agent-scan` (1.9k stars) for scanning MCP servers themselves for security issues.

**AGH use case -- Security Sentinel Agent:**

1. Agent runs nightly security scans via Snyk MCP across all project repositories.
2. Agent triages findings by severity -- critical/high vulns get immediate PRs with fixes.
3. Agent scans container images before deployment, blocking vulnerable images.
4. Agent generates SBOMs for compliance, stores in Notion (via Notion MCP).
5. For dependency vulnerabilities, agent creates PRs with version bumps and test results.
6. AGH observe layer tracks vulnerability trends and MTTR for security issues.

**Status:** MCP server exists (official, comprehensive). HIGH priority -- security scanning is critical.

#### Semgrep MCP

**What it does:** Static analysis for finding vulnerabilities, bugs, and enforcing coding standards. Semgrep Multimodal (March 2026) combines AI reasoning with deterministic rules for 8x more true positives.

**Existing MCP server:** The standalone MCP server was archived Oct 2025; MCP functionality is now built into the Semgrep binary itself. 639 stars (highest in category).

**AGH use case -- Code Review Copilot:**

1. Agent runs Semgrep on every PR before human review.
2. Agent applies custom rule sets based on project type (stored in AGH skills).
3. Agent provides inline comments on the PR with fix suggestions.
4. Agent learns from developer feedback (false positives) and adjusts rules.

**Status:** Built into Semgrep binary. HIGH priority for automated code review workflows.

#### Trivy MCP

**What it does:** Vulnerability scanning for local filesystems, container images, and remote repositories. Installs as a Trivy plugin.

**Existing MCP server:** Plugin-based (`trivy plugin install mcp`). Note: trivy-action was compromised twice in March 2026 -- pin to commit hashes.

**AGH use case:** Container image scanning before deployment, filesystem scanning for secrets/misconfigurations. Complements Snyk for open-source-only teams.

**Status:** MCP server exists (plugin). MEDIUM priority -- overlaps with Snyk.

#### Dependabot / Renovate (No MCP -- Build Opportunity)

**What it does:** Automated dependency updates. Renovate supports 90+ package managers across GitHub/GitLab/Bitbucket/Azure DevOps. Dependabot covers 30+ on GitHub only.

**Existing MCP server:** No MCP server exists for either tool. Notable gap identified in the ecosystem.

**AGH use case -- Dependency Guardian Agent:**

1. Agent monitors dependency versions across all repos.
2. Agent creates grouped PRs for dependency updates (similar to Renovate's grouping).
3. Agent runs security scans (via Snyk MCP) on proposed updates before creating PRs.
4. Agent auto-merges safe updates (patch versions with passing tests).
5. Agent flags breaking changes and creates migration guides.
6. AGH memory tracks dependency update success rates and common breakages.

**Status:** No MCP server exists. MEDIUM priority to build -- could differentiate AGH by combining dependency management with security scanning and automated testing.

---

### 6. Infrastructure as Code & Cloud

#### Terraform MCP (HashiCorp Official)

**What it does:** Terraform Registry integration, HCP Terraform and Terraform Enterprise workspace management, organization/project listing, private registry access. Dual transport (Stdio + StreamableHTTP).

**Existing MCP server:** Official HashiCorp server (`hashicorp/terraform-mcp-server`). Available as Docker image. AWS Labs also offers a Terraform MCP for AWS best practices and Checkov compliance.

**AGH use case -- Infrastructure Agent:**

1. Agent receives infrastructure request (e.g., "spin up a staging database").
2. Agent searches Terraform registry for appropriate modules.
3. Agent generates Terraform HCL, runs `terraform plan`, presents the plan for approval.
4. On approval, agent applies the plan and monitors resource creation.
5. Agent updates infrastructure documentation in Notion (via Notion MCP).
6. AGH skills layer stores reusable infrastructure patterns.

**Status:** MCP server exists (official). HIGH priority -- IaC is fundamental to DevOps.

#### Pulumi MCP

**What it does:** Infrastructure-as-code operations using real programming languages (TypeScript, Python, Go). Integrates with Pulumi Cloud for state management.

**Existing MCP server:** Official Pulumi MCP server. Integrates with Cursor, Claude Code, Windsurf.

**AGH use case:** Similar to Terraform but for teams preferring code-based IaC. Agent writes Pulumi programs in the team's preferred language.

**Status:** MCP server exists (official). MEDIUM priority -- smaller user base than Terraform.

#### Kubernetes MCP

**What it does:** kubectl operations, pod management, log access, resource monitoring, debugging, helm chart management. Multiple servers available for different needs.

**Existing MCP servers:**

- **kubectl-mcp-server:** Direct kubectl access -- diagnose pod crashes, read logs, check events.
- **k8m (multi-cluster):** 50+ tools for managing dev/staging/prod across 10+ clusters.
- **Lens MCP Server:** Native EKS/AKS integration, visual cluster management.
- **K8s MCP Toolkit:** Unified kubectl + helm + istioctl + argocd.

**AGH use case -- Kubernetes SRE Agent:**

1. Agent receives alert from PagerDuty/Grafana about pod crash loop.
2. Agent queries K8s MCP for pod events, logs, resource limits, and node health.
3. Agent identifies root cause (e.g., OOM kill, failed health check, bad config).
4. Agent applies fix (scale up, update resource limits, rollback deployment).
5. Agent creates post-incident report and updates runbook in Notion.
6. AGH memory stores incident patterns for faster future diagnosis.

**Status:** Multiple MCP servers exist. HIGH priority -- K8s is the deployment standard.

#### Cloud Providers (AWS / GCP / Azure)

**What they do:**

- **AWS:** 60+ official MCP servers covering Lambda, ECS, EKS, S3, EC2, RDS, CloudWatch, and more. FinOps queries like "show EC2 instances over $500/month unused for 30 days."
- **GCP:** Official servers (preview) for BigQuery, AlloyDB, Spanner, Cloud SQL, Firestore, Bigtable, Maps. Identity-first security via IAM.
- **Azure:** Official servers for Azure Storage, Cosmos DB, Azure CLI, Azure DevOps (work items, PRs, builds, pipelines).

**Multi-cloud:** Cloud Pilot MCP supports AWS/Azure/GCP/Alibaba Cloud with 51,900+ operations and OpenTofu integration.

**AGH use case -- Cloud Operations Agent:**

1. Agent monitors cloud costs and resource utilization.
2. Agent identifies unused resources and proposes cleanup (with cost savings estimates).
3. Agent provisions new resources based on Terraform/Pulumi plans.
4. Agent responds to CloudWatch/Stackdriver alerts with automated investigation.
5. AGH memory tracks cloud spend patterns and optimization history.

**Status:** Official MCP servers exist for all three. HIGH priority -- choose based on team's cloud provider.

---

### 7. Monitoring & Observability

#### Sentry MCP

**What it does:** Error tracking, performance monitoring, release tracking. Query issues, view stack traces, manage issue status, analyze error frequency and impact.

**Existing MCP server:** Official Sentry MCP server. Sentry also offers MCP server monitoring (beta, Aug 2025) built on OpenTelemetry for observing MCP servers themselves. Their own MCP server handles 50M requests/month.

**AGH use case -- Error Response Agent:**

1. Sentry webhook triggers AGH session on new critical error.
2. Agent queries Sentry MCP for full stack trace, affected users, error frequency.
3. Agent searches codebase for the relevant code, identifies the bug.
4. Agent creates a fix PR with tests covering the error case.
5. Agent links the PR to the Sentry issue and assigns for review.
6. After deployment, agent verifies the error rate drops to zero.
7. AGH observe layer tracks error-to-fix pipeline metrics.

**Status:** MCP server exists (official). HIGH priority -- this is the canonical "agent receives alert, fixes code, opens PR" workflow.

#### Datadog MCP

**What it does:** Live observability data access -- logs, metrics, traces, APM data, monitors, SLOs, service definitions. Also offers MCP client monitoring via LLM Observability for tracking agent behavior.

**Existing MCP server:** Official Datadog MCP Server (GA March 2026). Covers LLM Observability, product analytics, Cloud Network Monitoring, security, software delivery, synthetics, and workflow automation. Also community server (`shelfio/datadog-mcp`).

**AGH use case -- Observability Investigation Agent:**

1. Datadog alert triggers AGH session (e.g., p99 latency spike).
2. Agent queries Datadog MCP for relevant traces, logs, and metrics.
3. Agent correlates the spike with recent deployments (via GitHub/ArgoCD MCP).
4. Agent identifies the problematic commit/service and creates a revert PR if needed.
5. Agent updates the Datadog monitor with refined thresholds based on investigation.
6. AGH memory stores the investigation playbook for similar future incidents.

**Status:** MCP server exists (official, GA). HIGH priority -- Datadog is the enterprise observability standard.

#### Grafana MCP

**What it does:** Query dashboard data, inspect data sources, retrieve incident details. Optimized for minimal context window usage and reduced token costs.

**Existing MCP server:** Official Grafana MCP Server. AWS DevOps Agent includes a built-in Grafana MCP server supporting self-managed, Grafana Cloud, and Amazon Managed Grafana.

**AGH use case -- Dashboard-Driven Diagnosis:**

1. Agent queries Grafana dashboards for anomalies across services.
2. Agent correlates metrics with deployment timelines.
3. Agent generates incident summaries with relevant graphs/data for team review.
4. For Grafana Cloud IRM users, agent manages incident lifecycle end-to-end.

**Status:** MCP server exists (official). HIGH priority.

#### PagerDuty MCP

**What it does:** Incident management (view/acknowledge/resolve/reassign), on-call schedule queries, incident analytics, alert correlation and noise reduction.

**Existing MCP server:** Community MCP server. AWS DevOps Agent has native PagerDuty integration.

**AGH use case -- On-Call Copilot:**

1. PagerDuty alert triggers AGH session for on-call engineer.
2. Agent acknowledges the alert, gathers context from Datadog/Grafana/Sentry.
3. Agent runs diagnostic commands via K8s MCP, reads application logs.
4. Agent presents a diagnosis summary with recommended actions.
5. Agent executes approved remediation (restart pod, rollback, scale up).
6. Agent resolves the PagerDuty incident and creates a post-mortem draft.

**Status:** Community MCP server exists. HIGH priority -- on-call automation is a top developer pain point.

---

### 8. Communication & ChatOps

#### Slack MCP

**What it does:** Search channels/messages/files, send messages, read message history, read threads, create rich canvases. Workspace admin approval controls.

**Existing MCP server:** Official Slack MCP server hosted at `mcp.slack.com` with OAuth. GA early 2026. Note: rate limits changed May 2025 for non-Marketplace apps.

**AGH use case -- Team Communication Hub:**

1. Developers interact with AGH agents via Slack messages.
2. Agent posts deployment notifications, build status, and PR summaries to relevant channels.
3. Agent monitors channels for questions about code/architecture, answers using codebase knowledge.
4. Agent creates threaded incident response channels during outages.
5. Agent summarizes daily activity across channels for team leads.

**Status:** MCP server exists (official). HIGH priority -- Slack is the primary developer communication tool.

---

### 9. Documentation Platforms

#### Mintlify / ReadMe

**What they do:** Mintlify auto-generates MCP servers from documentation sites, providing search and API querying tools. ReadMe offers similar MCP endpoints with interactive API references. Both support llms.txt and AI-optimized content formats.

**Existing MCP servers:** Both auto-generate MCP endpoints. Mintlify hosts at `/mcp` path of any docs site. ~48% of documentation traffic is already from AI agents.

**AGH use case -- Documentation-Aware Agent:**

1. Agent queries Mintlify/ReadMe MCP to understand third-party API docs before writing integration code.
2. Agent uses documentation context to generate accurate API client code.
3. Agent validates generated code against API docs automatically.

**Status:** MCP servers auto-generated. LOW priority for AGH-specific extension -- agents can use these directly.

---

## High-Priority Extension Bundles for AGH

Based on this research, the following extension bundles represent the highest-value integrations:

### Bundle 1: Development Lifecycle (Ship Code)

- **GitHub/GitLab MCP** -- version control, PRs, code review
- **Linear/Jira MCP** -- issue tracking, sprint management
- **GitHub Actions/CircleCI MCP** -- CI pipeline management
- **SonarQube + Snyk MCP** -- quality gates + security scanning

**Value prop:** Agent takes a ticket from "To Do" to "Merged PR" autonomously, with quality and security checks at every step.

### Bundle 2: Incident Response (Fix Production)

- **Sentry/Datadog MCP** -- error/performance alerting
- **PagerDuty MCP** -- incident management
- **Grafana MCP** -- metrics/dashboard access
- **Kubernetes MCP** -- infrastructure diagnosis and remediation
- **Slack MCP** -- team communication during incidents

**Value prop:** Agent receives production alert, investigates across observability stack, diagnoses root cause, applies fix or rollback, resolves incident, posts summary -- reducing MTTR from hours to minutes.

### Bundle 3: Infrastructure Operations (Manage Infra)

- **Terraform/Pulumi MCP** -- infrastructure as code
- **AWS/GCP/Azure MCP** -- cloud resource management
- **Kubernetes + ArgoCD MCP** -- deployment and orchestration
- **Docker MCP** -- container management

**Value prop:** Agent manages infrastructure lifecycle from provisioning to deployment to scaling, with cost optimization and compliance checks.

### Bundle 4: Knowledge & Documentation (Stay Informed)

- **Notion/Confluence MCP** -- knowledge base management
- **Slack MCP** -- team communication context
- **GitHub MCP** -- code-level documentation
- **Mintlify/ReadMe MCP** -- external API documentation

**Value prop:** Agent maintains living documentation, auto-updates docs on code changes, answers team questions with codebase + docs context.

---

## Build-From-Scratch Opportunities

These integrations have no existing MCP server and represent differentiation opportunities for AGH:

| Integration                | What to Build                                                   | Why                                                          |
| -------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------ |
| **Dependabot/Renovate**    | Dependency update management with security scanning             | No MCP server exists; combining updates + security is unique |
| **GitHub Security Alerts** | Query Dependabot alerts, secret scanning, code scanning results | Notable gap -- no MCP for GitHub's security features         |
| **Shortcut**               | Issue tracking for teams using Shortcut                         | No MCP server found; growing user base                       |
| **Incident.io**            | Modern incident management (replacing Opsgenie)                 | Growing platform, no MCP server found                        |
| **LaunchDarkly**           | Feature flag management tied to deployments                     | Agent-controlled progressive rollouts                        |

---

## Security Considerations for AGH Extensions

Key security practices gleaned from the research:

1. **Least privilege:** Extensions should request minimal permissions. Read-only by default, write access only when explicitly needed.
2. **Stdio transport for sensitive operations:** `terraform apply`, database migrations, and production deployments should use stdio (no network exposure).
3. **Credential isolation:** Never pass cloud credentials to remote MCP servers. Run infrastructure extensions locally.
4. **Pin dependencies:** Always pin GitHub Actions and MCP server versions to commit SHAs (lesson from Trivy compromise, March 2026).
5. **Audit logging:** AGH's observe layer should log all extension tool invocations for compliance.
6. **Human-in-the-loop:** Destructive operations (delete, apply, deploy) should require explicit approval via AGH session.

---

## Sources

- [GitLab MCP Server Docs](https://docs.gitlab.com/user/gitlab_duo/model_context_protocol/mcp_server/)
- [mcp-atlassian (Jira + Confluence)](https://github.com/sooperset/mcp-atlassian)
- [Top 15 MCP Servers Every Developer Should Install in 2026](https://dev.to/jangwook_kim_e31e7291ad98/top-15-mcp-servers-every-developer-should-install-in-2026-n1h)
- [Best MCP Servers for Developers 2026 (Builder.io)](https://www.builder.io/blog/best-mcp-servers-2026)
- [CircleCI MCP Server](https://circleci.com/product/mcp/)
- [CircleCI MCP Server GitHub](https://github.com/CircleCI-Public/mcp-server-circleci)
- [GitHub Actions MCP Server](https://github.com/ko1ynnky/github-actions-mcp-server)
- [HashiCorp Terraform MCP Server](https://github.com/hashicorp/terraform-mcp-server)
- [AWS Terraform MCP Server](https://awslabs.github.io/mcp/servers/terraform-mcp-server)
- [Pulumi MCP Server](https://www.pulumi.com/docs/iac/guides/ai-integration/mcp-server/)
- [18 Best DevOps MCP Servers 2026 (Lens)](https://lenshq.io/blog/best-devops-mcp-servers)
- [SonarQube MCP Server](https://github.com/SonarSource/sonarqube-mcp-server)
- [SonarQube MCP Docs](https://docs.sonarsource.com/sonarqube-mcp-server)
- [Snyk MCP and Code Security Servers](https://chatforest.com/reviews/code-security-mcp-servers/)
- [Datadog MCP Server (Official)](https://www.datadoghq.com/product/ai/mcp-server/)
- [Datadog MCP Server Launch](https://www.datadoghq.com/about/latest-news/press-releases/datadog-launches-mcp-server/)
- [Sentry MCP Server Monitoring](https://blog.sentry.io/introducing-mcp-server-monitoring/)
- [Grafana MCP Server for Tracing](https://grafana.com/docs/grafana-cloud/send-data/traces/mcp-server/)
- [10 Best MCP Servers for Platform Engineers 2026](https://stackgen.com/blog/the-10-best-mcp-servers-for-platform-engineers-in-2026)
- [Notion MCP Server](https://developers.notion.com/docs/mcp)
- [Notion MCP Server GitHub](https://github.com/makenotion/notion-mcp-server)
- [Notion Hosted MCP Server Blog](https://www.notion.com/blog/notions-hosted-mcp-server-an-inside-look)
- [Atlassian Remote MCP Server](https://www.atlassian.com/blog/announcements/remote-mcp-server)
- [Slack MCP Server Docs](https://docs.slack.dev/ai/slack-mcp-server/)
- [Discord MCP Servers (Community)](https://chatforest.com/reviews/discord-mcp-servers/)
- [Mintlify MCP Server](https://www.mintlify.com/docs/ai/model-context-protocol)
- [ReadMe vs Mintlify 2026](https://readme.com/blog/readme-vs-mintlify)
- [Railway MCP Server](https://www.pulsemcp.com/servers/railway)
- [MCP Servers DevOps Complete Guide 2026 (Cloudship)](https://www.cloudshipai.com/blog/mcp-servers-devops-complete-guide-2026)
- [GCP MCP Servers Supported Products](https://docs.cloud.google.com/mcp/supported-products)
- [Five MCP Servers to Rule the Cloud (InfoWorld)](https://www.infoworld.com/article/4129024/five-mcp-servers-to-rule-the-cloud.html)
- [Dependabot vs Renovate 2026](https://appsecsanta.com/sca-tools/dependabot-vs-renovate)
- [Claude Code vs Cursor Comparison](https://uibakery.io/blog/claude-code-vs-cursor)
- [Aider vs OpenCode vs Claude Code vs Goose](https://sanj.dev/post/comparing-ai-cli-coding-assistants)
