# Tools And Skills

## Contents

- Tool-first operating model
- Discovery loop
- Skill loading
- Bundled skill resources
- Skill provenance and shadows
- Native AGH tool map
- Management-surface exceptions
- Skill authoring rules
- Reference-system lessons

## Tool-First Operating Model

AGH exposes runtime capabilities through a policy-filtered tool registry. Prefer native AGH tools over equivalent agh shell commands when a dedicated tool is callable. Tool calls are structured, policy-aware, observable, and easier to redact and audit.

Use shell commands for repository work, explicit operator requests, and management flows AGH keeps outside the normal tool-call loop.

## Discovery Loop

Use this sequence for AGH-native work:

1. Resolve canonical `agh__tool_search`, then search using the runtime domain or action.
2. Resolve canonical `agh__tool_info`, then inspect the selected ToolID before first invocation.
3. Invoke the returned dedicated tool reference with the descriptor's input schema.
4. Diagnose denied or missing tools from reason codes before changing surface.

`agh__*` names are canonical IDs, not harness call names. Use them for registry, policy, CLI, descriptors, and `tool_id`; call only the reference the harness returns.

For skills, resolve canonical `agh__skill_search`/`agh__skill_view`, then call returned references. Use CLI fallback only when denied, absent, or explicitly requested.

## Skill Loading

The prompt catalog lists skill names and descriptions, not full bodies. Load the full body on demand:

    agh skill view agh

Inside a tool-capable session, resolve the equivalent skill search/view tools through the active harness.
For resource files inside daemon-managed AGH sessions, use the returned skill view reference with the resource path. The CLI resource form is for local operator mode where skill resolution reads directly from the filesystem:

    agh skill view agh --file references/network.md

When a session receives repeated prompts with the same resolved skill catalog, AGH may replace the full `<current-available-skills>` block with a compact unchanged marker. Treat the previous full block in that session as current until AGH sends a later full catalog block.

AGH may also compact repeated `<agh-situation-context>` JSON sections with `"unchanged":true` markers. Reuse the previous full section for the same ACP session and workspace; call live AGH tools or context endpoints when you need an exact current value instead of prompt context.

## Bundled Skill Resources

Bundled AGH skills are compiled from the repository skills/<name>/ directories. The canonical AGH bundled skill is agh. It includes SKILL.md and flat references/\*.md resource files.

Resource files are load-bearing. A summary in SKILL.md is never a substitute for reading the referenced file selected by the router.

## Skill Provenance And Shadows

Every skill list/detail payload includes resolver provenance. `provenance.precedence_tier` names the winning tier, and installed-from metadata identifies bundle or extension ownership when present.

When multiple declarations use the same skill name, AGH keeps the normal precedence order and records losing declarations as shadows. Use these surfaces before assuming which skill body is active:

    agh skill where <name> --workspace <ref> --for-agent <agent>
    GET /api/skills/{name}/shadows?workspace=<ref>&for_agent=<agent>

The response shape is `SkillShadowsRecord` / `SkillShadowsResponse`: `winner` is the effective declaration, and each entry in `shadows` carries `path`, `tier`, `resolved_to_winner`, and `detected_at`. The winning entry is marked `resolved_to_winner: true`; lower-precedence declarations remain visible with `false`.

Do not diagnose skill drift from filesystem paths alone. Use the resolver view so workspace, agent-local, bundled, marketplace, extension, and additional-path precedence are all considered.

Marketplace install can write files and still fail discovery verification when the effective skill is
disabled, shadowed by a higher-precedence declaration, missing marketplace provenance, or reporting a
different slug. Treat a marketplace unavailable or not-discoverable install result as terminal until
local state changes. Use `agh skill where <name>`, inspect the winning source and path, then enable
the installed skill, remove or rename the shadowing declaration, or remove the broken install
directory before retrying.

## Native AGH Tool Map

Inside AGH, read references/native-tools.md before choosing a tool or CLI fallback. It lists daemon-native toolsets and stable `agh__*` IDs, but parameters and availability come from the live descriptor returned by canonical `agh__tool_info`.

## Management-Surface Exceptions

Keep these on operator CLI, HTTP, or UDS surfaces unless AGH explicitly exposes a scoped tool:

- daemon lifecycle, sockets, host/port, sandbox, provider bootstrap, and destructive repair
- creating, stopping, or mutating arbitrary sessions outside scoped authority
- MCP OAuth login/logout and browser-based auth
- trust roots, raw secrets, OAuth credentials, provider API-key bindings, PKCE material, and MCP auth secrets
- cross-session terminal-state mutation

Read-only inspection tools may exist for these domains. Do not invent a mutating tool call.

## Skill Authoring Rules

AGH skills follow progressive disclosure:

- Keep SKILL.md short and under the practical 500-line ceiling.
- Put heavy contracts in flat one-level references/\*.md files.
- Put the Required Reading Router near the top.
- Use hard STOP directives before steps that require reference content.
- Do not nest reference-to-reference dependencies.
- Add ## Contents to every reference file that might be partially read.

For this agh skill, do not add scripts. It is a documentation and routing bundle.

## Reference-System Lessons

Hermes distinguishes skills from tools: use skills for procedural guidance and shell workflows; use tools for authenticated, precise, binary, streaming, or realtime work. OpenClaw keeps skill precedence separate from tool allowlists and injects compact prompt catalogs with local paths. Claude Code loads directory-format skill-name/SKILL.md, tracks skill roots for resources, and supports hooks from skill metadata.

AGH follows the same lesson: one compact catalog entry, explicit resource loading, daemon-owned authority, and structured tool surfaces for state changes.
