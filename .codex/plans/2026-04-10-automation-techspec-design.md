# Design: Automation TechSpec Hardening

**Date:** 2026-04-10
**Status:** Approved
**Authors:** Pedro Nauck + Codex

## Summary

This design hardens the automation techspec before task decomposition. The goal is not to expand the product surface; it is to remove the ambiguous boundaries that would otherwise force workaround-shaped implementation.

The approved direction is a "v1 hardened" automation design:

- automation supports both `global` and `workspace` scope
- schedules and triggers remain separate persisted models
- a shared dispatcher remains the single execution path
- TOML keeps ownership of config-defined automation definitions
- runtime overlays are allowed only for `enabled/disabled`
- internal trigger ingress reuses the existing `observer/hooks` boundary
- external webhook triggers are first-class in v1 and require HMAC authentication
- concurrency and fire-limit semantics are made explicit and testable

## Problem

The current automation techspec is directionally strong, but several critical boundaries were underspecified or contradictory:

- scope was implicitly workspace-bound in some places and globally named in others
- webhook security was treated as a future enhancement instead of a v1 invariant
- trigger ingestion assumed new subscription seams that do not exist in the current runtime
- TOML source-of-truth and runtime mutability were not reconciled
- the global concurrency story was not actually global across all activation paths
- fire limits were framed as safety controls while remaining ephemeral in memory

Those issues are not implementation details. They are design decisions that must be settled before the work is decomposed into tasks.

## Approved Direction

### 1. Domain model and ownership

Automation supports two explicit scopes:

- `global`
- `workspace`

Jobs and triggers remain separate persisted entities. The system does not force a single generic persisted model in v1 because the architectural value comes from shared dispatch and shared execution governance, not from collapsing fundamentally different activation types into one schema.

Each automation definition has:

- stable identity
- explicit scope
- optional `workspace_id` when scope is `workspace`
- source ownership (`config` or `dynamic`)

Name uniqueness is scoped, not global:

- global names must be unique among global definitions
- workspace names must be unique only within the same workspace

### 2. TOML ownership and runtime overlays

TOML remains the owner of config-defined automation definitions.

For TOML-backed jobs and triggers:

- the definition in TOML is authoritative
- the database stores the materialized definition for runtime use
- a separate persistent overlay stores only operational runtime state

Approved overlay scope:

- `enabled/disabled` only

The following fields remain owned by TOML and are not runtime-mutable:

- identity
- scope
- workspace binding
- prompt
- schedule
- event
- retry policy
- fire-limit policy

Dynamic jobs and triggers continue to be fully defined and mutated through API/CLI storage paths.

### 3. Execution ownership and concurrency

The shared automation dispatcher owns execution governance.

That means every activation path must flow through the same concurrency gate:

- scheduler fires
- internal triggers
- webhook triggers
- manual API/CLI trigger requests

The approved model is:

- scheduler-level singleton controls remain local protections against overlap for a specific scheduled job
- the automation manager/dispatcher owns the global concurrency semaphore
- no activation path may bypass that gate

This prevents scheduler-only enforcement and keeps the system’s cost and load behavior coherent.

## Approved Event Ingress Model

### 1. Internal events

The canonical trigger ingress boundary in v1 is the existing daemon `observer/hooks` boundary.

Automation is a consumer of normalized runtime events; it does not introduce new direct subscription APIs in `session` or `memory/consolidation`.

Built-in internal sources are normalized from that boundary, including:

- `session.created`
- `session.stopped`
- `memory.consolidated`
- hook-completion derived automation events

### 2. Activation envelopes

Every source, internal or external, is normalized into a common activation envelope before matching:

```json
{
  "kind": "session.stopped",
  "scope": "workspace",
  "workspace_id": "ws_123",
  "source": "observer",
  "data": {
    "session_id": "sess_1",
    "agent_name": "researcher",
    "stop_reason": "completed"
  }
}
```

The trigger engine matches `Trigger` definitions against these normalized envelopes. The dispatcher only executes already-matched activations.

This keeps event sources decoupled from dispatch logic and avoids ad hoc `map[string]any` plumbing throughout the system.

## Approved Webhook Design

### 1. External webhook support is in scope for v1

Webhook triggers are real external integration points in v1, not localhost-only placeholders.

### 2. Webhook endpoint addressing

Opaque ID-only endpoints were rejected as too hard to operate manually.

Pure human-readable slugs were rejected because rename and collision semantics become fragile.

Approved endpoint shape:

- readable slug
- stable short ID suffix

Examples:

```text
POST /api/webhooks/global/github-main-push--wbh_01ABCXYZ
POST /api/webhooks/workspaces/ws_123/post-research--wbh_01DEF456
```

Approved trigger fields:

- `name`: human label, editable
- `endpoint_slug`: human-operable URL slug
- `webhook_id`: stable technical identifier used for canonical resolution
- `webhook_secret`: HMAC secret

The matching implementation resolves the endpoint by stable `webhook_id`, not by the slug text.

### 3. Webhook authentication

Webhook authentication is mandatory in v1.

Approved mechanism:

- HMAC per trigger
- timestamp included in request headers
- signature computed over the request body and timestamp material

The daemon must:

1. resolve the trigger from the stable endpoint identifier
2. validate timestamp freshness
3. validate HMAC signature
4. reject before dispatch on any auth failure
5. only after successful validation normalize the payload into an activation envelope

Fire limits remain supplemental protection and must never be treated as a replacement for authentication.

## Approved Operational Guarantees

### 1. Fire limits

Fire limits are safety controls, not best-effort hints.

Therefore they must not reset merely because the daemon restarted.

Approved direction:

- fire-limit evaluation uses persisted run history from SQLite for the active window
- in-memory tracking may exist as a cache, but it is not the source of truth

This closes the restart loophole and makes the limit meaningful for expensive LLM-backed automation.

### 2. Filter semantics

Trigger filters operate on explicit activation-envelope fields, not on unbounded arbitrary payload maps.

Approved semantics:

- exact-match comparisons only in v1
- no regex filtering in v1
- field paths and allowed keys must be defined by event envelope contracts

### 3. Template semantics

Prompt templates remain based on Go `text/template`, but with strict behavior:

- template parsing/validation occurs on create/update
- execution uses strict missing-key behavior
- invalid field references fail early

This ensures broken templates are surfaced at definition time instead of producing latent runtime failures.

## Data Model Consequences

The approved design implies the techspec should adopt the following structural changes:

- explicit `scope` field on jobs and triggers
- nullable `workspace_id` for global-scope entries
- uniqueness rules based on scope
- separate runtime overlay representation for TOML-backed operational state
- webhook identity fields distinct from human display naming
- trigger matching based on normalized activation envelopes

## Required Testing Invariants

The updated techspec should require tests for these scenarios:

1. Global and workspace-scoped automations with the same human name coexist correctly.
2. TOML-backed jobs/triggers allow only `enabled/disabled` overlay mutation and reject definition edits.
3. Dispatcher-enforced concurrency caps apply uniformly to schedule, trigger, webhook, and manual executions.
4. Fire-limit rejection still works after daemon restart within the same active window.
5. Webhook requests with invalid timestamp or HMAC fail before any dispatch occurs.
6. Internal trigger activation for session lifecycle and memory consolidation flows through the observer/hooks boundary rather than direct new subscription APIs.
7. Invalid prompt template field references fail during definition validation rather than during live dispatch.

## Non-Goals

This design does not introduce:

- a new internal event bus for the daemon
- a single generic persisted automation entity replacing jobs and triggers
- broad runtime mutation of TOML-defined automations
- unsigned or best-effort-authenticated external webhooks
- regex/filter-language complexity in v1

## Follow-On Work

Once the techspec is patched to match this design, task decomposition should follow this order:

1. Fix spec and ADR inconsistencies
2. Define scope-aware persistence and overlay rules
3. Define activation envelope contracts and trigger ingress plumbing
4. Define dispatcher-owned concurrency and fire-limit behavior
5. Define webhook transport/auth semantics
6. Decompose implementation tasks across storage, runtime, API, CLI, and web
