# Tools And Skills

## Contents

- Tool-first operating model
- Discovery loop
- Skill loading
- Bundled skill resources
- Native AGH tool map
- Management-surface exceptions
- Skill authoring rules
- Reference-system lessons

## Tool-First Operating Model

AGH exposes runtime capabilities through a policy-filtered tool registry. Prefer native AGH tools over equivalent agh shell commands when a dedicated tool exists and is callable. Tool calls are structured, policy-aware, observable, and easier for the daemon to redact and audit.

Use shell commands for ordinary repository work, explicit operator requests, and management flows that AGH intentionally keeps outside the normal tool-call loop.

## Discovery Loop

Use this sequence for AGH-native work:

1. Search with agh\_\_tool_search using the runtime domain or action.
2. Inspect with agh\_\_tool_info before the first invocation.
3. Invoke the dedicated tool with the descriptor's input schema.
4. Diagnose denied or missing tools from reason codes before changing surface.

For skills, search with agh**skill_search and load full instructions with agh**skill_view. Use the operator CLI fallback only when the tool path is denied, absent, or the user asks for CLI output.

## Skill Loading

The prompt catalog lists skill names and descriptions, not full bodies. Load the full body on demand:

    agh skill view agh

Inside a tool-capable session, use the equivalent skill search/view tools.
For resource files inside daemon-managed AGH sessions, use the native skill view tool with the resource path instead of the CLI `--file` fallback. The CLI resource form is for local operator mode where skill resolution reads directly from the filesystem:

    agh skill view agh --file references/network.md

## Bundled Skill Resources

Bundled AGH skills are compiled from the repository skills/<name>/ directories. The canonical AGH bundled skill is agh. It includes SKILL.md and flat references/\*.md resource files.

Resource files are load-bearing. A summary in SKILL.md is never a substitute for reading the referenced file selected by the router.

## Native AGH Tool Map

Agents running inside AGH should read references/native-tools.md before choosing a tool or CLI fallback. That file lists the daemon-native toolsets and stable agh** IDs, but the source of truth for parameters and availability is always the live descriptor returned by agh**tool_info.

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
