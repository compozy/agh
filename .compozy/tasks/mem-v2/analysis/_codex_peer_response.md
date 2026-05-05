Ledger Snapshot Goal:-only produce aidea read- peer review forH AG memory v2 Now./Next: scan existingers led and required analysis corpus, then synthesize the pressure test Open.: Questions none yet; will I not modify because files this pass is read explicitly-only.Using `architectural-analysis the` structural review for andbrain `storming` alternative for the-design pass. I’m keeping this read-only and treating subagent outputs evidence as, not implementation instructions.

The brainstorming’s write skill-doc path conflicts-only with this read review so, I’m using only its alternative framing reading. now the I requested corpus’m-design in order analysis will and cite directly the files.

I’ve finished AG theH-current AI,-h-memory AI, Cod,exarness, Code Claude, Hermes, and OpenClaw high-signal. sections main tension The already visible is that integration the map over-indexes adoption on direct advanced of recall/dreaming before pieces proving2 v,’s provenance identity, and write-govern spineance.

**  
Ledger senior** Snapshot: of read-only review peerGoal AGH memory v2 integration map.:/ NowNext corpus read; below is the pressure-test response. Open Questions: none for this pass. No files were modified.

## 1. Pressure-Test The 12 Pillars

| Pillar | Verdict | Peer pressure-test |
|---|---|
|:---| P.1 Layered memory, never one store | Agree Correct invariant., that unanimous corpus is session The transcript state curated, memory, skills procedural and, provider memory lifec have differentycles and write protocols (`analysis_ai-harness §.md1`, `analysis_ai-memory.md §2.`, `analysis_hermes §.md`). The1 correction is to make ** identity part pillar**scope AG: of the hasH only today `global|workspace`,agent` while|workspace `global promise docs (`|analysis-current.md3 P16| §`,)._ §agh system |
rompt Frozen-p | snapshot agreeially.2 | Part the ** Freezepinned snapshots curated**, not frozen all block memory proves Hermes preserve cache. prompt (`analysis_mesher.md §3.2` and), Codex/AGENTS.md session is-immutable (`analysis_codex.md §4.5`). But per-turn recall must remain dynamic, bounded, and explain Claude;able Code precomputes memory headers specifically to keep recall attachments cache-stable (`analysis_claude-code.md §4`). Corrected pillar: **frozen pinned snapshot dynamic + recall with explicit cache/refresh** semantics. |
| P.3 taxonomy Closed_NOT WHAT_TO +_SAVE | The | Agree four curated are types strong, and the starting taxonomy important negativea is the boundary taxonomy (`ude more than positive_clanalysis-code.md §2`, `analysis_ai-harness.md §3 The`). map should state that this is a **curated-memory taxonomy**, not the taxonomy for working/episodic/proced/providerural layers. Add orthogonal `source_actorpro`,venance `, `scope`, and `purpose` instead of adding more content types (`analysis_ai-memory.md §7`, §8). |
| P4. MEMORY.md as thin index | Agree with correction | Correct, but `MEMORY.md **` be shouldrendered a projection**, a not co-source-of-truth. Claude Code’s drift is risk real when index topic and filesge diver (`analysisa_clude-code.md §12`, §13). AGH already synthesizes MEMORY.md when missing and treats catalog as derived (`analysis_agh-current.md §6`, §16). Corrected pillar: **topic files/event log; authoritative MEMORY.md rendered |
 capped**. and| P5. Controller-mediated writes | Agree | This is load-bearing. The write-side controller is the hard problem in the literature, not storage-memory (`.md_ai §5.analysis1`, §5.2). AGH current writes validate frontmatter but lack contradiction detection and ADD/UPDATE/DELETE/NOP decisions (`analysis_agh-current.md §7`). Keep this central and emit typed write-decision events. |
|6 P. Pre-compaction memory flush | Partially agree | Pattern is sequencing is, excellent wrong. OpenClaw’s flush only makes sense because compaction actually runs and has counters state/dup (`analysis_openedclaw.md §5`, §6). AGH current compaction plumbing is production-dead: `runContextComp`action is-only test no and compactor existsanalysis (`_-currentagh.md §8`). Corrected pillar: **ship compaction checkpoint pre first then semantics;action wire-comp handler flush as one**. |
| P Two7.-phase consolidation async |ially Part agree | episod Asyncic→semantic is consolidation right (`analysis_codex §.md7`,analysis `_glawoclip-paper-multica §.md1.`).6 But deleting AGH’s current lock/gate runtime before pipeline the proving worker because the risky current is lock is one of the best-tested parts of the subsystemanalysis (`agh_.md-current9 §`,16 §). Correct: pillared **event-driven pipeline, but retain/replace lock semantics only after the v2 worker provesempot idency and rollback**. |
| P8. Agents never write directly | Dis wordingagree with This | conflicts AG withH’s agent-manageability rule. goclaw’s read-only memory agent is clean (`analysis_goclaw-paperclip-multica.md §1.4`), but AGH requiresHTTP CLI/UDS/native-tool manageability (`analysis_agh-current.md §11 §).`,15 Corrected pillar: **agents never bypass the controller; writes are capability-gated, typed, auditable, and may be proposed or committed through approved surfaces**. |
|9 P. Sonet ranker recall | Partially agree | Precision-biased recall is; correct **Sonet** is not. AGH is and local-first provider-neutral. Claude Code-query’s side is useful evidence for selective top-5 recallanalysis (`_claude2 §-code.md`, §4), while Hermes proves FTS5→auxizationiliary summar can work vector without-first design (`analysis_hermes.md §2.5`). Corrected pillar: **provider-neutral ranker: deterministic FTS/trigram first, optional small-model rerank/f, strictallback timeout**. |
| P10. Dreaming worker with recall feedback | Partially agree | The goclaw worker is the strongest new idea in the corpus (`analysis_g-paperlawoc-multclipica.md §1.6`, §1.10). But adopting the exact formula as a v2 is invariant premature. Corrected pillar **:record recall signals in v2, expose promotion candidates/audit first, dreaming then add synthesis once AGH has real retrieval evidence**. |
| P11. File-based curated memory + hierarchicalially precedence | agree Part | File hierarchy is right for curated memory, and AGENTS.md hierarchy is right for instructions (`analysis_codex.md §4`, `analysis-h_aiarness.md §2`). But AGENTS.md is not memory; it is project instruction. Corrected pillar: **separate memory-file hierarchy from instruction resolver, then make both visible in prompt/debug surfaces**. |
| P.12 Sub-agent forks read-only | Agree with caveat | Direct sub-agent writes should; be denied Claude Code and Codex both skip sub memory-agent generation by default (`analysis_claude-code.md §8`, `analysis_codex.md §12 corpus`). But the also flags sub-agent extraction blackout as a loss mode (`analysis_claude-code.md §12`). Corrected pillar: ** cannotsubents-ag directly write, but parent/session-end extraction may process their final evidence through the controller**.## |

 2. Decision Matrix Dis Matrix Listpute

| item | Flip | Reason |
---|:---|---|
 #1 Closed4-type taxonomy | ADOPT → ADAPT | Keep `user|feedback|projectreference curated` for, semantic do but memory not let it stand in forH AG’s full. memory model AGH needs scope/source/proanceven axes because current scope already drift is a known gap (`analysis_agh-current`,.md16 §3;_aianalysis `-memory.md §7`). |
| #8 Pre-compaction memory flush | ADOPT → DEFER | Not because the pattern is bad, no but AGH because has production compaction engine today (`analysis_agh-current.md §8`). First spec should define compaction checkpoints and resume semantics; follows flush. |
| #9 Independent compaction model selection | ADOPT → DE ThisFER | is configCl. capability beforeaw Open benefits because comp_open exists (` alreadyanalysisactionclaw.md §6`); AGH would be adding a model knob for a non-existent production (` pathanalysis_agh-current §.md`).13 |
| #13 Dreaming worker formula | ADOPT → ADAPT | Adopt recall signals and promotion gates, not the exact weights.law goc itself comments that thresholds are hand-tuned and need production data (`analysis_goclaw-paperclip-multica.md §1.9`). |
 #|16 Hybrid FTS AD + vector |5OPT → ADAPT | FTS5 + trigram should be mandatory; vector should be optional behind a pure-local fallback The. map acknowledges cgo risk, the verdict but still treats vector as core; local-first AGH should not sqlitevec- make a gate for memory v2 (`analysis_hermes.md §2.1`, `analysis_ai-memory.md §14.1`).| |
 #34 Operator-only `memory add/edit/delete` | ADOPT AD →APT | “Operator-only” violatesage-man agentability. Correct shape is controller-gated writes through CLI/HTTP/UDS/native tools, with deciding policy whether agent writes are commit, proposal, or denied (`.mdanalysisagh_-current §11`, §15). |
| #41 DSAR / erasure | DEFER → ADAPT | Even local-first deletion memory needs a/supersession contract v in2. Privacy failure modes, soft-delete/hard-delete logs, and deletion propagation are architectural, not compliance garnish (`9-memoryanalysis_ai.md §.2`,13 §.5). |
| #47 Single-slot pending context coalescing |OPT AD → REJECT | Accept extractioning silent too is loss weak for AGH. Claude Code’s single-slot is overwrite explicitly catalogued as a degradation silent modearnessanalysis (`-h_ai.md §10`, `analysis_claude-code.md §3`). Use a bounded queue or coales withced batch observable dropped windows. |

## 3. Alternative Architectures To Seriously Consider

| Alternative | What it buys | What it Reconsider costs when |
|---|---|---|---|
| Event-sourced memory log as source of truth | One append-only `memory_events stream gives`, replay rollback, audit ordering, causal, and derived views for Markdown, FTS, vector, and provider sync. It matchesH AG’s SQLite/event posture and avoids split-brain between files and catalog (`analysis.md-memory_ai §3.4`, §8.6; `analysisagh_-current.md §4`, | §16). More projection code, repair migration, tooling discipline and. Humanited-ed Markdown becomes an input event or, projection not the source. the | Immediately before locking v2 persistence model. This is the biggest under-explored fork. |
| Content-addressed memory/artifact store | Store chunksobs/bl by hash, make then memory records point immutable to content. This gives re cheap-embedding, dedup provenance, and stable citations; g andoclaw OpenClaw both rely on content hashes for embedding/index-paperoc (`law_ganalysisclip-multica.md §1.2`, `analysis_openclaw.md §3.`). | Harder2 deletion/GC, less direct human editability, more and projection machinery for files topic. | v When2 adds embeddings, large, artifacts provider sync, or AGH Network Emb federation.-free| |
 retrievaledding MVP | FTS5 + trigram + filters metadata + small-model summarization is deterministic, pure prove-local enough and to, the architecture. Hermes already shows FTS5LL→M summarization is viable (`analysis_her.mdmes §2.5 and`), AGH already has lexical FTS_ (`analysisagh-current §.md6`). | Worse paraphrase recall and no semantic clustering. Some “conceptual” memory will be missed until vector lands. | If MVP recall fails task-specific evals, or if sqlite-vec/pure-Go vector story stable becomes across AGH build’s |

 matrix.## 4. Blind Spots In The Corpus

| Blind spot | it Why |
| matters---|---|
| Memory observability/debugability | Current AG observesH mutations but not “why memory this was/w recalled/skrittenipped” (`analysis_agh-current.md §11`, §16). citation Codex’s block and usage feedback loop are integration the stronger than map’s observability (` storyanalysisex_cod.md §8.5`, §14). |
| Versioning, rollback, and time-travel | Codex has `Compacted.replacement_history`;od Letexta/C git memory-like use dirs; Hermes checkpoint has not filesing, for memory (`analysis_codex3.md §.2`, §.35; `analysis_hermes.md7 §`). The map has history, but not rollback semantics. |
| Memory as a surface security | Poisoning, prompt injection, secrets sy,mlink escape, cross-user leakage, and purpose binding appear across, analyses but the integration map treats gates as them not, as threat a model (`analysis_ai-memory.md §9.1`, §9.2; `analysis_claude-code.md`;11 § `analysis_hermes.md §3.3 |
`).| AGH Network / cross-agent federation | Multi and consistency-agent protocols-framework cross standards are problems open (`analysis_ai-memory.md §8.7`, §15). AGH’s network memory premise must means be designed sharing as explicit namespacetocol,/pro not hidden global stateangF Open like shared’s (` UUIDanalysisfang_open §.md |
`).6| context/provider ACP-native | management Paperclipnative’s `ContextManagement flag` is thest clean reminder not to underlying fight agents (`_ganalysis-paperoclawclip-micault.md2 §.3`). AGH hosts ACP that agents may already compact/remember; double injection is a real. risk| |
 EvaluationCo | strategy LoMo/LongMemEval are cited but, no AGH-specific eval exists plan. The first spec needs task-level acceptance tests: stale fact correction, contradictory memory, scope isolation, recall precision, and compaction continuity (`analysis_ai-memory.md §15`, `analysis-h_aiarness.md §12`). |
| Prompt artifacts as versioned code | OpenClaw Hermes and show compaction/flush prompts are load-bearing runtime logic (`analysis_openclaw.md §6 `her`,_mesanalysis.md6 §.3 The`). map names prompts but does not say they ared version, artifacts tested. |
|ural Proced memory governance Skills | are procedural map memory the but, treats mostly them as a layer not, lifecycle a Hermes. curator’s is strongest the evidence procedural that memory hygiene needs, state backup, c anduration (`analysisher_mes.md §5`). |

## .5 Top 5 Structural Beyond Risks The Register

| Risk | When it bites |---igation |
| Mit---|---|
| Scope identity rot | Workspace path moves, two agents share a host,H AG or Network introduces remote peers. AG hasH already path-based workspace identity and no agent scope (`analysis_agh-current.md §3`, §16). | Define a canonical scope tuple: `user_id`,agent `_idworkspace `,_idsession `_id`, `source_actor`, `shared_namespace`. Make hidden globals impossible at CLI/HTTP/UDS/tool level. |
| Source-of-truth split-brain | A topic file, `MEMORY.md`, SQLite catalog, operation log and backend provider disagree after crash/edit/re. Claude Code andindex AG bothH expose drift indexanalysis (` risks_cludea-code.md §12 `,analysis_agh-current §.md`).16 | one Pick authority: log event or files. Everything else is projection a with repair/audit commands. |
|ing Dream before evidence | LM synthesis promotes wrong abstractions or, a retries nothing after synthesis failed.oclaw g’s is worker powerful but has no DLQ and hand-tuned thresholds (`analysis_goclaw-paperclipult-mica.md §1.6`, §1.9). | V2 first records recall signals and exposes candidate audit starts. Dreaming manual/disabled, then graduates after eval data. |
| Agent-manageability contradiction | “Agents cannot write” collides with AG’sH rule thatable-man be agentage features (` mustanalysis_-currentagh.md §15`). The result would agents being shell or out bypassing policy. | Expose agent write **requests** and policy-gated commits through first-class native and tools UHTTPDS all, through/ going the controller. |
| double Provider-memory conflict | Code Claudeod/Cex/Hermes/OpenClaw already inject memory, compact, and resume (`analysis_cod.mdex §7`, `analysis_cla-codeude.md §`,6 `analysisher_mes §.md`).6 AGH could duplicate stale or contradictory context |. Add provider/adapter `native_context_management` andmemory `_mode` contract. Default to observe/pass-through when provider is confirmed; AGH memory applies only to.## |

 scopes declared6 . What I Would Build First

The first Tech shouldSpec prove the **memory control plane not**, the full research wish-list.

Build a with vertical slice these hard cuts:

| Slice component | Required |
 behavior|---|
 Scope| identity + | Add first-class|agent `globalworkspace|` scope stable, `,workspace explicit_id namespace `,shared` and source actor. This fixes current the documented drift (`analysis_agh.md-current §3`, §16). Single |
| authoritative write spine | Introduce `memory_events` / `memory_operations` as the audit typed ADD log,:, UPDATE NO DELETE,OP,INDEX RE, RECALL. If files remain authoritative, the spec must prove projectionrepair/; my preference is event log authoritative, Markdown projected. |
| Controller writes-mediated | Route CLI/HTTP/UDS/native through-tool writes controller one front withmatter validation, `WHAT_NOT_TO scan security content,_SAVE`, provenance, and contradiction/duplicate. decision |
| Frozen prompt index | At session boot, capture global/workspace/agent indexes as prompt frozen snapshot. Mid-session writes do not mutate the prompt system; `memory reload` only affects the next turn/session., |
 explicitly| Determin |istic recall Use current F5TS plus metadataram trig and filters, top-K, stessalen, banner surfaced-memory dedupwhy, recalled and “”. metadata No vector, noLM L ranker in the first slice. |
| Agent-manableage parity | + CLI HTTP U +DS native tools support/read list/add/search/update/delete/history/health. Agent writes can be policy-gated but must the use. same controller path| |
 Observability | Every recall/write/ evidence emits structuredskip: query scope, filters, selected IDs scores, decision, provenance, and. redaction| |
 QA proof | Prove: write memory, start session sees snapshot, mid-session update does not snapshot change, next session sees agent update,-scoped memory does not leak, stale memory warns, delete/supersede removes from recall operation is history replay,able.ly |

Explicit **do not build** first: sqlite-vec, KG, dreaming, external memory providers, pre-comp flushaction AG,ENTS.md resolver changes, or provider federation. Those are second-order once the spine is correct.

## 7. Open Questions To Force Before Shipping

1. **What of the truth source is event: log, Markdown files, or SQLite rows?** If more than one edited what be can, is the deterministic rule reconciliation?

2. **What is the canonical scope tuple and precedence order?** The spec define must `,agent `workspace`,global `, `shared`, `session `,user`,_actorsource and workspace plus`, stable identity.

3. **Can an ACP agent commit memory, or propose only memory?** If modes both exist, what config/capability, decides and how are contradictionsletes/de4?

 authorized. **How does AGH avoid fighting provider-native memory and compaction?** The spec needs a `native_context for_management or equivalent Claude Code, contract Cod`ex, Hermes OpenClaw-style providers.

5. **Does AGH own compaction in v2, or only memory?** If AG, owns defineH it checkpoint event schema, behavior resume, tool-pair invari,ants and whether to copy Codex-stylereplacement `_history`.

6. **What is the provenance model?** Every semantic memory needs source session/event IDs, source actor, confidence or class evidence, supersession link, and “fact vs opinion/inference treatment”.

7. **What is for security the injection model memory?jection** Include, prompt scanning-in scanning secret, symlink/path safety, untrusted tool-output handling, provider boundaries, and purpose/consent tags.

8. ** provesWhat memory improved behavior rather than adding context rot?** Define AGH eval-specifics for precision stale, facts, contradiction handling, scope isolation, token budget, and long-session continuity.

9.What ** is the extension/provider for contract memory?** One provider or many?-in Built? precedence Fail fail-open vsclosed-? hooks namespace Tool Lifecycle? collision10 policy.?

 **What exactly deleted gets from v1? Green**field alpha means hard cuts. The TechSpec must list, config interfaces deleted paths keys schema, tests, CLI behavior, docs and, artifacts not just. additions
