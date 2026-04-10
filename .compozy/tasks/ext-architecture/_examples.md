# Extension Architecture — High-Level API Examples

Examples showing how extension authors interact with AGH using subprocess extensions (Go and TypeScript).

---

## 1. Subprocess Hook Extension (Go)

A content safety validator that blocks prompts containing secrets.

### Manifest (`extension.toml`)

```toml
[extension]
name = "secret-guard"
version = "0.1.0"
description = "Blocks prompts that contain API keys or secrets"
min_agh_version = "0.5.0"

[capabilities]
provides = ["content.validate"]

[[resources.hooks]]
name = "secret-guard-hook"
event = "input.pre_submit"
mode = "sync"
executor.kind = "subprocess"
executor.command = "./bin/secret-guard"
executor.args = ["--hook", "input_pre_submit"]

[subprocess]
command = "./bin/secret-guard"
args = ["serve"]

[security]
capabilities = ["message.read"]
```

### Extension Code (Go)

```go
package main

import (
    "context"
    "strings"

    agh "github.com/anthropics/agh/sdk/go" // placeholder — final module path TBD
)

func main() {
    ext := agh.NewExtension(agh.ExtensionConfig{
        Name:    "secret-guard",
        Version: "0.1.0",
    })

    patterns := []string{"sk-", "AKIA", "ghp_", "-----BEGIN RSA"}

    ext.Handle("execute_hook", func(ctx context.Context, params agh.HookPayload) (*agh.HookResult, error) {
        for _, pat := range patterns {
            if strings.Contains(params.Message, pat) {
                return &agh.HookResult{
                    Allow:  false,
                    Reason: fmt.Sprintf("Message contains a potential secret (pattern: %s)", pat),
                }, nil
            }
        }
        return &agh.HookResult{Allow: true}, nil
    })

    ext.Start()
}
```

### Build & Install

```bash
# Build
go build -o bin/secret-guard .

# Install
agh extension install ./secret-guard/
```

---

## 2. Subprocess Extension — Memory Backend (Go)

A pgvector-backed memory backend that replaces AGH's default SQLite memory.

### Manifest (`extension.toml`)

```toml
[extension]
name = "pgvector-memory"
version = "0.2.0"
description = "PostgreSQL pgvector semantic memory backend"
type = "subprocess"
min_agh_version = "0.5.0"

[capabilities]
provides = ["memory.backend"]

[actions]
requires = [
    "sessions/list",
    "sessions/events",
]

[subprocess]
command = "./bin/agh-ext-pgvector"
args = ["serve"]
health_check_interval = "30s"
shutdown_timeout = "10s"

[subprocess.env]
DATABASE_URL = "{{env:PGVECTOR_DATABASE_URL}}"

[resources]
skills = ["skills/"]

[security]
capabilities = ["memory.read", "memory.write", "session.read"]
```

### Extension Code (Go)

```go
package main

import (
    "context"
    "log"

    agh "github.com/anthropics/agh/sdk/go" // placeholder — final module path TBD
)

func main() {
    ext := agh.NewExtension(agh.ExtensionConfig{
        Name:    "pgvector-memory",
        Version: "0.2.0",
        Capabilities: []string{"memory.backend"},
    })

    db := connectPgvector(os.Getenv("DATABASE_URL"))

    // Handle memory/store — AGH calls this when storing a memory
    ext.Handle("memory/store", func(ctx context.Context, params agh.StoreParams) (*agh.StoreResult, error) {
        embedding := embed(params.Content)
        err := db.Insert(ctx, params.Key, params.Content, embedding, params.Tags)
        if err != nil {
            return nil, fmt.Errorf("pgvector store: %w", err)
        }
        return &agh.StoreResult{}, nil
    })

    // Handle memory/recall — AGH calls this when searching memory
    ext.Handle("memory/recall", func(ctx context.Context, params agh.RecallParams) (*agh.RecallResult, error) {
        embedding := embed(params.Query)
        rows, err := db.Search(ctx, embedding, params.Limit)
        if err != nil {
            return nil, fmt.Errorf("pgvector recall: %w", err)
        }
        entries := make([]agh.MemoryEntry, len(rows))
        for i, r := range rows {
            entries[i] = agh.MemoryEntry{Key: r.Key, Content: r.Content, Score: r.Score}
        }
        return &agh.RecallResult{Entries: entries}, nil
    })

    // Use Host API — read session events for context enrichment
    ext.OnReady(func(host *agh.HostAPI) {
        sessions, _ := host.Sessions.List(context.Background())
        log.Printf("pgvector-memory connected. %d active sessions.", len(sessions))
    })

    // Handle health checks
    ext.Handle("health_check", func(ctx context.Context, _ any) (*agh.HealthResult, error) {
        err := db.Ping(ctx)
        if err != nil {
            return &agh.HealthResult{Healthy: false, Message: err.Error()}, nil
        }
        return &agh.HealthResult{Healthy: true}, nil
    })

    ext.Start() // Blocks, reads stdin, writes stdout
}
```

---

## 3. Subprocess Extension — Memory Backend (TypeScript)

The same pgvector backend, but in TypeScript using `@agh/extension-sdk`.

### Manifest (`extension.toml`)

```toml
[extension]
name = "pgvector-memory-ts"
version = "0.2.0"
description = "PostgreSQL pgvector memory backend (TypeScript)"
type = "subprocess"
min_agh_version = "0.5.0"

[capabilities]
provides = ["memory.backend"]

[actions]
requires = ["sessions/list"]

[subprocess]
command = "node"
args = ["dist/index.js"]
health_check_interval = "30s"

[subprocess.env]
DATABASE_URL = "{{env:PGVECTOR_DATABASE_URL}}"

[security]
capabilities = ["memory.read", "memory.write", "session.read"]
```

### Extension Code (TypeScript)

```typescript
import { Extension, HostAPI, StoreParams, RecallParams } from '@agh/extension-sdk';
import { PgVector } from './pgvector';

const ext = new Extension({
    name: 'pgvector-memory-ts',
    version: '0.2.0',
    capabilities: { provides: ['memory.backend'] },
    actions: { requires: ['sessions/list'] },
});

const db = new PgVector(process.env.DATABASE_URL!);

// Handle memory/store
ext.handle('memory/store', async (ctx, params: StoreParams) => {
    const embedding = await embed(params.content);
    await db.insert(params.key, params.content, embedding, params.tags);
    return {};
});

// Handle memory/recall
ext.handle('memory/recall', async (ctx, params: RecallParams) => {
    const embedding = await embed(params.query);
    const rows = await db.search(embedding, params.limit ?? 10);
    return {
        entries: rows.map(r => ({
            key: r.key,
            content: r.content,
            score: r.score,
        })),
    };
});

// Use Host API on startup
ext.onReady(async (host: HostAPI) => {
    const sessions = await host.sessions.list();
    console.error(`pgvector-memory-ts connected. ${sessions.length} active sessions.`);
});

// Health check
ext.handle('health_check', async () => {
    const ok = await db.ping();
    return { healthy: ok };
});

ext.start();
```

---

## 4. Subprocess Hook Extension (TypeScript)

A prompt enhancer that adds workspace context to every prompt.

### Manifest (`extension.toml`)

```toml
[extension]
name = "prompt-enhancer"
version = "0.1.0"
description = "Adds workspace context to every prompt"
min_agh_version = "0.5.0"

[capabilities]
provides = ["prompt.provider"]

[[resources.hooks]]
name = "workspace-context"
event = "prompt.post_assemble"
mode = "sync"
executor.kind = "subprocess"
executor.command = "node"
executor.args = ["dist/index.js", "--hook", "prompt_post_assemble"]

[subprocess]
command = "node"
args = ["dist/index.js", "serve"]

[security]
capabilities = ["message.read", "message.write"]
```

### Extension Code (TypeScript)

```typescript
import { Extension } from '@agh/extension-sdk';

const ext = new Extension({
    name: 'prompt-enhancer',
    version: '0.1.0',
    capabilities: { provides: ['prompt.provider'] },
});

ext.handle('execute_hook', async (ctx, params: {
    session_id: string;
    agent_name: string;
    workspace_root: string;
    prompt: string;
}) => {
    return {
        updated_prompt: `[Workspace: ${params.workspace_root}]\n\n${params.prompt}`,
    };
});

ext.start();
```

---

## 5. Subprocess Extension — Observe Exporter

Exports AGH events to OpenTelemetry.

### Manifest (`extension.toml`)

```toml
[extension]
name = "otel-exporter"
version = "1.0.0"
description = "Export AGH events to OpenTelemetry"
type = "subprocess"
min_agh_version = "0.5.0"

[capabilities]
provides = ["observe.exporter"]

[actions]
requires = ["observe/events", "observe/health", "sessions/list"]

[subprocess]
command = "./bin/agh-ext-otel"
health_check_interval = "15s"

[subprocess.env]
OTEL_ENDPOINT = "{{env:OTEL_ENDPOINT}}"
OTEL_SERVICE_NAME = "agh"

[security]
capabilities = ["observe.read", "session.read"]
```

### Extension Code (Go)

```go
package main

import (
    "context"
    "time"

    agh "github.com/anthropics/agh/sdk/go" // placeholder — final module path TBD
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
)

func main() {
    ext := agh.NewExtension(agh.ExtensionConfig{
        Name:         "otel-exporter",
        Version:      "1.0.0",
        Capabilities: []string{"observe.exporter"},
    })

    exporter := initOTelExporter(os.Getenv("OTEL_ENDPOINT"))

    // Periodic poll for new events via Host API
    ext.OnReady(func(ctx context.Context, host *agh.HostAPI) {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        var lastSeen time.Time

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                events, _ := host.Observe.Events(ctx, agh.EventsQuery{
                    Since: &lastSeen,
                    Limit: 100,
                })
                for _, ev := range events {
                    exporter.Export(ctx, toOTelSpan(ev))
                    lastSeen = ev.Timestamp
                }
            }
        }
    })

    ext.Handle("health_check", func(ctx context.Context, _ any) (*agh.HealthResult, error) {
        return &agh.HealthResult{Healthy: exporter.IsConnected()}, nil
    })

    ext.Start()
}
```

---

## 6. Extension Bundling Resources

An extension that bundles skills, agent definitions, and hook declarations together — like a Claude Code plugin.

### Directory Structure

```
my-devops-pack/
    extension.toml
    skills/
        k8s-troubleshoot.md       # Skill: Kubernetes debugging
        terraform-review.md       # Skill: Terraform plan review
    agents/
        devops-agent.md           # Agent definition with custom system prompt
    bin/
        agh-ext-devops            # Subprocess binary (optional)
```

### Manifest (`extension.toml`)

```toml
[extension]
name = "devops-pack"
version = "1.0.0"
description = "DevOps skills, agent, and incident response hooks"
type = "subprocess"
min_agh_version = "0.5.0"

[resources]
skills = ["skills/"]
agents = ["agents/"]

[[resources.hooks]]
name = "incident-notifier"
event = "session.post_stop"
mode = "async"
executor.kind = "subprocess"
executor.command = "./bin/agh-ext-devops"
executor.args = ["--hook", "incident-notify"]

[[resources.hooks]]
name = "cost-guard"
event = "session.pre_create"
mode = "sync"
executor.kind = "subprocess"
executor.command = "./bin/agh-ext-devops"
executor.args = ["--hook", "cost-check"]

[resources.mcp_servers]
[resources.mcp_servers.kubectl]
command = "mcp-kubectl"
args = ["--context", "production"]

[capabilities]
provides = ["prompt.provider"]

[actions]
requires = ["sessions/list", "sessions/events", "observe/health"]

[subprocess]
command = "./bin/agh-ext-devops"
args = ["serve"]

[security]
capabilities = ["session.read", "observe.read"]
```

### What happens on `agh extension install ./my-devops-pack/`

```
1. DISCOVER  → Found extension.toml in ./my-devops-pack/
2. PARSE     → Manifest parsed: name=devops-pack, type=subprocess
3. VALIDATE  → Version check OK, checksum verified, capabilities valid
4. REGISTER  → 
   ├── Skills: k8s-troubleshoot.md, terraform-review.md → skills.Registry
   ├── Agents: devops-agent.md → config.AgentDef resolution
   ├── Hooks: incident-notifier, cost-guard → hooks.DeclarationProvider
   └── MCP: kubectl server → MCPResolver
5. INITIALIZE → Launch ./bin/agh-ext-devops serve → handshake
6. ACTIVATE   → Extension live. Hooks dispatching. Host API available.
```

---

## 7. Host API Usage Patterns

### Pattern: Channel Adapter (extension creates sessions from external messages)

```typescript
import { Extension, HostAPI } from '@agh/extension-sdk';
import { SlackClient } from './slack';

const ext = new Extension({
    name: 'slack-adapter',
    version: '0.1.0',
    capabilities: { provides: [] },
    actions: { requires: ['sessions/create', 'sessions/prompt', 'sessions/stop'] },
});

const slack = new SlackClient(process.env.SLACK_TOKEN!);

ext.onReady(async (host: HostAPI) => {
    // Listen for Slack messages
    slack.onMessage(async (msg) => {
        // Create a session per thread
        const threadKey = `slack-${msg.channel}-${msg.thread_ts}`;
        let sessionId = activeThreads.get(threadKey);

        if (!sessionId) {
            // Create new AGH session
            const result = await host.sessions.create({
                agent: 'default',
                workspace: '/path/to/project',
            });
            sessionId = result.session_id;
            activeThreads.set(threadKey, sessionId);
        }

        // Send the Slack message as a prompt
        await host.sessions.prompt({
            session_id: sessionId,
            message: msg.text,
        });

        // Stream events back to Slack
        const events = await host.sessions.events({
            session_id: sessionId,
            limit: 50,
        });
        for (const ev of events) {
            if (ev.type === 'message' && ev.data.role === 'assistant') {
                await slack.reply(msg.channel, msg.thread_ts, ev.data.text);
            }
        }
    });
});

ext.start();
```

### Pattern: Scheduled Task (extension creates sessions on a timer)

```go
ext.OnReady(func(host *agh.HostAPI) {
    // Every day at 9am, run a standup summary
    scheduler.Every("0 9 * * *", func() {
        result, err := host.Sessions.Create(ctx, agh.CreateParams{
            Agent:  "standup-summarizer",
            Prompt: "Summarize yesterday's git activity and open PRs",
        })
        if err != nil {
            log.Printf("standup session failed: %v", err)
            return
        }
        log.Printf("standup session created: %s", result.SessionID)
    })
})
```

### Pattern: Memory Enrichment (extension reads sessions to enrich memory)

```go
ext.OnReady(func(host *agh.HostAPI) {
    // Periodically scan completed sessions and extract learnings
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            sessions, _ := host.Sessions.List(ctx)
            for _, s := range sessions {
                if s.State != "stopped" { continue }

                events, _ := host.Sessions.Events(ctx, agh.EventsQuery{
                    SessionID: s.ID,
                    Type:      "tool_call",
                })

                insights := extractInsights(events)
                for _, insight := range insights {
                    host.Memory.Store(ctx, agh.StoreParams{
                        Key:     fmt.Sprintf("learning-%s-%d", s.ID, i),
                        Content: insight,
                        Tags:    []string{"auto-extracted", "session-learning"},
                    })
                }
            }
        }
    }()
})
```

---

## 8. CLI Interaction

```bash
# List installed extensions
$ agh extension list
NAME              VERSION  TYPE        STATE    TIER  CAPABILITIES
secret-guard      0.1.0    subprocess  active   content.validate
pgvector-memory   0.2.0    subprocess  active   memory.backend
otel-exporter     1.0.0    subprocess  active   observe.exporter
devops-pack       1.0.0    subprocess  active   prompt.provider

# Install from local path
$ agh extension install ./my-extension/
✓ Manifest parsed: my-extension v0.1.0 (subprocess)
✓ Checksum verified
✓ Resources registered: 2 skills, 1 agent, 3 hooks
✓ Extension installed

# Install from git URL
$ agh extension install github.com/user/agh-ext-pgvector@v0.2.0
✓ Cloned and verified
✓ Extension installed

# Disable without uninstalling
$ agh extension disable pgvector-memory
✓ pgvector-memory disabled (subprocess stopped)

# Re-enable
$ agh extension enable pgvector-memory
✓ pgvector-memory enabled (subprocess started, handshake OK)

# Check extension health
$ agh extension status pgvector-memory
Name:         pgvector-memory
Version:      0.2.0
Type:         subprocess
State:        active
PID:          42891
Uptime:       2h 15m
Health:       healthy (last check 12s ago)
Capabilities: memory.backend
Actions:      sessions/list, memory/store, memory/recall
Resources:    2 skills
```

---

## 9. Testing Extensions

### Unit Testing (TypeScript)

```typescript
import { TestHarness } from '@agh/extension-sdk/testing';
import { describe, it, expect } from 'vitest';

describe('pgvector-memory', () => {
    const harness = new TestHarness();

    // Mock Host API responses
    harness.mockHostAPI('sessions/list', () => [
        { id: 'sess-1', name: 'test', agent: 'claude', state: 'active' },
    ]);

    it('stores and recalls memory', async () => {
        const ext = await harness.loadExtension('./dist/index.js');

        // Simulate AGH calling memory/store
        const storeResult = await harness.call('memory/store', {
            key: 'test-key',
            content: 'The deploy script is at /scripts/deploy.sh',
            tags: ['project-knowledge'],
        });
        expect(storeResult).toEqual({});

        // Simulate AGH calling memory/recall
        const recallResult = await harness.call('memory/recall', {
            query: 'where is the deploy script?',
            limit: 5,
        });
        expect(recallResult.entries).toHaveLength(1);
        expect(recallResult.entries[0].content).toContain('deploy.sh');
    });

    it('rejects unauthorized Host API calls', async () => {
        const ext = await harness.loadExtension('./dist/index.js', {
            capabilities: ['memory.read'], // no memory.write
        });

        // Extension tries to call memory/store via Host API
        await expect(
            harness.simulateHostAPICall('memory/store', { key: 'x', content: 'y' })
        ).rejects.toThrow('capability_denied: memory.write');
    });
});
```

### Integration Testing (Go)

```go
func TestSubprocessExtensionEndToEnd(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()

    // Build and install test extension
    extDir := filepath.Join(dir, "test-ext")
    installTestExtension(t, extDir, TestExtManifest{
        Name: "secret-guard",
        Capabilities: []string{"content.validate"},
    })

    // Create extension manager and start
    registry := extension.NewRegistry(globalDB)
    mgr := extension.NewManager(registry, extension.WithLogger(testLogger))
    require.NoError(t, mgr.Start(t.Context()))
    defer mgr.Stop(t.Context())

    // Verify extension loaded and handshake completed
    info, err := mgr.Get("secret-guard")
    require.NoError(t, err)
    assert.Equal(t, "active", info.State)

    // Dispatch a hook via the existing hook system
    payload := hooks.InputPreSubmitPayload{
        Message: "my key is sk-abc123",
    }
    result, err := hookDispatcher.DispatchInputPreSubmit(t.Context(), payload)
    require.NoError(t, err)

    // Verify the hook blocked the message
    assert.Contains(t, result.DenyReason, "secret")
}
```

---

## 10. JSON-RPC Protocol Example (Raw)

What the bidirectional communication looks like on the wire between AGH and a subprocess extension. This matches the normative protocol spec in `_protocol.md`.

```
── AGH → Extension (initialize handshake) ──────────────────────
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
  "protocol_version":"1",
  "supported_protocol_versions":["1"],
  "agh_version":"0.5.0",
  "extension":{"name":"pgvector-memory","version":"0.2.0","source_tier":"user"},
  "capabilities":{
    "provides":["memory.backend"],
    "granted_actions":["sessions/list","sessions/events"],
    "granted_security":["memory.read","memory.write","session.read"]
  },
  "methods":{
    "daemon_requests":["execute_hook","health_check","shutdown"],
    "extension_services":["memory/store","memory/recall","memory/forget"]
  },
  "runtime":{
    "health_check_interval_ms":30000,
    "health_check_timeout_ms":5000,
    "shutdown_timeout_ms":10000,
    "default_hook_timeout_ms":5000
  }
}}

── Extension → AGH (initialize response) ──────────────────────
{"jsonrpc":"2.0","id":1,"result":{
  "protocol_version":"1",
  "extension_info":{"name":"pgvector-memory","version":"0.2.0","sdk_name":"@agh/extension-sdk","sdk_version":"0.1.0"},
  "accepted_capabilities":{
    "provides":["memory.backend"],
    "actions":["sessions/list","sessions/events"],
    "security":["memory.read","memory.write","session.read"]
  },
  "implemented_methods":["memory/store","memory/recall","memory/forget","health_check","shutdown"],
  "supported_hook_events":[],
  "supports":{"health_check":true,"provide_tools":false}
}}

── AGH → Extension (daemon calls memory/store) ────────────────
{"jsonrpc":"2.0","id":2,"method":"memory/store","params":{
  "key":"user-pref-timezone",
  "content":"User prefers UTC-3 (São Paulo)",
  "scope":"workspace",
  "tags":["user-preference"]
}}

── Extension → AGH (store response) ───────────────────────────
{"jsonrpc":"2.0","id":2,"result":{}}

── Extension → AGH (extension calls Host API) ─────────────────
{"jsonrpc":"2.0","id":100,"method":"sessions/list","params":{}}

── AGH → Extension (Host API response) ────────────────────────
{"jsonrpc":"2.0","id":100,"result":[
  {"id":"sess-abc","name":"debug-session","agent":"claude","state":"active"}
]}

── AGH → Extension (health check) ─────────────────────────────
{"jsonrpc":"2.0","id":3,"method":"health_check","params":{}}

── Extension → AGH (healthy) ──────────────────────────────────
{"jsonrpc":"2.0","id":3,"result":{"healthy":true,"message":"","details":{}}}

── AGH → Extension (shutdown) ─────────────────────────────────
{"jsonrpc":"2.0","id":4,"method":"shutdown","params":{
  "reason":"daemon_shutdown",
  "deadline_ms":10000
}}

── Extension → AGH (ack and exit) ─────────────────────────────
{"jsonrpc":"2.0","id":4,"result":{"acknowledged":true}}
```
