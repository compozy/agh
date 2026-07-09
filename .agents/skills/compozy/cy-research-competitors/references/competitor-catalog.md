# Competitor Catalog

Reference repos available under `.resources/` (or `~/dev/knowledge/<name>` for KB versions). When the user asks for "competitor research" without a list, default to the matrix below filtered by domain.

## Active reference systems

| Name | Path | Domain | Best for |
|------|------|--------|----------|
| `claude-code` | `.resources/claude-code/` | AI harness | Skill plugins, hook taxonomy (25+ events), AutoDream memory consolidation, JSONL transcripts, permission engine, token-budget cascade |
| `codex-cli` | `.resources/codex/` (via `~/.codex/`) | AI harness | Apply-patch tool, structured-prompt sessions, gpt-5.4 reasoning effort modes |
| `hermes` | `.resources/hermes/` | Long-running agents | ProcessRegistry, lifecycle hardening, MCP OAuth+PKCE, durable scheduler, memory health, ACP adapter |
| `openclaw` | `.resources/openclaw/` | Daemon-based AI runtime | CLI/daemon lifecycle, JSONL session store |
| `openfang` | `.resources/openfang/` | Agent OS reference | 14-crate workspace, kernel struct, daemon-as-composition-root, signature verification on skills |
| `goclaw` | `.resources/goclaw/` | Go-native harness | Typed function hooks, idiomatic Go AI harness patterns |
| `pi-mono` | `.resources/pi-mono/` | Pi monorepo | session_start/before_agent_start hook taxonomy, prompt_builder.py |
| `multica` | `.resources/multica/` | Multi-agent harness | Multi-agent patterns, harness coexistence |
| `paperclip` | `.resources/paperclip/` | Workflow / scheduler | Paper artboard system, scheduler, automation flows |
| `agent-networks` | KB-only | Protocol research | A2A, ANP, MCP, ACP, NLIP, AGNTCY, AP2, x402 |
| `ai-harness` | KB-only | Harness landscape | Context engineering, token budget compaction |
| `ai-memory` | KB-only | Memory research | CoALA, Complementary Learning Systems, episodic/semantic/procedural memory |

## Default sets by topic

- **Memory / consolidation:** `claude-code` + `ai-memory` + `pi-mono`.
- **Skills / lifecycle hooks:** `claude-code` + `hermes` + `openfang` + `pi-mono`.
- **Agent network / protocol:** `agent-networks` + `claude-code` + `openfang`.
- **Daemon / runtime / lifecycle:** `openfang` + `openclaw` + `hermes`.
- **Long-running sessions / supervision:** `hermes` + `openfang`.
- **Workflow / scheduler:** `paperclip` + `goclaw` + `hermes`.
- **AI SDK / streaming / harness:** `claude-code` + `codex-cli` + `ai-harness`.

When the user is in a brand-new domain, default to a 4-system mix from the columns above.
