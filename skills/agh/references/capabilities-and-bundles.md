# Capabilities And Bundles

## Contents

- Capability vocabulary
- Extensibility surfaces
- Agent manageability
- Bundles
- Hooks
- Config lifecycle

## Capability Vocabulary

The canonical AGH artifact name is capability. Do not use recipe, workflow, procedure, or playbook for current AGH behavior unless quoting historical material.

A capability should be discoverable, manageable by agents, and represented through public runtime surfaces. It is incomplete if it only works through internal Go calls or the web UI.

## Extensibility Surfaces

When adding or changing AGH behavior, decide which surfaces are affected:

- extensions and extension resources
- hooks
- skills and capabilities
- tools and toolsets
- bundles
- registries
- bridge SDKs
- CLI, HTTP, and UDS APIs
- docs and generated references

No-impact is acceptable only when there is evidence.

## Agent Manageability

Every user-visible runtime capability needs an agent-operable path:

- CLI with -o json or -o jsonl where relevant
- HTTP/UDS parity when state crosses the daemon boundary
- discoverable status/config output
- deterministic errors and reason codes
- docs that describe the agent path

UI-only management is incomplete.

## Bundles

Bundles activate related runtime resources together. Treat bundle projection as daemon-owned state. Do not make a bundle depend on prompt prose for authority.

When changing bundle behavior, update resources, registries, config docs, CLI/API surfaces, and tests in the same change. Greenfield AGH favors hard cuts over compatibility bridges.

## Hooks

Hooks are typed dispatch at the owning state transition. They are not a generic event bus and must not tail event/log tables to infer work.

Hooks may deny, narrow, annotate, or observe. They must not bypass safety primitives such as claim tokens, leases, TTL, lineage, spawn caps, or permission narrowing.

Skill-declared hooks are part of the skill contract. Keep hook declarations structured and validated, not buried in prose.

## Config Lifecycle

Any feature or refactor must state whether config.toml keys, defaults, docs, and examples are added, changed, or removed. In greenfield alpha, delete obsolete config paths instead of creating aliases or fallback bridges.

If a rename touches code, storage, APIs, CLI, extensions, specs, docs, and task artifacts, update them together.
