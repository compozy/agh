# Analysis: ACP Tool Registry Compatibility

## Scope

This analysis answers whether ACP imposes a tool registry pattern that AGH must follow, and which `.resources/*` projects materially use ACP in ways that affect the Tool Registry TechSpec. The research combines official ACP/MCP documentation with read-only subagent passes over `.resources/rayclaw`, `.resources/harnss`, `.resources/acpx`, `.resources/openclaw`, `.resources/opencode`, and an inventory pass across every top-level `.resources/*` project.

## Executive Conclusion

ACP does not define a durable, programmatic tool registry for callable tools. ACP defines session lifecycle, prompt streaming, client authority callbacks, permission requests, MCP server bootstrap fields, and observable tool-call events. Those tool-call events carry `toolCallId`, `title`, `kind`, `status`, locations, raw input, raw output, and content, but they do not carry a stable `name` field equivalent to MCP `Tool.name`.

Therefore, AGH should not model its Tool Registry as an ACP registry, and should not use ACP `title` as a durable policy or collision key.

For session-visible AGH tools, the strongest compatibility path remains the accepted MVP path: an AGH-hosted MCP server backed by the daemon Tool Registry. MCP supplies the externally callable `Tool.name`; ACP supplies the way ACP-compatible agents receive `mcpServers`, report tool execution, and request permission.

The practical design correction is:

- AGH should use one canonical provider-safe `ToolID` everywhere, using reserved double-underscore namespace separators, for example `agh__skill_view`, `mcp__github__create_issue`, or `ext__linear__search`.
- The same `ToolID` should be the hosted MCP `Tool.name`; AGH should not introduce a second wire alias in the MVP.
- ACP `title` is display-only and event-only.
- ACP `ToolKind` is a risk/display hint, not registry identity.
- `permissions.mode` remains the session approval ceiling; registry policy remains the granular layer below it.

## Official Protocol Constraints

### ACP

Official ACP schema evidence:

- ACP `session/new`, `session/load`, and `session/resume` include `mcpServers`; agents are expected to connect to those MCP servers for the session. Source: <https://agentclientprotocol.com/protocol/schema>.
- ACP `ToolCall` is event/reporting data with `toolCallId`, `title`, `kind`, `status`, `rawInput`, `rawOutput`, `locations`, and `content`. The schema describes `title` as human-readable and `toolCallId` as unique within a session. It does not expose a durable callable `name` field. Source: <https://agentclientprotocol.com/protocol/schema>.
- ACP `ToolKind` values are coarse categories such as `read`, `edit`, `delete`, `move`, `search`, `execute`, `think`, `fetch`, `switch_mode`, and `other`. The schema says these help clients pick icons and display progress, which is weaker than registry identity. Source: <https://agentclientprotocol.com/protocol/schema>.
- ACP `session/request_permission` carries a `toolCall` object plus permission options. It is a permission bridge for a concrete tool call, not a registry discovery API. Source: <https://agentclientprotocol.com/protocol/schema>.
- The official "ACP Registry" is an agent registry: a catalog of ACP-compatible agents and their install/run metadata, not a callable tool registry. Source: <https://agentclientprotocol.com/registry>.

### MCP

Official MCP schema evidence:

- MCP `tools/list` returns `Tool[]`.
- MCP `Tool` has `name`, optional `title`, optional `description`, `inputSchema`, optional `outputSchema`, annotations, execution metadata, and `_meta`.
- MCP describes `name` as intended for programmatic/logical use and `title` as intended for UI/end-user contexts. Source: <https://modelcontextprotocol.io/specification/draft/schema>.

Implication: AGH should treat MCP `Tool.name` as the session wire name when exposing AGH registry tools through hosted MCP. ACP does not replace that name.

## ACP Usage Inventory Across `.resources/*`

| Project | ACP usage | Tool registry relevance |
|---|---|---|
| `.resources/acpx` | ACP client/orchestrator and conformance tooling. | Has an agent/adapter registry, not a callable tool registry. Passes `mcpServers`; models tool calls by `toolCallId`, title, kind, status, raw input/output. |
| `.resources/collaborator-ai` | ACP client/orchestrator. | No registry found. Uses ACP tool update titles for display. |
| `.resources/goclaw` | ACP client/orchestrator. | No formal registry. Uses method switches and permission heuristics for ACP callbacks. |
| `.resources/harnss` | ACP client/orchestrator with Electron bridge. | Has ACP agent registry and UI rendering adapters, not a tool registry. Converts configured MCP servers to ACP `McpServer[]`. |
| `.resources/hermes` | ACP server/agent implementation plus ACP client shim. | Relevant: registers ACP-provided MCP servers into Hermes agent state and valid tool names; maps Hermes tools to ACP `ToolKind` and titles. |
| `.resources/multica` | ACP client/orchestrator. | No formal registry. Parses titles such as `terminal:` and `read:` for UI normalization. |
| `.resources/openclaw` | ACP server/client/runtime bridge. | Has internal tool catalog and plugin/MCP surfaces, but main ACP bridge does not expose an ACP tool registry and rejects per-session `mcpServers`. |
| `.resources/opencode` | Native ACP server. | Has a real internal `ToolRegistry`, but ACP does not expose it as a registry API. Accepts ACP `mcpServers` and converts them into internal MCP config. |
| `.resources/paperclip` | Docs/reference only. | Conceptual ACP references only. |
| `.resources/rayclaw` | ACP client/orchestrator. | Exposes ACP control as local `acp_*` tools; ACP-reported tool calls are telemetry, not registry entries. |
| `.resources/sandbox-agent` | ACP adapter/proxy/client package. | Agent launch registry only; no ACP tool registry found. |
| `.resources/t3code` | ACP schema/client/runtime package. | Schema and runtime tracking for ACP tool events; no broad tool registry. |

No meaningful ACP evidence was found in `.resources/cc-posts`, `.resources/chat`, `.resources/openfang`, `.resources/pi`, or `.resources/symphony`. `.resources/claude-code` had an `ACP` false positive inside an embedded/base64-like string, not implementation evidence.

## Deep Dives

### RayClaw

RayClaw is an ACP client/orchestrator. It spawns configured ACP agents, runs JSON-RPC lifecycle calls (`initialize`, `session/new`, `session/prompt`, `session/end`), and exposes ACP orchestration to RayClaw's primary LLM through local wrapper tools named `acp_coding`, `acp_new_session`, `acp_prompt`, `acp_end_session`, `acp_list_sessions`, `acp_submit_job`, and `acp_job_status`.

Those `acp_*` names are RayClaw's local tool registry convention, not ACP. RayClaw's ACP tool-call handling treats incoming `session/update` tool calls as observations and records them by title/raw input. It does not dispatch those reported ACP tool calls through RayClaw's local registry.

Important evidence:

- `.resources/rayclaw/src/acp.rs:436-453` initializes ACP with client capabilities, not a host tool registry.
- `.resources/rayclaw/src/acp.rs:752-843` handles `session/request_permission`.
- `.resources/rayclaw/src/acp.rs:846-984` parses ACP tool-call progress.
- `.resources/rayclaw/src/acp.rs:1521-1531` creates sessions with `mcpServers: []`.
- `.resources/rayclaw/src/tools/acp.rs:16-40` registers the local `acp_*` wrapper tools.
- `.resources/rayclaw/tests/acp_integration.rs:128-240` enforces local tool-name uniqueness, allowed characters, length, and collision checks.

Transferable points:

- Separate ACP orchestration tools from normal runtime tools.
- Treat ACP tool calls as child-agent telemetry unless AGH deliberately bridges them.
- Do not prefer `allow_always` as an automatic approval default the way RayClaw does under `auto_approve`; AGH should keep durable grants explicit.
- Reject or disambiguate sanitized name collisions rather than truncating.

### Harnss

Harnss is an ACP client/orchestrator with an Electron bridge and React UI. It converts renderer MCP server configs into ACP SDK `McpServer[]`, including stdio and remote transports, then passes them into `newSession` and `loadSession`. It also supports live reload through ACP `loadSession` when available.

Harnss does not consume or expose an ACP tool registry. Its "registry" evidence is an ACP agent registry and a UI-side static MCP renderer table. ACP tool calls are converted into UI messages keyed by `toolCallId`, using title/kind/raw input/output normalization.

Important evidence:

- `.resources/harnss/electron/src/ipc/acp-sessions.ts:193-215` converts MCP configs to ACP `McpServer[]`.
- `.resources/harnss/electron/src/ipc/acp-sessions.ts:365-483` wires ACP connection callbacks, event forwarding, and permission bridge.
- `.resources/harnss/electron/src/ipc/acp-sessions.ts:521-546` starts ACP sessions with MCP servers.
- `.resources/harnss/electron/src/ipc/acp-sessions.ts:793-828` reloads sessions with MCP servers.
- `.resources/harnss/src/hooks/useACP.ts:194-337` converts ACP tool events into UI messages.
- `.resources/harnss/src/hooks/useACP.ts:413-473` handles ACP permission requests.
- `.resources/harnss/src/lib/engine/acp-adapter.ts:267-358` derives display/tool renderer names from ACP title/kind/raw input.
- `.resources/harnss/src/components/McpToolContent.tsx:83-138` supports SDK-style `mcp__Server__tool` names and ACP-style `Tool: Server/tool` titles in UI rendering.

Transferable points:

- Normalize ACP event data at the boundary into AGH's canonical tool-call observation model.
- Preserve raw ACP permission options; do not collapse manual allow/deny into "first allow" or "first reject" if the protocol provides multiple option IDs.
- Keep rendering names separate from policy names.
- Pass MCP servers as session bootstrap/load data when AGH chooses per-session MCP support.

### ACPX

ACPX is a headless ACP client/orchestrator. Its registry is an agent/adapter registry mapping names like `codex`, `claude`, `gemini`, and others to launch commands. This is not a callable tool registry.

ACPX implements client authority callbacks such as filesystem read/write, terminal create/output/wait/kill/release, and `session/request_permission`. It parses `mcpServers` from config and passes them through to `session/new` and `session/load`. It does not discover or normalize MCP tools into a registry.

Important evidence:

- `.resources/acpx/src/agent-registry.ts:38-107` maps adapter names to commands.
- `.resources/acpx/src/mcp-servers.ts:100-177` parses MCP server configs.
- `.resources/acpx/src/acp/client.ts:475-538` wires ACP client callbacks and initialize capabilities.
- `.resources/acpx/src/acp/client.ts:638-693` passes `mcpServers` to `session/new` and `session/load`.
- `.resources/acpx/src/permissions.ts:98-152` implements coarse permission decisions.
- `.resources/acpx/src/session/conversation-model.ts:310-353` persists tool events keyed by tool call ID.
- `.resources/acpx/conformance/cases/021-prompt-post-success-drain.json:1-50` shows late tool updates can arrive after prompt success.

Transferable points:

- Keep agent/provider registries separate from Tool Registry.
- ACP compatibility includes filesystem and terminal callbacks where advertised; those callbacks must share AGH's registry policy engine or be routed through equivalent approval gates.
- Preserve distinct identities: AGH record IDs, ACP session IDs, provider-native session IDs, tool call IDs, and registry tool IDs.
- Add a protocol-aware drain/settle window for late `tool_call_update` events.

### OpenClaw

OpenClaw's main `openclaw acp` bridge is a Gateway-backed ACP server. It forwards prompts to the Gateway and translates Gateway events into ACP session updates. The main bridge advertises MCP HTTP/SSE support as disabled and rejects non-empty per-session `mcpServers`; its docs say MCP should be configured at the Gateway/agent layer.

OpenClaw has rich internal tool catalogs and plugin/MCP surfaces, but the main ACP bridge does not expose them as an ACP tool registry. Tool identity in ACP is display/event identity: title formatting plus inferred `ToolKind`.

Important evidence:

- `.resources/openclaw/src/acp/server.ts:4-13` and `.resources/openclaw/src/acp/server.ts:104-122` bootstrap the ACP stdio server.
- `.resources/openclaw/src/acp/translator.ts:519-540` advertises ACP capabilities.
- `.resources/openclaw/src/acp/translator.ts:542-603` handles session creation/loading.
- `.resources/openclaw/src/acp/translator.ts:1417-1424` rejects non-empty `mcpServers`.
- `.resources/openclaw/src/acp/translator.ts:848-940` maps Gateway tool events to ACP tool updates.
- `.resources/openclaw/src/acp/event-mapper.ts:297-342` formats tool titles and infers tool kind.
- `.resources/openclaw/src/agents/tool-catalog.ts:20-37` and `.resources/openclaw/src/agents/tool-catalog.ts:306-393` define a separate internal tool catalog.
- `.resources/openclaw/extensions/acpx/src/runtime-internals/mcp-proxy.mjs:33-64` shows the ACPX extension can inject MCP servers into embedded ACP sessions, unlike the main gateway bridge.

Transferable points:

- AGH must explicitly choose whether its ACP bridge accepts per-session `mcpServers` like OpenCode/Harnss/ACPX or rejects them like OpenClaw's gateway bridge.
- If AGH supports both runtime-managed MCP and ACP-provided MCP servers, precedence and collision rules must be explicit.
- Do not use substring heuristics for registry policy where explicit tool metadata is available.

### OpenCode

OpenCode implements a native ACP server and has a real internal `ToolRegistry`. This is the strongest local reference for how an agent can maintain a rich internal registry while ACP still sees only session lifecycle, MCP bootstrap, tool-call updates, and permission requests.

OpenCode accepts ACP per-session `mcpServers`, stores them in ACP session state, converts them into internal MCP config, and adds them through its SDK. Its internal MCP naming pattern exposes MCP tools as `sanitize(server) + "_" + sanitize(tool)` while preserving the original MCP tool name for the actual call.

Important evidence:

- `.resources/opencode/packages/opencode/src/cli/cmd/acp.ts:23-60` bootstraps `opencode acp`.
- `.resources/opencode/packages/opencode/src/acp/types.ts:1-16` and `.resources/opencode/packages/opencode/src/acp/session.ts:8-75` store ACP session state, including `mcpServers`.
- `.resources/opencode/packages/opencode/src/acp/agent.ts:534-578` advertises MCP support.
- `.resources/opencode/packages/opencode/src/acp/agent.ts:584-687` accepts MCP servers on session creation/loading.
- `.resources/opencode/packages/opencode/src/acp/agent.ts:1216-1254` converts ACP MCP servers into internal MCP config.
- `.resources/opencode/packages/opencode/src/mcp/index.ts:115-146` and `.resources/opencode/packages/opencode/src/mcp/index.ts:618-651` implement sanitized server/tool naming while preserving raw MCP names.
- `.resources/opencode/packages/opencode/src/tool/tool.ts:34-43` and `.resources/opencode/packages/opencode/src/tool/registry.ts:163-207` define internal tool definitions and registry behavior.
- `.resources/opencode/packages/opencode/src/acp/agent.ts:273-455` emits ACP tool-call lifecycle updates.
- `.resources/opencode/packages/opencode/src/acp/agent.ts:190-271` bridges internal permission events to ACP `session/request_permission`.

Transferable points:

- Keep AGH's internal registry richer than ACP.
- Store ACP-provided MCP servers in session state, not global daemon config.
- Preserve raw MCP server/tool names separately from the canonical AGH `ToolID`.
- Emit a stable ACP lifecycle, preferably `pending -> in_progress -> completed/failed`, even when the underlying runtime first reports a running event.
- Do not rely on a single-underscore sanitized naming scheme without collision diagnostics.

### Hermes, Multica, GoClaw, Sandbox-Agent, T3Code

These projects reinforce the same split:

- `.resources/hermes` is relevant because it registers ACP-provided MCP servers into agent state and valid tool names, then maps tool events into ACP kinds/titles. It has useful registry ideas, but ACP remains the session/event layer.
- `.resources/multica` and `.resources/goclaw` normalize ACP tool titles/kinds for display and permission heuristics; neither shows a protocol-level tool registry.
- `.resources/sandbox-agent` has ACP HTTP-to-stdio adapter and launch registry logic, but no callable ACP tool registry.
- `.resources/t3code` provides ACP schema/client/runtime tracking for tool-call events, not a broad registry.

## Design Implications For AGH

1. ACP compatibility is not a reason to avoid a daemon Tool Registry. ACP leaves tool discovery/execution models to the agent/runtime, or to MCP servers supplied to the session.

2. AGH should expose daemon-owned session tools through an AGH-hosted MCP server in the MVP. This matches the accepted ADR-002 direction and aligns with ACP's `mcpServers` field.

3. The registry's canonical ID must not be ACP `title`. Use one stable provider-safe `ToolID` across AGH and hosted MCP.

4. The registry should store one canonical callable identity plus metadata:
   - `ToolID`: provider-safe lower snake segments separated by reserved `__`, for example `agh__skill_view`.
   - `DisplayTitle`: user-facing title only.
   - `SourceRef`: structured provenance, for example built-in, MCP server, extension ID, bundle ID, provider ID.

5. Collision handling must be fail-closed:
   - Canonical `ToolID` collision: provider registration error or conflicted diagnostic.
   - Sanitized external-name collision: tool is not exposed to the session until disambiguated.
   - Display title collision: allowed, because titles are not policy identities.

6. Operator and session projections should remain separate:
   - Operator surfaces show unavailable, unauthorized, and conflicted tools with reason codes.
   - Session/model surfaces expose only callable tools after availability, authorization, approval ceiling, and collision checks.

7. ACP permission policy integration must remain ceiling-based:
   - `deny-all` denies by default.
   - `approve-reads` auto-approves only registry-classified read-only tools and ACP read/search callbacks AGH classifies as read-only.
   - `approve-all` skips approval prompts for otherwise allowed tools, but does not bypass registry deny rules, extension grants, session lineage, source trust, availability, hooks, or conflict checks.

8. ACP filesystem and terminal callbacks, if AGH advertises them, must not bypass Tool Registry policy. Either route them through the registry as first-class built-in tools or share the same policy/approval engine with equivalent telemetry and hooks.

9. ACP `ToolKind` should be explicit metadata on AGH descriptors. Heuristics from title/kind are fallback-only for external ACP events that AGH observes but does not own.

10. AGH should persist observed ACP tool calls separately from registry definitions. Observations are keyed by `toolCallId` within a session and carry title/kind/status/raw input/output. Registry entries are keyed by canonical `ToolID`.

11. AGH should support late tool-call updates after prompt completion by draining the ACP event stream for a bounded window or until protocol-specific completion conditions are met.

12. AGH should decide explicitly whether to accept third-party ACP `mcpServers` from clients:
    - If accepted, store them as session-scoped tool sources with clear precedence and conflict policy.
    - If rejected, document the OpenClaw-style stance and require MCP sources to be configured through AGH's registry/config lifecycle.
    - For this TechSpec, the safer MVP path is AGH-managed hosted MCP first, with acceptance of client-supplied MCP servers as a compatibility extension only if collision and source-trust rules are implemented.

## Accepted Naming And Collision Recommendation

Adopt one canonical public `ToolID` format:

- Canonical ID: provider-safe lower snake segments separated by reserved double underscore, for example `agh__skill_view`, `agh__tool_search`, `mcp__github__create_issue`, `ext__linear__search`.
- Hosted MCP `Tool.name`: same as the canonical `ToolID`.
- Display title: human-readable and non-unique, for example `View Skill`.
- Source/provenance: structured fields, not inferred solely from name prefixes.
- No shadowing: providers cannot replace an existing canonical ID unless they are the same source updating the same record.
- No silent truncation: if sanitization or length rules would collide, registration/session projection marks the tool conflicted and hides it from the session surface.
- Policy, dispatch, telemetry, hooks, CLI, HTTP, UDS, and hosted MCP all use the same `ToolID`.

This keeps AGH's internal registry expressive while respecting MCP wire compatibility and ACP's event-oriented model.

## Evidence Summary

Official protocol sources:

- ACP schema: <https://agentclientprotocol.com/protocol/schema>
- ACP agent registry: <https://agentclientprotocol.com/registry>
- MCP draft schema: <https://modelcontextprotocol.io/specification/draft/schema>

Primary local evidence:

- `.resources/rayclaw/src/acp.rs`
- `.resources/rayclaw/src/tools/acp.rs`
- `.resources/rayclaw/tests/acp_integration.rs`
- `.resources/harnss/electron/src/ipc/acp-sessions.ts`
- `.resources/harnss/src/hooks/useACP.ts`
- `.resources/harnss/src/lib/engine/acp-adapter.ts`
- `.resources/acpx/src/acp/client.ts`
- `.resources/acpx/src/mcp-servers.ts`
- `.resources/acpx/src/permissions.ts`
- `.resources/openclaw/src/acp/translator.ts`
- `.resources/openclaw/src/acp/event-mapper.ts`
- `.resources/opencode/packages/opencode/src/acp/agent.ts`
- `.resources/opencode/packages/opencode/src/tool/registry.ts`
- `.resources/opencode/packages/opencode/src/mcp/index.ts`

Subagent inventory evidence:

- ACP implementations/usages: `acpx`, `collaborator-ai`, `goclaw`, `harnss`, `hermes`, `multica`, `openclaw`, `opencode`, `paperclip`, `rayclaw`, `sandbox-agent`, `t3code`.
- No meaningful ACP evidence: `cc-posts`, `chat`, `openfang`, `pi`, `symphony`.
- Excluded false positive: `claude-code`.
