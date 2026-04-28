# Complex Real-World Scenarios — AGH Network Feature

**Status:** behavior-first E2E backlog
**Owner:** runtime + web team
**Audience:** `real-scenario-qa`, `qa-report`, `qa-execution`, E2E authors, and release QA
**Last updated:** 2026-04-28

> This document is not a CLI-only or runtime-only checklist. The primary goal is to
> describe realistic startup/operator journeys where AGH Network is used through the
> product: daemon runtime, CLI, HTTP/UDS APIs, and the `web/` Network workspace.
> Low-level router, delivery, and audit assertions are readiness/regression checks,
> not a substitute for behavior-first E2E proof.

---

## 0. QA posture

AGH Network E2E should answer one product question:

> Can a real operator coordinate multiple startup roles through AGH, see the same
> network truth in the browser and CLI, and recover from realistic coordination
> mistakes without inspecting private internals?

Use the skill chain in this order:

1. `qa-report` turns one scenario below into a test charter with roles, data, expected
   outcome, disruption probes, and evidence requirements.
2. `real-scenario-qa` bootstraps an isolated startup lab with daemon, runtime home,
   provider home, Web proxy target, browser evidence directory, and realistic agents.
3. `qa-execution` runs the journey end to end through CLI/API/Web, captures evidence,
   files issues for any broken behavior, and reruns the journey after fixes.

Smoke checks, unit tests, `make verify`, route-render checks, and endpoint-only checks
are useful readiness gates. They do not complete a scenario by themselves.

---

## 1. Current product surfaces

These are the surfaces the scenarios can exercise today.

| Surface                | Testable behavior                                                                                                                                                                                                                                  |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Daemon/runtime         | Embedded network manager, local peer lifecycle, channels, direct/say messages, inbox queues, audit/message store projection, task ingress guards.                                                                                                  |
| CLI                    | `agh network status`, `agh network peers [channel]`, `agh network channels`, `agh network send --session --channel --kind --body`, `agh network inbox --session`.                                                                                  |
| HTTP/UDS API           | `/api/network/status`, `/api/network/peers`, `/api/network/peers/{peer_id}`, `/api/network/peers/{peer_id}/messages`, `/api/network/channels`, `/api/network/channels/{channel}`, `/api/network/channels/{channel}/messages`, `/api/network/send`. |
| Web                    | `web/` route `/network`, backed by `web/src/routes/_app/network.tsx` and `web/src/systems/network`, including status, channel list, channel detail, peer rooms, timelines, kind filters, create-channel dialog, and compose actions.               |
| Session/task adjacency | Network channels can be created from selected local agents; sessions join channels; tasks and runs can carry `network_channel` and validate against active channels.                                                                               |

Use `AGH_WEB_API_PROXY_TARGET` from the active bootstrap manifest/env whenever an
isolated daemon is not running on the default port.

---

## 2. Evidence requirements

Every scenario must collect overlapping evidence for at least one shared object:
the same channel, peer, message, session, task, or artifact must be visible through
more than one public surface.

Required evidence:

- Bootstrap block: manifest path, lab root, runtime home, provider home, daemon API,
  Web base URL, and QA output path.
- CLI outputs: JSON/TOON output for status, channels, peers, send, and inbox commands
  used in the scenario.
- API outputs: HTTP or UDS payloads for the same channel/message/peer observed by CLI.
- Browser evidence: `browser-use:browser` URL, DOM snapshot, and screenshot for the
  `/network` state that an operator must understand. Use `agent-browser` only after
  recording why browser-use was unavailable.
- Runtime artifacts: persisted task/run/channel/message IDs and generated startup
  artifacts when the scenario creates them.
- Disruption evidence: the wrong-channel, stale-view, failed-send, missing-handoff,
  blocked dependency, or invalid request that proves AGH reports actionable state.
- Final verification: `make verify` after code/doc changes, plus the replayed journey
  after any bug fix.

Provider-backed agents should be used when credentials and provider CLIs are reachable.
If they are not reachable, document the exact boundary and do not replace live-agent
proof with fake final confidence.

---

## 3. Startup scenario matrix

| Scenario                   | Startup outcome                                                              | Agents/roles                                      | CLI evidence                                                 | Web evidence                                                                    | API/runtime evidence                                     | Disruption probe                                  |
| -------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------- | -------------------------------------------------------- | ------------------------------------------------- |
| Launch room coordination   | Cross-functional launch team shares status and handoffs in one channel.      | founder, product, engineering, support, QA, comms | `status`, `channels`, `peers launch`, `send`, `inbox`        | Create/open `launch`, inspect timeline, peer rail, kind filters, compose result | `/api/network/channels/launch`, `/messages`, `/send`     | Wrong recipient or missing local peer for compose |
| Customer escalation room   | Support escalates a customer issue to engineering and confirms handoff.      | support, backend, founder, QA                     | `send --kind direct`, `inbox`, `peers support-escalation`    | Direct peer room shows handoff and receipt-like timeline                        | peer messages endpoint and task/run linked to channel    | Missed handoff or stale direct room               |
| Release triage             | Bug bash produces task/run evidence tied to a network channel.               | QA, frontend, backend, release lead               | channel list, task CLI/API where available, network messages | `/network` plus task/run Web route when in scope                                | task/run `network_channel`, channel messages             | Failed run or blocked dependency is visible       |
| Daily operating digest     | Operators reconstruct yesterday/today status from persisted network history. | founder, ops, product, engineering                | `channels`, `peers`, historical message query via API        | Browser shows understandable channel timeline and not protocol noise only       | message store rows through supported API                 | Presence noise hides useful history               |
| Hiring funnel coordination | Synthetic candidate review is coordinated without leaking unsafe data.       | recruiter, screener, founder, hiring manager      | `send`, `inbox`, channel/peer listing                        | Candidate review channel and peer lanes are readable                            | channel messages and task state with synthetic data only | Wrong channel assignment or stale task channel    |
| Capability handoff         | Operator chooses the right peer after inspecting advertised capabilities.    | researcher, coder, reviewer, operator             | `peers`, `send --kind direct`, `inbox`                       | Peer detail shows capability brief and directed room                            | whois/capability discovery through runtime/API           | Capability exists but no usable peer or channel   |

---

## 4. Scenario A — launch room coordination

**Operator story.** A small startup is preparing a public launch. The operator creates
a `launch` channel for founder, product, engineering, support, QA, and comms agents.
The team shares launch readiness, assigns follow-up, and the operator validates from
the browser that the room state is understandable before the launch window.

**Setup.**

- Bootstrap an isolated QA lab with daemon, Web, unique `AGH_HOME`, unique provider
  home, and `AGH_WEB_API_PROXY_TARGET`.
- Use configured real agents when possible. Minimum local roles: `founder`,
  `product`, `engineer`, `support`, `qa`, `comms`.
- Create the `launch` channel from the Web create-channel dialog when possible; if the
  UI cannot create it, create it through `POST /api/network/channels` and file the Web
  issue instead of skipping browser validation.

**Journey.**

1. Operator opens `/network`, confirms enabled status, and creates `launch` with the
   selected startup roles.
2. CLI confirms the same channel through `agh network channels` and the same peers
   through `agh network peers launch`.
3. Operator sends a broadcast through the browser compose control.
4. CLI sends a second `say` message using one local session ID:
   `agh network send --session <session_id> --channel launch --kind say --body '{"text":"Launch checklist ready","stage":"launch"}'`.
5. Browser reopens the `launch` room and verifies both messages, peer membership,
   message counts, and useful metadata.
6. API validates the same objects through `/api/network/channels/launch` and
   `/api/network/channels/launch/messages`.

**Assertions.**

- The channel exists once, with the intended purpose and selected local sessions.
- CLI, API, and Web agree on channel name, local peer count, session IDs when exposed,
  and message IDs for the created messages.
- Browser evidence proves the operator can tell who spoke, what happened, and which
  room is active.
- Presence heartbeats are inspectable when requested but do not drown the default
  operator timeline.
- A failed compose attempt with no local peer or an invalid room produces a visible
  error, not silent loss.

---

## 5. Scenario B — customer escalation room

**Operator story.** A support agent receives a customer-impacting incident. Support
needs to hand off to backend engineering, keep the founder informed, and confirm that
QA knows what to retest.

**Setup.**

- Create `support-escalation` with support, backend, QA, and founder agents.
- Seed a synthetic customer incident with non-sensitive data, for example
  `ACME-LOCAL-001` and a reproducible symptom.

**Journey.**

1. Support sends a channel summary from CLI with `kind: say`.
2. Support sends a direct handoff to backend with `kind: direct`, `--to <backend_peer>`,
   and an `interaction_id` for the escalation.
3. Backend replies, or, when live provider-backed agents are unavailable, the operator
   records the provider boundary and only validates public network state.
4. Browser opens `/network`, selects the backend peer room, and confirms the directed
   timeline is scoped to that peer while the channel room still contains the public
   escalation summary.
5. API reads `/api/network/peers/{peer_id}/messages` and
   `/api/network/channels/support-escalation/messages`.

**Assertions.**

- Direct messages remain scoped to the peer lane; broadcast messages remain in the
  channel lane.
- CLI `inbox` for the backend session shows queued inbound handoff when prompting is
  blocked or the session is unavailable.
- Browser makes the handoff understandable: support origin, backend target, channel,
  interaction ID if surfaced, and current queue/message state.
- If a message targets the wrong peer or stale peer, the failure is visible through
  CLI/API and the browser does not claim the handoff succeeded.

---

## 6. Scenario C — release triage and task-linked coordination

**Operator story.** QA finds a release blocker during a bug bash. The release lead
uses AGH to coordinate frontend, backend, and QA agents, and task/run state stays
connected to the network channel.

**Setup.**

- Create `release-triage` with QA, frontend, backend, and release lead agents.
- Create a real task or task tree with `network_channel: release-triage` through the
  public task surface available in the lab.

**Journey.**

1. QA posts a reproducible bug summary to `release-triage`.
2. Release lead assigns investigation through task/run APIs or CLI.
3. Frontend/backend agents exchange at least one directed handoff in the channel.
4. Browser validates `/network` channel and peer rooms, then navigates to the task/run
   surface when the route is in scope.
5. API verifies the task/run `network_channel` and channel message history.

**Assertions.**

- The same release blocker is visible as a network message and as task/run state.
- The task/run does not drift to a different channel.
- Browser state is actionable for an operator: who owns the bug, which channel carries
  coordination, and what remains blocked.
- A blocked dependency, failed run, or wrong-channel assignment is treated as product
  evidence and filed if the UI/CLI/API disagree.

---

## 7. Scenario D — daily operating digest

**Operator story.** The founder starts the morning by reconstructing what happened in
operations, product, and engineering channels. AGH should make useful coordination
history visible without requiring direct database inspection.

**Setup.**

- Create `ops-digest`, `product-digest`, and `engineering-digest`.
- Seed each channel with at least one meaningful `say` message and one directed
  follow-up.

**Journey.**

1. CLI lists channels and peers for each digest room.
2. API fetches channel message history with and without presence messages when
   supported by query parameters.
3. Browser opens `/network`, searches/selects each room, and captures the default
   timeline plus the presence-inclusive view if the UI exposes the toggle.
4. Operator records a morning summary artifact outside the browser, linked back to
   the channel/message IDs.

**Assertions.**

- Persisted channel history is readable through public API and browser state.
- Presence events are available for debugging but default operator history is not
  dominated by heartbeat noise.
- Browser room selection, kind filters, and detail rail help the operator understand
  the day, not just inspect raw protocol events.
- A stale or missing message in Web that exists in API is a bug, not a test caveat.

---

## 8. Scenario E — hiring funnel coordination

**Operator story.** A startup founder coordinates a synthetic hiring loop. Recruiter,
screener, and hiring manager agents exchange structured notes and a final decision
without leaking real candidate data.

**Setup.**

- Use only synthetic candidate identifiers and content.
- Create `hiring-funnel` with recruiter, screener, founder, and hiring manager agents.
- If task surfaces are in scope, create a task with `network_channel: hiring-funnel`.

**Journey.**

1. Recruiter broadcasts candidate context to the channel.
2. Screener sends direct feedback to the hiring manager peer.
3. Founder posts final decision criteria to the channel.
4. Browser verifies the channel timeline, peer rooms, and any task/run adjacency.
5. CLI/API confirm the same messages and channel binding.

**Assertions.**

- Synthetic hiring data stays in the intended `hiring-funnel` channel and peer lanes.
- Direct feedback is not shown as a broadcast message.
- Channel/task binding remains consistent if a task is updated or run is enqueued.
- Wrong-channel task mutation rejects or surfaces a clear stale-channel error where
  the public task API exposes it.

---

## 9. Scenario F — capability handoff

**Operator story.** The operator needs a researcher, coder, and reviewer to coordinate.
Before sending a handoff, the operator checks which peer advertises the right capability.

**Setup.**

- Configure at least two agents with distinct capability catalog entries when the lab
  supports authored capabilities.
- Create `capability-handoff`.

**Journey.**

1. CLI lists peers in `capability-handoff`.
2. API/Web inspect peer detail and capability brief for the target agent.
3. Operator sends a direct handoff to the selected peer.
4. Browser verifies the peer room and capability context are visible enough to explain
   why that peer was selected.

**Assertions.**

- Capability briefs are visible through public peer surfaces when configured.
- Rich catalog data is requested only through the supported discovery/API path; do not
  invent automatic capability routing.
- Direct handoff goes to the selected peer, and the browser timeline matches CLI/API.
- If capability data is absent, the scenario records the missing product evidence
  instead of pretending routing was capability-aware.

---

## 10. Technical readiness and regression checks

These checks remain important, but they are support rails for the startup journeys
above. They can be Go integration tests, API tests, CLI tests, or Web component tests.
They do not count as complete E2E scenario proof unless tied to one journey above.

| Check                                   | What it protects                                                               | Suggested surface                                                    |
| --------------------------------------- | ------------------------------------------------------------------------------ | -------------------------------------------------------------------- |
| Capability brief vs full catalog        | `greet` and `whois` projection do not expose too much or too little.           | Go integration + API peer detail + Web peer detail.                  |
| Remote peer TTL                         | Silent peers disappear after `2 * greet_interval` and reappear on fresh greet. | Go integration + API peers + Web peer list refresh.                  |
| Interaction correlation                 | `interaction_id`/`reply_to` keep directed exchanges scoped under fanout.       | Go integration + CLI send/inbox + peer messages API.                 |
| Terminal interaction rejection          | Completed/failed/canceled interactions cannot be reopened.                     | Go integration + API rejection evidence.                             |
| Expired or duplicate envelope rejection | `expires_at` and replay-window behavior protect inboxes.                       | Go integration + audit metadata.                                     |
| Queue overflow FIFO                     | Busy sessions drop oldest queued messages first.                               | Go integration + CLI inbox + status metrics.                         |
| Prompt back-pressure                    | Network prompts queue when a session is already prompting.                     | Go integration + CLI inbox/status.                                   |
| CLI/API/Web parity                      | Shared contract payloads stay consistent across operator surfaces.             | E2E harness + browser `/network`.                                    |
| Presence dedup                          | Heartbeats do not dominate operator timelines.                                 | API messages + Web kind/presence controls.                           |
| Task ingress gate                       | `task.write` and channel binding reject unauthorized or stale-channel peers.   | Task API/integration + browser task/network adjacency when in scope. |
| `network.*` config lifecycle            | Network config defaults, validation, and disabled behavior are stable.         | Config CLI/API + Web disabled state.                                 |

---

## 11. Removed or blocked until primitives land

Keep these out of final scenario success criteria until the named primitive exists.
They may be mentioned as known gaps or future scenarios.

| Topic                                                     | Missing primitive or surface                                                                  |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Automatic on-call/moderation routing by capability        | Runtime-owned capability selection/routing policy, not just discovery.                        |
| Full support transcript replay from audit JSONL           | Body persistence or a documented transcript assembler from message store.                     |
| Cross-restart lifecycle and inbox recovery                | Durable lifecycle state, durable delivery queues, deterministic reattach/redelivery contract. |
| Broker restart, external NATS, federation                 | External topology and reconnection contract beyond embedded transport.                        |
| Time-skew rejection with `clock_skew`                     | Future timestamp tolerance and registered reason code.                                        |
| Direct-to-self rejection with `ReasonCodeDirectedToSelf`  | Registered reason code and explicit self-target policy.                                       |
| Prompt-injection heuristic detection                      | Runtime detector and operator-visible signal beyond untrusted wrapping.                       |
| Body-level claim token or PII redaction in network audit  | Body-persisting audit path and network-specific redaction policy.                             |
| Slow-consumer metrics                                     | Exposed NATS/runtime slow-consumer counters.                                                  |
| Slack/webhook/email bridge workflows                      | Bridge runtime wiring and operator-visible bridge state.                                      |
| Recipe, `echo`, `revoke`, tribute, signatures, hash-chain | Corresponding protocol kinds, canonicalization, verification, and persistence.                |
| Hook dispatch from inbound network events                 | Typed hook call sites for accepted inbound network transitions.                               |
| Mid-session capability mutation re-greet                  | Capability mutation API and heartbeat invalidation/re-greet mechanism.                        |

---

## 12. Highest-leverage QA seeds

Start with these in order:

1. **Launch room coordination** because it exercises Web channel creation, CLI/API
   parity, channel messages, peer visibility, and operator readability.
2. **Customer escalation room** because it proves direct messages, handoff semantics,
   inbox behavior, and peer-room browser evidence.
3. **Release triage and task-linked coordination** because it proves network state can
   matter to adjacent startup work, not just standalone messaging.
4. **Daily operating digest** because it catches stale Web reads, noisy presence
   history, and persistence/read-model gaps.
5. **Capability handoff** because it tests the product promise of discoverable agent
   abilities without pretending automatic routing exists.

Each seed must finish with a QA report that includes the bootstrap block, exact CLI
commands, API payloads, browser URL/screenshot/DOM evidence, generated artifacts or
task IDs when present, and filed issues for every mismatch.
