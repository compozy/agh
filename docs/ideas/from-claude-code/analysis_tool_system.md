# Tool System Analysis

## How It Works (in Claude Code)

### 1. The `buildTool()` Factory Pattern

Every tool in Claude Code is created via the `buildTool()` factory function defined in `Tool.ts`. This function takes a partial tool definition (`ToolDef`) and produces a complete `Tool` object by merging user-provided methods with safe defaults.

The factory fills in seven "defaultable" methods when omitted:

- `isEnabled` -> `() => true`
- `isConcurrencySafe` -> `() => false` (fail-closed: assume NOT safe for parallel execution)
- `isReadOnly` -> `() => false` (fail-closed: assume writes)
- `isDestructive` -> `() => false`
- `checkPermissions` -> `Promise.resolve({ behavior: 'allow', updatedInput: input })` (defer to the general permission system)
- `toAutoClassifierInput` -> `() => ''` (skip auto-mode classifier by default; security-relevant tools MUST override)
- `userFacingName` -> `() => name`

Each tool declaration includes:

- **`name`** - unique identifier
- **`aliases`** - backward compatibility when tools are renamed
- **`searchHint`** - 3-10 word capability phrase for ToolSearch keyword matching
- **`inputSchema`** - Zod schema for validating input (also used to generate JSON Schema for the API)
- **`outputSchema`** - optional Zod schema for output typing
- **`call()`** - the actual execution function
- **`description()`** / **`prompt()`** - dynamic description generation (can vary by context)
- **`checkPermissions()`** - tool-specific permission logic
- **`validateInput()`** - input validation before permission check
- **`isReadOnly()`** / **`isDestructive()`** / **`isConcurrencySafe()`** - behavioral flags
- **`toAutoClassifierInput()`** - produces a compact representation for the auto-mode security classifier
- **`maxResultSizeChars`** - threshold for persisting large results to disk
- **`shouldDefer`** / **`alwaysLoad`** - controls lazy loading via ToolSearch
- **`isSearchOrReadCommand()`** - tells the UI whether to collapse the result
- **`interruptBehavior()`** - `'cancel'` or `'block'` when user submits new message mid-execution
- **`preparePermissionMatcher()`** - factory for hook `if` condition matching (parsed once, called per pattern)
- **`backfillObservableInput()`** - adds legacy/derived fields to observable copies without mutating the API-bound input (preserves prompt cache)
- **Rendering methods** - `renderToolUseMessage`, `renderToolResultMessage`, `renderToolUseProgressMessage`, etc. (React components)
- **`mapToolResultToToolResultBlockParam()`** - serializes output into the API's `tool_result` format
- **`extractSearchText()`** - flattened text for transcript search indexing

The type system uses `BuiltTool<D>` which mirrors the runtime `{ ...TOOL_DEFAULTS, ...def }` spread at the type level, preserving exact types from the definition.

### 2. The Permission System

The permission system has **five layers** operating in sequence:

#### Layer 1: Permission Modes

Six modes control the baseline behavior:

- **`default`** - ask for permission on non-read operations
- **`acceptEdits`** - auto-allow file edits in the working directory
- **`bypassPermissions`** - skip all permission checks (requires explicit enable)
- **`dontAsk`** - convert all `ask` decisions to `deny` (non-interactive)
- **`auto`** - use ML classifier to decide (the "YOLO mode")
- **`plan`** - model-initiated planning mode

#### Layer 2: Static Rule Matching (Allow/Deny/Ask)

Rules are stored in settings at three scopes: `userSettings`, `projectSettings`, `localSettings`, plus `cliArg`, `command`, `session` sources. Each rule has:

- `toolName` - which tool it applies to (e.g., `"Bash"`)
- `ruleContent` - optional parameter match (e.g., `"git commit:*"` for prefix matching)
- `behavior` - `allow` | `deny` | `ask`

Rule evaluation order:

1. Check blanket deny rules (tool entirely blocked) -> filter tool from API
2. Check blanket allow rules (tool entirely allowed) -> auto-allow
3. Check content-specific rules via `preparePermissionMatcher()` for pattern matching
4. Tool-specific `checkPermissions()` for complex logic (e.g., Bash subcommand analysis)

#### Layer 3: Tool-Specific Permission Checks

Each tool implements `checkPermissions()` for domain-specific logic. For Bash, this is extremely complex:

- Parse command into subcommands via `splitCommand_DEPRECATED` + tree-sitter AST
- For each subcommand: strip safe env vars, strip safe wrappers (timeout/nice/nohup), extract base command
- Check against deny/allow/ask rules with wildcard pattern matching
- Run `bashCommandIsSafe_DEPRECATED` security checks (20+ validators)
- Check path constraints, sed constraints, read-only validation
- For compound commands, aggregate subcommand results (all must pass)

#### Layer 4: Bash Security Validators (bashSecurity.ts)

23 numbered security check IDs covering:

- Incomplete commands, obfuscated flags, shell metacharacters
- Command substitution patterns (`$()`, `${}`, backticks, process substitution)
- Input/output redirection validation
- IFS injection, `/proc/environ` access, brace expansion
- Control characters, unicode whitespace
- Zsh-specific dangerous commands (zmodload, sysopen, ztcp, etc.)
- Comment-quote desync attacks, quoted newlines

#### Layer 5: Auto-Mode ML Classifier (YOLO Classifier)

When in `auto` mode, a separate LLM call classifies tool actions:

- **Transcript compression**: Conversation history is compressed into a compact format where each tool call becomes a single line (e.g., `Bash ls` or `{"Bash":"ls"}`). Each tool controls its representation via `toAutoClassifierInput()`.
- **Two-stage XML classifier**: Stage 1 (fast) runs with `max_tokens=64` for immediate yes/no. If allowed, returns immediately. If blocked, escalates to Stage 2 (thinking) with chain-of-thought reasoning to reduce false positives.
- **Fast-path optimizations**: Before running the classifier, checks: (1) acceptEdits mode would allow it, (2) tool is on the safe allowlist (read-only tools like Grep, Glob, FileRead)
- **Fail-closed**: On API errors, parse failures, or aborts -> always block
- **Prompt caching**: System prompt and CLAUDE.md get `cache_control` markers; Stage 2 shares the same prefix as Stage 1 for cache hits
- **Denial tracking**: Consecutive denials trigger fallback to interactive prompting to prevent infinite loops

#### Permission Decision Flow

```
validateInput() -> checkPermissions() -> hasPermissionsToUseToolInner()
  |                    |                       |
  v                    v                       v
  reject/pass    allow/deny/ask/pass     Layer 2 rules check
                                               |
                                     [if ask] -> mode transform:
                                         dontAsk -> deny
                                         auto -> classifier
                                         default -> prompt user
```

### 3. The Bash AST Parser

Claude Code has a **pure-TypeScript bash parser** (`utils/bash/bashParser.ts`) producing tree-sitter-bash-compatible ASTs. Key design decisions:

#### Parser Architecture

- **No WASM dependency**: Pure TS implementation for portability
- **50ms timeout**: Bails out on pathological/adversarial input
- **50,000 node budget**: Prevents OOM on deeply nested input
- **UTF-8 byte offsets**: Node positions use byte offsets matching tree-sitter convention
- **Golden corpus validated**: Tested against 3,449 inputs from the WASM parser

#### Tokenizer

The lexer handles: WORD, NUMBER, OP, NEWLINE, COMMENT, DQUOTE, SQUOTE, ANSI_C, DOLLAR, DOLLAR_PAREN, DOLLAR_BRACE, DOLLAR_DPAREN, BACKTICK, LT_PAREN, GT_PAREN, EOF. It tracks both JS string index and UTF-8 byte offset simultaneously.

#### AST Security Analysis (`ast.ts`)

The core design principle is **FAIL-CLOSED via EXPLICIT ALLOWLIST**:

- Only known-safe node types are allowed (STRUCTURAL_TYPES: `program`, `list`, `pipeline`, `redirected_statement`)
- DANGEROUS_TYPES explicitly enumerated: `command_substitution`, `process_substitution`, `expansion`, `subshell`, `for_statement`, etc.
- Any unknown node type -> `'too-complex'` -> requires user permission
- The analysis answers exactly one question: "Can we produce a trustworthy argv[] for each simple command?"

Output types:

- `{ kind: 'simple', commands: SimpleCommand[] }` - fully analyzed
- `{ kind: 'too-complex', reason: string, nodeType?: string }` - requires permission prompt
- `{ kind: 'parse-unavailable' }` - parser not ready

SimpleCommand structure:

```typescript
type SimpleCommand = {
  argv: string[]; // argv[0] is command, rest are args
  envVars: { name: string; value: string }[]; // Leading VAR=val
  redirects: Redirect[]; // File redirections
  text: string; // Original source span
};
```

Safe variable resolution: Known shell variables ($HOME, $PWD, $USER, $PATH) are resolved inside strings. $() substitutions produce `__CMDSUB_OUTPUT__` placeholder; inner commands are checked separately. Variables set in the same command via assignment are tracked in a `varScope`.

#### Wrapper Stripping

Both regex-based (`stripSafeWrappers`) and argv-based (`stripWrappersFromArgv`) stripping of known-safe wrappers: `timeout`, `time`, `nice`, `nohup`, `stdbuf`. Safe env vars (`GOARCH`, `NODE_ENV`, `RUST_LOG`, etc.) are stripped before permission rule matching.

Security notes:

- Bare shell prefixes (`sh`, `bash`, `zsh`, `env`, `xargs`, `sudo`) are NEVER suggested as allow-rule prefixes
- Env var value pattern uses allowlist `[A-Za-z0-9_./:-]+` to reject `$()`, backticks, `;|&`
- Horizontal whitespace only (`[ \t]+`) after env var values, NOT `\s+` which matches newlines

### 4. Tool Result Injection via `tool_result` Messages

Tool results flow back to the model through several mechanisms:

#### Standard tool_result

`mapToolResultToToolResultBlockParam()` serializes tool output into the API's `tool_result` format. Each tool controls its own serialization.

#### Large Result Persistence

When a tool result exceeds `maxResultSizeChars` (default 50k chars, configurable per-tool via GrowthBook):

1. Full result is written to disk at `{projectDir}/{sessionId}/tool-results/{toolUseId}.json`
2. A preview is generated (first N bytes)
3. Model receives: `<persisted-output>Preview content... [Full output saved to /path/file]</persisted-output>`
4. Read tool has `maxResultSizeChars: Infinity` to avoid circular persist->read->persist loops

#### Content Replacement State

Per-conversation thread state tracks which tool results have been replaced/cleared, preventing the aggregate tool result budget from being exceeded. Shared across cache-sharing forks.

#### ToolSearch tool_reference Injection

When ToolSearch finds matching deferred tools, it returns `tool_reference` blocks:

```typescript
content: content.matches.map(name => ({
  type: "tool_reference",
  tool_name: name,
}));
```

The API expands these into full tool definitions in the model's context. The message history is scanned via `extractDiscoveredToolNames()` to include only discovered tools in subsequent API requests.

#### Knowledge Injection via tool_result

Tools can inject contextual knowledge through their results:

- System reminders can be appended alongside tool results
- `newMessages` in `ToolResult<T>` can inject user/assistant/system messages
- `contextModifier` can modify the `ToolUseContext` for subsequent tools (only for non-concurrency-safe tools)
- `contentBlocks` in permission decisions can include images (e.g., user-pasted screenshots as feedback)

### 5. Tool Schema Caching and Deferred Loading

#### Deferred Loading Architecture

Tools are categorized as deferred or always-loaded:

**Deferred** (loaded via ToolSearch):

- All MCP tools (unless `alwaysLoad: true` via `_meta['anthropic/alwaysLoad']`)
- Built-in tools with `shouldDefer: true` (e.g., NotebookEdit, WebFetch, WebSearch)

**Never deferred**:

- ToolSearch itself (the model needs it to load everything else)
- Agent tool when fork-subagent is enabled
- Brief tool (primary communication channel)

Deferred tools are sent with `defer_loading: true` in the API request. The model sees only tool names (no schema) until it uses ToolSearch to fetch them.

#### Tool Description Caching

`getToolDescriptionMemoized` caches tool descriptions by name for keyword search scoring. Cache is invalidated when the set of deferred tools changes (MCP servers connect/disconnect).

#### Auto-threshold for Tool Search

When `ENABLE_TOOL_SEARCH=auto` (or `auto:N`), tool search activates only when deferred tool descriptions exceed N% of the context window (default 10%). Uses exact token counting API when available, falls back to character heuristic (2.5 chars/token).

#### Prompt Cache Stability

Built-in tools are sorted alphabetically as a contiguous prefix, MCP tools sorted separately as a suffix. This ensures the server's cache breakpoint after the last built-in tool remains stable even when MCP tools change. `uniqBy('name')` ensures built-ins win on name conflicts.

#### Discovered Tool Tracking

`extractDiscoveredToolNames()` scans message history for `tool_reference` blocks from ToolSearch results. On compaction, the discovered set is snapshotted onto `compactMetadata.preCompactDiscoveredTools` so tools aren't lost.

#### Delta Announcements

Instead of re-listing all deferred tools on every turn, the system tracks deltas:

- New tools added (MCP server connected)
- Tools removed (MCP server disconnected)
- Announced via `deferred_tools_delta` attachment messages

---

## Key Patterns Worth Adopting

### 1. buildTool() Factory with Fail-Closed Defaults

The factory pattern ensures every tool has consistent behavior without boilerplate. The key insight is **fail-closed defaults**: `isConcurrencySafe: false`, `isReadOnly: false`. Tools must explicitly opt into capabilities.

```typescript
// From Tool.ts
const TOOL_DEFAULTS = {
  isEnabled: () => true,
  isConcurrencySafe: (_input?: unknown) => false, // assume NOT safe
  isReadOnly: (_input?: unknown) => false, // assume writes
  isDestructive: (_input?: unknown) => false,
  checkPermissions: input => Promise.resolve({ behavior: "allow", updatedInput: input }),
  toAutoClassifierInput: (_input?: unknown) => "", // skip classifier by default
  userFacingName: (_input?: unknown) => "",
};

export function buildTool<D extends AnyToolDef>(def: D): BuiltTool<D> {
  return { ...TOOL_DEFAULTS, userFacingName: () => def.name, ...def } as BuiltTool<D>;
}
```

### 2. Layered Permission Architecture

Five distinct layers, each with a clear responsibility:

- **Modes** set the baseline policy
- **Static rules** handle known-good/known-bad patterns
- **Tool-specific checks** handle domain complexity
- **Security validators** catch adversarial patterns
- **ML classifier** handles ambiguous cases

The critical insight: **static fast-paths before expensive classifier calls**. The acceptEdits check and safe-tool allowlist avoid the classifier API call for 80%+ of tool invocations.

### 3. AST-Based Security with Explicit Allowlist

```typescript
// From ast.ts - the core safety property
const STRUCTURAL_TYPES = new Set(["program", "list", "pipeline", "redirected_statement"]);
// Any node type NOT explicitly handled -> too-complex -> ask user
```

This is far more secure than a denylist approach. New bash features are automatically blocked until explicitly analyzed and allowlisted.

### 4. toAutoClassifierInput() - Tool-Controlled Security Projection

Each tool controls what the security classifier sees. Read-only tools return `''` (skip classifier). Bash returns the command string. File edit returns a compact representation. This prevents information leakage and reduces classifier token usage.

```typescript
// GlobTool - just the pattern
toAutoClassifierInput(input) { return input.pattern }

// BashTool - the full command (security-critical)
toAutoClassifierInput(input) { return input.command }
```

### 5. Deferred Tool Loading with ToolSearch

The `tool_reference` pattern is elegant: instead of sending all tool schemas upfront (which can consume 10%+ of context), deferred tools appear as names only. ToolSearch returns `tool_reference` blocks that the API expands inline. This keeps the initial prompt lean while making all tools discoverable.

### 6. Command Semantic Interpretation

```typescript
// From commandSemantics.ts
const COMMAND_SEMANTICS = new Map([
  [
    "grep",
    exitCode => ({
      isError: exitCode >= 2,
      message: exitCode === 1 ? "No matches found" : undefined,
    }),
  ],
  [
    "diff",
    exitCode => ({ isError: exitCode >= 2, message: exitCode === 1 ? "Files differ" : undefined }),
  ],
]);
```

Rather than treating all non-zero exit codes as errors, tools understand command-specific semantics. `grep` returning 1 means "no matches" not "error".

### 7. Progressive Security Checks with Cap

```typescript
// From bashPermissions.ts
export const MAX_SUBCOMMANDS_FOR_SECURITY_CHECK = 50;
```

Complex compound commands that would produce exponential subcommand growth are capped at 50, falling back to `ask` (safe default). This prevents DoS while maintaining security.

### 8. Safe Wrapper and Env Var Stripping

The two-phase stripping approach is important:

- **Phase 1**: Strip safe env vars (`NODE_ENV`, `GOARCH`, etc.) before wrapper stripping
- **Phase 2**: Strip safe wrappers (`timeout`, `nice`, `nohup`) WITHOUT stripping env vars (because `timeout VAR=val cmd` means VAR=val IS the command, not an env var)

This prevents `nohup rm -rf /` from matching a `Bash(rm:*)` allow rule correctly.

---

## Ideas for Our System

### 1. Go Tool Factory with Functional Options

Translate the `buildTool()` pattern to Go using functional options:

```go
type Tool struct {
    Name              string
    InputSchema       *jsonschema.Schema
    IsConcurrencySafe func(input map[string]any) bool  // default: false
    IsReadOnly        func(input map[string]any) bool  // default: false
    CheckPermissions  func(input map[string]any, ctx *ToolContext) (*PermissionResult, error)
    Call              func(input map[string]any, ctx *ToolContext) (*ToolResult, error)
    MaxResultSize     int  // bytes before persisting to disk
}

func NewTool(name string, opts ...ToolOption) *Tool {
    t := &Tool{
        Name:              name,
        IsConcurrencySafe: func(_ map[string]any) bool { return false },
        IsReadOnly:        func(_ map[string]any) bool { return false },
        MaxResultSize:     50_000,
        // ... other fail-closed defaults
    }
    for _, opt := range opts { opt(t) }
    return t
}
```

### 2. Three-Tier Permission System

Simplify to three tiers for our Go system:

```go
type PermissionTier int
const (
    TierAllow    PermissionTier = iota  // auto-approve (read-only tools, safe commands)
    TierClassify                         // needs classifier check (writes, network)
    TierRequire                          // always requires human approval (destructive ops)
)
```

Each tool declares its tier. The kernel checks:

1. Static deny rules (loaded from config) -> reject immediately
2. Static allow rules -> approve immediately
3. Tool-declared tier -> route to appropriate handler

### 3. Bash Command Safety via AST in Go

Use a Go bash parser (e.g., `mvdan.cc/sh/v3/syntax`) for AST-based command analysis:

```go
type CommandAnalysis struct {
    Kind     string          // "simple", "too-complex", "parse-error"
    Commands []SimpleCommand
    Reason   string          // why too-complex
}

type SimpleCommand struct {
    Argv      []string
    EnvVars   map[string]string
    Redirects []Redirect
    Text      string
}
```

The key principle to adopt: **explicit allowlist of node types**. Anything not in the allowlist -> require permission.

### 4. Tool Result Budget with Disk Persistence

Implement result persistence for large outputs:

```go
type ToolResultStore struct {
    SessionDir     string
    MaxResultChars int  // per-tool, default 50k
}

func (s *ToolResultStore) Store(toolUseID string, result string) (string, error) {
    if len(result) <= s.MaxResultChars {
        return result, nil
    }
    path := filepath.Join(s.SessionDir, "tool-results", toolUseID+".txt")
    os.WriteFile(path, []byte(result), 0644)
    preview := result[:min(4096, len(result))]
    return fmt.Sprintf("<persisted-output>%s\n[Full output: %s]</persisted-output>", preview, path), nil
}
```

### 5. Deferred Tool Registry

Implement lazy tool loading for MCP tools:

```go
type ToolRegistry struct {
    eager    map[string]*Tool      // always available
    deferred map[string]*Tool      // loaded on demand
    discovered map[string]struct{} // discovered this session
}

func (r *ToolRegistry) Search(query string) []*Tool {
    // Keyword search over deferred tools
    // Return tool_reference blocks for API expansion
}
```

### 6. Security Validator Pipeline

Create a pipeline of composable validators for bash commands:

```go
type SecurityValidator func(cmd string, ctx *ValidationContext) *SecurityViolation

var bashValidators = []SecurityValidator{
    validateIncompleteCommands,
    validateCommandSubstitution,
    validateRedirections,
    validateBraceExpansion,
    validateControlChars,
    validateUnicodeWhitespace,
    validateZshDangerousCommands,
    // ... 20+ validators
}

func ValidateCommand(cmd string) *SecurityViolation {
    ctx := newValidationContext(cmd)
    for _, v := range bashValidators {
        if violation := v(cmd, ctx); violation != nil {
            return violation
        }
    }
    return nil
}
```

### 7. Classifier Input Projection Per Tool

Each tool should control what the security classifier sees:

```go
type Tool struct {
    // ... other fields
    ClassifierInput func(input map[string]any) string
}

// Read-only tool: skip classifier
NewTool("Glob", WithClassifierInput(func(_ map[string]any) string { return "" }))

// Bash: expose full command
NewTool("Bash", WithClassifierInput(func(input map[string]any) string {
    return input["command"].(string)
}))
```

### 8. Dangerous Pattern Registry

Maintain a registry of dangerous command patterns, separate from the parser:

```go
var dangerousPatterns = []DangerousPattern{
    {Pattern: `\$\(`, Message: "$() command substitution"},
    {Pattern: `\$\{`, Message: "${} parameter substitution"},
    {Pattern: `<\(`,  Message: "process substitution <()"},
    {Pattern: `>\(`,  Message: "process substitution >()"},
}

var dangerousBashCommands = map[string]bool{
    "zmodload": true, "emulate": true, "sysopen": true,
    "syswrite": true, "zpty": true, "ztcp": true,
}
```

---

## Key Files Reference

| File                                      | Description                                                                                                                       |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `Tool.ts`                                 | Core Tool type definition with 60+ fields, `buildTool()` factory with fail-closed defaults, ToolUseContext type                   |
| `tools.ts`                                | Tool registry: `getAllBaseTools()`, `getTools()`, `assembleToolPool()`, feature-flag gated tool inclusion                         |
| `constants/tools.ts`                      | Tool allowlists/denylists for agents: `ALL_AGENT_DISALLOWED_TOOLS`, `ASYNC_AGENT_ALLOWED_TOOLS`, `COORDINATOR_MODE_ALLOWED_TOOLS` |
| `tools/BashTool/BashTool.tsx`             | Bash tool implementation: command execution, search/read classification, progress tracking, image output handling                 |
| `tools/BashTool/bashPermissions.ts`       | Bash permission logic: safe env vars/wrappers stripping, rule matching, compound command analysis, 700+ lines                     |
| `tools/BashTool/bashSecurity.ts`          | 23 security validators: command substitution, redirections, IFS injection, brace expansion, Zsh-specific attacks                  |
| `tools/BashTool/commandSemantics.ts`      | Exit code interpretation per command (grep 1 = no matches, diff 1 = files differ)                                                 |
| `tools/BashTool/readOnlyValidation.ts`    | Read-only command validation with per-command flag allowlists (git, docker, gh, ripgrep, pyright)                                 |
| `tools/BashTool/pathValidation.ts`        | Path constraint checking for bash commands                                                                                        |
| `tools/BashTool/sedValidation.ts`         | Sed-specific command validation                                                                                                   |
| `utils/bash/bashParser.ts`                | Pure-TypeScript bash parser: tokenizer, lexer, full AST generation matching tree-sitter-bash output                               |
| `utils/bash/ast.ts`                       | AST-based security analysis: fail-closed allowlist, SimpleCommand extraction, wrapper/variable resolution                         |
| `utils/permissions/permissions.ts`        | Central permission engine: `hasPermissionsToUseTool()`, rule evaluation, auto-mode classifier dispatch, denial tracking           |
| `utils/permissions/PermissionResult.ts`   | Permission result types: `allow`, `deny`, `ask`, `passthrough`                                                                    |
| `utils/permissions/PermissionRule.ts`     | Rule types: `PermissionBehavior`, `PermissionRuleValue`, `PermissionRule`                                                         |
| `utils/permissions/yoloClassifier.ts`     | Auto-mode ML classifier: 2-stage XML classification, transcript compression, prompt construction, 1500 lines                      |
| `utils/permissions/bashClassifier.ts`     | Bash-specific classifier (stub in external builds, full ML classifier in ant builds)                                              |
| `utils/permissions/classifierDecision.ts` | Safe tool allowlist for auto-mode: tools that skip classifier entirely                                                            |
| `utils/permissions/dangerousPatterns.ts`  | Dangerous bash/PowerShell patterns: code execution entry points, interpreter prefixes                                             |
| `utils/permissions/shellRuleMatching.ts`  | Shell command rule matching: wildcard patterns, prefix extraction                                                                 |
| `utils/toolResultStorage.ts`              | Large result persistence: disk storage, preview generation, per-tool thresholds                                                   |
| `utils/toolSearch.ts`                     | Tool search infrastructure: auto-threshold, deferred tool tracking, delta announcements, 750 lines                                |
| `tools/ToolSearchTool/ToolSearchTool.ts`  | ToolSearch tool: keyword search, `select:` syntax, `tool_reference` result blocks                                                 |
| `tools/ToolSearchTool/prompt.ts`          | ToolSearch prompt: `isDeferredTool()` logic, deferred tool formatting                                                             |
| `tools/GlobTool/GlobTool.ts`              | Clean example of a simple read-only tool using `buildTool()`                                                                      |
| `types/permissions.ts`                    | Canonical permission types: modes, rules, decisions, reasons - extracted to break import cycles                                   |
