# Multi-Agent Orchestration Patterns: Analysis and Recommendations for AGH

- **Source:** [From Chaos to Choreography: Multi-Agent Orchestration Patterns That Actually Work](https://www.youtube.com/watch?v=2czYyrTzILg) -- Sandipan Bhaumik (Databricks), AI Engineer Conference, 2026-04-08
- **Created:** 2026-04-09
- **Scope:** Where distributed systems patterns apply in AGH's three-phase architecture

---

## 1. Context

Sandipan Bhaumik's talk distills 18 years of distributed systems engineering (NHS, Tier 1 banks, AWS, Databricks) into six patterns for multi-agent coordination. His core thesis: **multi-agent systems are distributed systems problems, not AI problems.** Teams fail because they treat adding agents like adding features, when it's actually building a coordination problem with exponential complexity.

This document maps each pattern onto AGH's architecture -- the daemon internals (Phase 1-2), the AGH Network protocol (Phase 3), and the gaps between them. Some patterns validate existing design decisions. Others expose work that must happen in the daemon layer before the network protocol matters.

---

## 2. Summary of Patterns from the Talk

| #   | Pattern                   | One-liner                                                                        |
| --- | ------------------------- | -------------------------------------------------------------------------------- |
| 1   | Choreography              | Agents coordinate through events on a message bus. Decentralized, autonomous.    |
| 2   | Orchestration             | Central coordinator manages workflow DAG. Agents never talk to each other.       |
| 3   | Immutable state snapshots | Versioned, append-only state handoff. No shared mutable state between agents.    |
| 4   | Data contracts            | Input/output schema validation at every agent boundary. Reject bad data early.   |
| 5   | Circuit breaker           | Fail fast after N consecutive failures. Prevent cascading failure across agents. |
| 6   | Compensation / Saga       | Every agent has execute + compensate. Orchestrator walks backward on failure.    |

His decision framework: **simple workflow + high autonomy = choreography**; **complex workflow + low autonomy tolerance = orchestration**; **complex + autonomous = hybrid with saga patterns**.

---

## 3. Pattern-by-Pattern Analysis

### 3.1 Choreography vs Orchestration

**What the talk says:** These are the two fundamental coordination patterns. Choreography is event-driven pub/sub where agents react to events autonomously. Orchestration is centralized control where a coordinator calls agents in sequence/parallel and manages all state. Most teams pick one instinctively and regret it. The answer is usually "both, at different layers."

**Where this lives in AGH today:**

AGH already operates at **two distinct coordination layers** that map directly to this pattern split:

**Layer A -- Within the daemon (single-machine, Phase 1-2):**
The session Manager is an **orchestrator**. It owns the lifecycle, calls the ACP driver, manages state transitions, records events, handles shutdown. Agents (subprocesses) never talk to each other. All coordination flows through the Manager. This is exactly the orchestration pattern from the talk.

The coordinator mode studied from Claude Code (see `docs/ideas/from-claude-code/analysis_multi_agent.md`) reinforces this: the coordinator loses direct tool access and operates exclusively through worker delegation. Workers run asynchronously. Results flow back through the coordinator.

**Layer B -- Across daemons (network, Phase 3):**
The AGH Network protocol (RFC 003/004) supports **both patterns** through its message kinds:

- **Choreography:** `say` broadcasts to spaces via NATS pub/sub. Any peer subscribed to the space reacts autonomously. `greet` provides presence heartbeats. This is pure event-driven choreography.
- **Orchestration:** `direct` messages with `interaction_id` enable targeted request/response flows. `receipt` provides acknowledgment. `trace` reports progress through the lifecycle. A coordinator peer could use `direct` to manage a workflow DAG across remote agents.

**Assessment:** The existing design is sound. The protocol correctly supports both patterns at the wire level. The daemon correctly uses orchestration internally. No changes needed to the RFCs.

**Recommendation:** When implementing the network layer (Phase 3), the daemon should expose **both coordination modes** to users:

1. **Space-level choreography** -- agents join a space, listen for broadcasts, react autonomously. Good for monitoring, alerting, recipe sharing, peer discovery.
2. **Interaction-level orchestration** -- one agent opens `direct` interactions with specific peers, tracks progress via `trace`, handles `receipt` acknowledgments. Good for structured workflows (code review chains, deployment pipelines, multi-step analysis).

The daemon's internal `session.Manager` pattern (centralized orchestrator with clean state machine) should serve as the template for interaction-level orchestration over the network.

---

### 3.2 Immutable State Snapshots

**What the talk says:** The #1 cause of multi-agent bugs is shared mutable state. His credit decisioning system failed because two agents read from a stale cache simultaneously. The fix: immutable state snapshots with versioning. Each agent produces a new sealed version. No updates, only appends. State evolution becomes a replayable log. When something fails, roll back to version N-1. When debugging, binary search through state history.

**Where this lives in AGH today:**

AGH's event storage already follows this pattern partially:

- **SessionDB** (`internal/store/sessiondb/`) uses **append-only event recording**. Events are INSERT-only with auto-incrementing sequence numbers. No updates. Each event is immutable once written. This is the right foundation.
- **Token usage** is stored per-turn with UPSERT aggregation in GlobalDB -- this is the one place where the pattern breaks (aggregate updates rather than append-only).
- **Session metadata** (`meta.json`) is mutable -- it gets rewritten on state changes. This is acceptable for session-level state but wouldn't work for inter-agent state handoff.

**Where the gap is:**

AGH has no concept of **inter-agent state handoff**. Today, sessions are isolated. Agent A's session and Agent B's session share nothing except:

1. The global memory store (file-based, not versioned)
2. The filesystem (agents can read/write files in the workspace)
3. The scratchpad directory (from Claude Code patterns, gated behind feature flag)

When the daemon supports multi-agent workflows (Phase 2 coordinator mode, Phase 3 network), this becomes critical. If Agent A produces analysis and Agent B consumes it, the handoff needs:

- Immutability guarantee: Agent B receives a snapshot, not a live reference
- Version tracking: Which version of Agent A's output did Agent B receive?
- Lineage: If Agent C fails, what state versions led to the failure?

**Recommendation for daemon layer (Phase 2):**

The interaction model should use **versioned handoff state** when passing work between agents within the daemon. This doesn't need to be a new database -- it can build on the existing event recording pattern:

```
Turn 1: Agent A produces → event recorded with content hash
Turn 2: Coordinator extracts output → creates handoff envelope with version + hash
Turn 3: Agent B receives handoff → validates hash → processes → produces new version
```

The `observe` package already records events with sequence numbers, session IDs, and turn IDs. A handoff event type (e.g., `EventTypeHandoff`) that records: source agent, target agent, content hash, and version number would give us the immutable state snapshot pattern with minimal new infrastructure.

**Recommendation for network layer (Phase 3):**

The AGH Network `ext` field is the right place for state version metadata on cross-daemon handoffs:

```json
{
  "ext": {
    "agh.handoff_version": 3,
    "agh.handoff_digest": "sha256:abc123...",
    "agh.handoff_source": "agent-a"
  }
}
```

This keeps the wire protocol clean (state versioning is an application concern, not a protocol concern) while enabling implementations to track lineage. The `trace` message kind already supports `artifact_refs` in its body, which could reference specific state versions.

---

### 3.3 Data Contracts

**What the talk says:** Agents need contracts. Agent A can't just throw arbitrary data at Agent B and hope it works. Every handoff boundary should have schema validation. If the research agent outputs low-confidence data, the analysis agent rejects it at the boundary, not three agents downstream.

**Where this lives in AGH today:**

AGH has several proto-contract mechanisms:

1. **ACP protocol** -- The ACP client defines typed event structures (`AgentEvent`, `PromptRequest`, `ACPCaps`). This is a contract between the daemon and agents. Well-defined, enforced at the code level.

2. **Recipe artifact** -- RFC 003 defines `recipe` with explicit `inputs`, `outputs`, and `requirements` fields. This is a first-class data contract for portable procedures. Example from the RFC:

   ```json
   {
     "inputs": ["failing test output", "repository or package path"],
     "outputs": ["patch summary", "verification notes"],
     "requirements": ["Go toolchain", "workspace write access"]
   }
   ```

3. **Peer Card capabilities** -- `capabilities` in the Peer Card (e.g., `chat.translate`, `workspace.patch.apply`) are advisory signals about what a peer can do. Not a contract, but a pre-handoff compatibility check.

4. **Skill metadata** -- Skills declare their name, description, and version in YAML frontmatter. MCP servers declare their command and environment requirements. This is contract-adjacent.

**Where the gap is:**

None of these enforce **runtime data contracts between agents**. When Agent A hands work to Agent B within the daemon, there's no schema validation at the boundary. The coordinator pattern from Claude Code relies on the coordinator to craft specific specs with file paths and line numbers -- this is a soft contract enforced by prompt engineering, not by the system.

**Recommendation for daemon layer (Phase 2):**

For within-daemon multi-agent workflows, data contracts should be **lightweight and prompt-driven**, not schema-enforced. Reasons:

- Agent outputs are natural language + tool results, not structured data
- The coordinator mode already handles synthesis and spec-crafting
- Adding JSON Schema validation between agents would be over-engineering for the current use case

The right pattern is: the coordinator validates that worker output meets expectations before passing it to the next worker. This is what the Claude Code coordinator mode already does (the "synthesis" phase where the coordinator reads findings and crafts specific specs).

**Recommendation for network layer (Phase 3):**

For cross-daemon handoffs, data contracts matter more because:

- You can't trust remote peers to produce well-formed output
- Network latency makes round-trip corrections expensive
- Different daemon versions may have different capabilities

The `recipe` artifact already has the right shape for this. A recipe's `inputs`/`outputs`/`requirements` are the contract. The `Peer Card` capabilities are the pre-flight check. The `receipt` with `reason_code: "unsupported"` is the rejection mechanism.

No RFC changes needed. The existing protocol primitives support this pattern.

---

### 3.4 Circuit Breaker

**What the talk says:** When Agent B fails repeatedly (5 consecutive failures), stop calling it. The circuit breaker opens, and you fail fast instead of waiting for timeouts. After a cooldown, test with one request. If it succeeds, close the circuit. This prevents cascading failures -- one agent going down doesn't bring the entire workflow down.

**Where this lives in AGH today:**

AGH has no circuit breaker pattern. The current architecture doesn't need one because:

- Sessions are isolated -- one agent crash doesn't affect other sessions
- The ACP driver has a graceful stop sequence (cancel -> SIGTERM -> wait -> SIGKILL)
- The daemon monitors process exits via background goroutines

**Where the gap is:**

When AGH supports multi-agent workflows, circuit breakers become essential at two levels:

1. **Daemon-internal (Phase 2):** If the coordinator spawns 5 workers and Worker C consistently crashes, the coordinator should stop sending work to Worker C rather than blocking the entire workflow.

2. **Network-level (Phase 3):** If Peer B stops responding to `direct` messages, Peer A should stop sending them after N failures. The protocol already has the building blocks: `receipt(rejected, reason_code: "busy")` signals overload, and `expires_at` provides timeout semantics.

**Recommendation for daemon layer (Phase 2):**

Implement a simple circuit breaker in the session Manager for multi-agent coordinator mode:

```go
type CircuitBreaker struct {
    mu           sync.Mutex
    state        CircuitState  // closed, open, half-open
    failures     int
    threshold    int           // e.g., 5
    resetTimeout time.Duration // e.g., 60s
    lastFailure  time.Time
}
```

This wraps agent driver calls. When an agent fails N times consecutively:

1. Circuit opens -- skip that agent, use fallback (different agent, cached result, or skip)
2. After timeout, circuit goes half-open -- try one call
3. If it succeeds, close. If it fails, re-open.

The circuit breaker state should be observable: emit events via the Notifier so the web UI can show agent health status.

**Recommendation for network layer (Phase 3):**

Circuit breakers at the network level are **implementation concerns, not protocol concerns**. The RFC correctly doesn't define them. But the implementation guide (or a future "AGH Network Best Practices" document) should recommend:

- Track `receipt(rejected)` counts per peer
- Use `expires_at` on outgoing messages to bound wait time
- If a peer stops sending `greet` heartbeats (2x the 30s interval = 60s), mark it offline and stop routing to it
- The periodic `greet` mechanism in RFC 003 Section 10.5 already functions as implicit health checking

---

### 3.5 Compensation / Saga Pattern

**What the talk says:** Every agent has two methods: `execute` and `compensate`. If Agent C fails mid-workflow, the orchestrator walks backward: Agent B compensates (undoes its work), then Agent A compensates. You're back to the initial state. No partial transactions. No stuck workflows.

**Where this lives in AGH today:**

The RFC explicitly excludes this. Section 7.3 of RFC 003:

> [Lifecycle states] do not imply: workflow graph semantics, orchestration plans, retries as protocol state, **compensation logic**

And this is the right call. Compensation is an application-level concern.

**Where the gap is:**

When AGH runs multi-agent workflows that modify the filesystem (code generation, refactoring, deployment), partial failures leave the workspace in an inconsistent state. Consider:

1. Agent A generates a new Go file
2. Agent B modifies an existing file to import it
3. Agent C runs tests -- they fail
4. Now the workspace has a half-finished change

Without compensation, the user manually cleans up. With compensation:

1. Agent B reverts its import modification
2. Agent A deletes its generated file
3. Workspace is back to pre-workflow state

**Recommendation for daemon layer (Phase 2):**

For within-daemon multi-agent workflows, compensation should use **git as the compensation mechanism**:

1. Before starting a multi-agent workflow, create a git checkpoint (stash or lightweight branch)
2. Each agent's work is tracked as a set of file changes
3. If the workflow fails, the orchestrator can `git checkout` back to the checkpoint
4. If the workflow succeeds, the changes remain in the working tree

This is simpler and more reliable than implementing custom `compensate()` methods per agent. Git is already the compensation engine -- AGH just needs to formalize checkpoint/rollback around multi-agent workflows.

The `session.Manager` should support:

```go
type WorkflowOpts struct {
    GitCheckpoint bool          // Create checkpoint before workflow
    Compensate    CompensateMode // none, git-rollback, custom
}
```

**Recommendation for network layer (Phase 3):**

Cross-daemon compensation is genuinely hard and should be deferred. The AGH Network interaction lifecycle already has the right terminal states (`completed`, `failed`, `canceled`), and the `receipt(canceled)` mechanism supports initiator-side withdrawal. But true distributed saga with compensating transactions across daemons is a Phase 4+ concern.

For now, the protocol provides the signaling primitives. Application-level saga coordination can be built on top of `direct` + `trace` + `receipt` without protocol changes.

---

### 3.6 Observability

**What the talk says:** "Without bulletproof observability, choreography will destroy you." His strongest warning. He emphasizes per-agent tracing, state evolution replay, and the ability to binary search through state history to find where things went wrong.

**Where this lives in AGH today:**

AGH has strong observability foundations:

1. **Event recording** -- Every prompt turn is recorded as a sequence of immutable events in SessionDB. Events include: user messages, agent messages, thoughts, tool calls, tool results, permissions, usage stats.

2. **Correlation primitives** -- Events have `session_id`, `turn_id`, `sequence` numbers. The transcript system can reconstruct full conversation history.

3. **Global observation** -- The Observer writes event summaries, token stats, and permission logs to GlobalDB for cross-session visibility.

4. **AGH Network correlation** -- RFC 003/004 define `trace_id`, `causation_id`, `reply_to` for distributed correlation. The `trace` message kind reports progress through the interaction lifecycle.

5. **Health metrics** -- Uptime, active sessions, active agents, database sizes, version info.

**Where the gap is:**

Current observability is **per-session, not per-workflow**. When a multi-agent workflow spans multiple sessions (coordinator + N workers), there's no unified view of:

- Which workers were spawned for which coordinator request
- The causal chain across sessions (coordinator prompt -> worker A -> worker B -> result)
- Aggregate token usage / latency per workflow (not per session)
- State evolution across the workflow (what each agent received and produced)

**Recommendation for daemon layer (Phase 2):**

Introduce a **workflow correlation ID** that spans multiple sessions:

1. When the coordinator spawns workers, it generates a `workflow_id`
2. Each worker session records the `workflow_id` in its metadata
3. The Observer can query events grouped by `workflow_id` across sessions
4. The web UI can show a workflow timeline view: coordinator -> worker A (parallel) -> worker B -> result

This maps directly to the talk's emphasis on "one dashboard showing the entire system state." The existing `trace_id` field in the AGH Network envelope serves the same purpose at the network layer -- extend the concept into the daemon's internal event model.

The `store.SessionEvent` table could add an optional `workflow_id` column, or it could use the existing event content JSON (canonical payload) to carry `workflow_id` without schema changes.

**Recommendation for network layer (Phase 3):**

The RFC's correlation primitives (`trace_id`, `causation_id`, `reply_to`) are sufficient for cross-daemon observability. The `trace` message kind with its state values (`working`, `needs_input`, `completed`, `failed`, `canceled`) gives receivers enough to track interaction progress.

One addition to consider for a future RFC revision: a standard `ext` convention for workflow-level metadata:

```json
{
  "ext": {
    "agh.workflow_id": "wf_abc123",
    "agh.workflow_step": 3,
    "agh.workflow_total_steps": 5
  }
}
```

This is RECOMMENDED convention, not normative -- exactly the v0 extension model.

---

## 4. Synthesis: What Needs to Happen and Where

### 4.1 Nothing changes in the RFCs

The AGH Network protocol (RFC 003/004) operates at the **wire protocol layer**. All six patterns from the talk are **application-level concerns** that belong in the daemon implementation, not the protocol specification. The RFCs correctly exclude:

- Workflow engine semantics
- Orchestration plans
- Compensation logic
- Retry as protocol state
- State management backends

The protocol provides the right primitives for implementations to build these patterns on top:

- Choreography via `say` broadcast + NATS pub/sub
- Orchestration via `direct` + `interaction_id` + `trace` lifecycle
- Acknowledgment via `receipt` with status and reason codes
- Correlation via `trace_id` + `causation_id` + `reply_to`
- Capability signaling via Peer Card
- Data contracts via `recipe` inputs/outputs/requirements
- Overload signaling via `reason_code: "busy"`
- Presence/health via periodic `greet` heartbeats

### 4.2 Daemon-internal work (Phase 2 priorities)

These patterns must be implemented in the daemon before the network protocol matters, because **single-machine multi-agent coordination is the prerequisite for networked coordination**.

| Priority | Pattern                    | Where in codebase                                                                       | Effort  |
| -------- | -------------------------- | --------------------------------------------------------------------------------------- | ------- |
| P0       | Coordinator mode           | `internal/session/` -- new workflow orchestration on top of existing Manager            | Medium  |
| P1       | Workflow correlation       | `internal/observe/` + `internal/store/` -- workflow_id spanning sessions                | Small   |
| P1       | Circuit breaker            | `internal/session/` -- wrapping agent driver calls                                      | Small   |
| P2       | Immutable handoff state    | `internal/session/` + `internal/store/sessiondb/` -- new event type for handoffs        | Medium  |
| P2       | Git-based compensation     | `internal/session/` -- checkpoint/rollback around multi-agent workflows                 | Medium  |
| P3       | Inter-agent data contracts | Soft enforcement via coordinator prompt engineering, not system-level schema validation | Minimal |

### 4.3 Network implementation work (Phase 3)

When implementing AGH Network, these patterns should be documented as **recommended practices** in an implementation guide:

1. **Use `greet` heartbeats as implicit health monitoring** -- if a peer misses 2x the interval, treat it as offline (circuit breaker)
2. **Use `ext` for workflow metadata** -- `agh.workflow_id`, `agh.handoff_version`, `agh.handoff_digest`
3. **Use `recipe` inputs/outputs as data contracts** -- validate before execution
4. **Use `receipt(rejected, reason_code: "busy")` for backpressure** -- callers implement circuit breaker logic
5. **Use `trace` state transitions for observability** -- build workflow timelines from `trace` events
6. **Use `causation_id` chains for debugging** -- walk backward through message causation to find root cause

---

## 5. Connection to Existing AGH Work

### 5.1 Claude Code multi-agent analysis

The `docs/ideas/from-claude-code/analysis_multi_agent.md` document analyzed Claude Code's coordinator mode, fork mode, scratchpad, and team memory. The talk's patterns validate and extend that analysis:

- **Coordinator mode = Orchestration pattern** -- Claude Code's coordinator (loses tools, delegates to workers, synthesizes results) is exactly the orchestration pattern from the talk.
- **Scratchpad = Shared state (with caveats)** -- The scratchpad directory is shared mutable state, which the talk warns against. The mitigation is that the coordinator manages access order -- workers don't write to the same files concurrently because the coordinator sequences their work.
- **Task notification = Receipt pattern** -- Worker results arriving as `<task-notification>` XML is a form of receipt. The AGH Network `receipt` and `trace` messages formalize this.
- **Fork mode = Choreography at micro-scale** -- Agents autonomously deciding to parallelize (fork) is choreography within a single session. The anti-recursion guard (fork marker detection) prevents infinite choreography loops.

### 5.2 Skills system (RFC 002)

Skills with lifecycle hooks are a **contract mechanism**:

- `on_session_created` hooks inject context before first prompt -- this is the "data contract" for the agent's operating environment
- `on_session_stopped` hooks consolidate learnings -- this is the "compensation" for knowledge management (save what was learned, clean up temporary state)
- MCP server declarations in skill metadata are **capability contracts** -- the skill declares what tools it needs, the daemon provisions them

### 5.3 Agent definitions (RFC 001)

Agent-scoped skills and memory are **isolation guarantees**:

- Agent A's skills directory is exclusive -- Agent B can't access them
- Agent A's memory is scoped -- no cross-contamination
- This prevents the "shared mutable state" problem at the agent level

The `tools` field in agent definitions (e.g., `tools: [Read, Grep, Glob]`) is a **capability restriction** -- a form of contract that limits what the agent can do, reducing blast radius if it misbehaves.

---

## 6. The Key Insight

Bhaumik's talk reinforces one meta-pattern: **agent coordination is not special.** It's the same distributed systems engineering that has been solved for decades in databases, microservices, and message queues. The patterns (choreography, orchestration, immutable state, contracts, circuit breakers, sagas) are all borrowed from existing distributed systems practice.

AGH's architecture already reflects this insight:

- The daemon is an **operating system**, not an AI framework
- Sessions are **processes** with lifecycle management
- Events are **immutable logs** with sequence numbers
- The network protocol is a **wire format**, not a workflow engine
- NATS is the **transport**, not the coordination logic

The work ahead is not inventing new patterns -- it's applying proven distributed systems patterns to the specific problem of multi-agent coordination, at the right layer, with the right granularity. The daemon handles within-machine coordination (orchestration, state management, failure recovery). The protocol handles cross-machine communication (message format, routing, discovery, trust). Neither tries to do the other's job.

---

## 7. References

- **Talk:** Sandipan Bhaumik, "From Chaos to Choreography: Multi-Agent Orchestration Patterns That Actually Work," AI Engineer Conference, April 2026. [YouTube](https://www.youtube.com/watch?v=2czYyrTzILg). [Slides](https://drive.google.com/file/d/18LqVzhfVS3iULYuy2EshWoMLmQt3rdpT/view?usp=sharing).
- **Speaker:** Data & AI Tech Lead at Databricks. 18 years in distributed data systems (NHS, Tier 1 banks, AWS, Databricks). [LinkedIn](https://www.linkedin.com/in/sandipanbhaumik).
- **Internal references:**
  - `docs/rfcs/003_agh-network-v0.md` -- AGH Network v0 protocol
  - `docs/rfcs/004_agh-network-v1.md` -- AGH Network v1 trust + conformance
  - `docs/ideas/from-claude-code/analysis_multi_agent.md` -- Claude Code multi-agent patterns
  - `docs/rfcs/001_agent-md-with-skills-memory.md` -- Agent definitions with scoped skills/memory
  - `docs/rfcs/002_skills-system-final.md` -- Skills lifecycle and governance
  - `internal/session/manager.go` -- Session Manager (orchestration root)
  - `internal/observe/observer.go` -- Event observation and recording
  - `internal/store/sessiondb/session_db.go` -- Append-only session event store
- **Distributed systems references cited in the talk:**
  - Circuit breaker pattern (Michael Nygard, "Release It!")
  - Saga pattern (Hector Garcia-Molina, Kenneth Salem, 1987)
  - Event choreography (Chris Richardson, "Microservices Patterns")
  - Immutable infrastructure (Chad Fowler, 2013)
