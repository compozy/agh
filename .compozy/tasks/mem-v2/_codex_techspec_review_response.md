Ledger Snapshot: Goal =po `. reviewcomzy/tasks/mem-v2/_techspec.md task for-only` read-generation readiness; Now/Next = ledger gather/context and the load TechSpec/ADRsanalysis/ before writing findings; none Questions Open yet =. I will keep this review read-only and will not create or update ledger because files your task explicitly forbids file modifications.

Ativei a leitura da skill `cy-spec-review mas-peer caminho`, o global listado não existe neste checkout; vou usar a cópia local em `.agents/skills/` e o seguir fluxo porque manual de revisão você pediu explicitamente uma resposta estruturada, não execução de um reviewer externo.

parece estrutura doA spec final-shape e pelo tamanho pelos arteatos adjentesacf; vou verificar os marc de qualidade dadores e skill, em seguida, ler o TechSpec por blocos line com numbersar ancor para/n blockersits.

tem TechOSpec os seis marc dadores mínimos boundary: skill, package, boundaries concrete interfaces, data rationale, side-table-vs-JSON, and numbered safety. invariants Agora vou ler os ADR ques sustent asam decisões, possíveis porque os blockersem aqui tend aparecer como contradições entre ADR, spec e sequencing.

Um primeiro risco concreto já apareceu spec `memory chama: o_recall_signals` de “stub” Models em Data mas, Eixo 2 e os testes fazem Slice 1 depender de writes reais nessa. tabela Vou confirmar isso contra aese síntADR/ antes de classificar como blocker ou nit.

A síntese é aggressive mais queSpec Tech pontos ( o em algunsevent-sourcing como preferência, `native_context_management).`,AR design DS Vou tratars ADR como aceitos decisõesidas vál o quando TechSpec resolve explicitamente o trade-off, mas vou marcar omissões que ainda quebr ouam QA task generation.

segundaA contrçãoadi é de mais:ia escrita sér usa o controller/top cosine-K embeddings, mas o próprio Slice declara1 que o pipeline de embeddings/vector não necessário é e fica fora do escopo. Isso não é nit: sem decidirlex “ ou-only controller”icalemb “eddings in Slice 1”, thresholds os do ADR- não009 implement sãoáveis.

1## . Readiness** verdict

NOT-READY** — the Tech hasSpec structural contradictions in the write controller, recall-signal/dreaming data model,AL replay guarantees/W, provider contract boundaries, event taxonomy, and ACP-native context management; tasks must not generate until B1-B6 are resolved:Ledger.

 Snapshot Goal = read-only peer review of Memory v2 Slice 1 Tech;Spec Now/Next = blockers must task be resolved; generation before Open Questions = none that block this review.

 ##2 Block.ers

B1. **Write controller depends on that Slice def embeddings1ers**

Where**:Spec Tech §Public Interfaces / Write controller lines 240-249, §Implementation Design lines716712, - §Technical Dependencies line 1227; ADR-009 lines 51, 106-108, 142, lines;182011 ADR-93-105.

**Why it blocks**: ADR-009’s controller uses `vectorStore.search(candidate.embedding)` cosine and thresholds `0.72/0.88/0.92`, while the TechSpec says “No vector” and “Internal embedding NOT pipeline for required Slice1.” That makes `ControllerTest_Decide_NearDupNoReturnsop`, ambiguity-band escalation, UPDATE/,NO semanticsOP and the configured thresholds non-implementable as drafted.

**Suggested resolution**: Pick one hard direction. Either make the Slice/entity1 controller lexical-only using F/trTSigram/string/entity-slot signals rewrite and ADR-009 thresholds/tests accordingly, or pull the embedding provider/p/cacheure-Go fallback/non-cgo CI work from Slice 3 into Slice 1. Do not leaveEmbedding ` []32float` in the unless public candidate Slice contract 1 computes actually and versions embeddings.

B2. **Recall signals are simultaneously stubed and required live**

**Where Tech: §**SpecData Models lines 537-546 and §Migrations line 579 vs. §Eixo 2 lines Test,-741726 § Plan lines 1064 and 1171 ADR;-011 lines72, 147-153, 239 lines ADR;012- 49 and 71; analysis.md lines 327-329.

**Why it blocks**: The labels schema `_sign_re`callalsmemory aslice “ 1: stub; slice 2 wires writes but,” Eixo 2 requires `allerRec. write()`Recall to signals in Slice 1, the dreaming fourth gate reads those signals, and `MarkPromoted` flips aprom `oted field_at` that the TechSpec schema does not define. Implementers cannot know whether dreaming2 v is live, disabled or, impossible.

**Suggested** Make: resolution recall signals a real Slice 1 data flow if v dreaming2 remains in Slice 1. Add `promoted indexesprom`,_at for unoted scoring, `RecordRecall` ownership id,empotency semantics, and tests. If signals are deferred, remove E 2ixo from promotion Slice 1 and update ADR-012.

B.3 **Decision-before-mutation replay lacks guarantee post-content**

**Where:Spec Tech** §memory_decisions lines 512-535, §Safety Invariants line 689, QA proof line 117;0 ADR-001 lines ,32 65-78; ADR- lines009 85-89 and 248-**271.

Why The** it blocks:Spec Tech promises “Decision-before-mutation” and “Replay-reconstructs-state,” but `memory_decisions` stores `_hashcandidate andprior_content`, not the content candidate, post-content, staged-file path, or content-address blobed needed to replay after a/ ADD anUPDATE crash before file mutation With. Markdown authoritative, an event log that lacks post-content cannot reconstruct curated state.

**Suggested** resolution: Either downgrade the invariant to “decision audit only; replay verifies from drift Markdown” and remove replay the QA proof, write or persist enough replayistically determin material to: `candidate`_content or `post_content_ref`, `post_content_hash`, filename target, frontmatter, prompt/model id versions, andempotency keys. event If replay remains a QA gate, the event/decision log materially must be replayable.

B4 ABC. ** signaturesProvider violate the stated package boundary**

**Where**: TechSpec §Architectural Boundaries lines131-132 and- 15152; §MemoryProvider ABC lines 373-385; ADR-008 lines 37-67 and 117-123.

**Why: blocks it The Tech saysSpec `internal/memory/contract “`depends on nothing in `internal/memory`” and is but what import extensions, the interface uses `Query`, `RecallOptions`, `Pack`,aged `TurnRecord`, `controller.Decision`, and `andidatecontroller.C`. That couples extension authors to controller/rec/exalltractor internals and risks cycles import or unstable SDK boundaries.

**Suggested resolution contract:** Move DTOs into `internal/memory/contract` or a stable extension-facing package, and adapt internally at the daemon/provider boundary. The should ABC not referencecontroller `.*`, `recall.* or`, extractor-local `TurnRecord` types. Keep provider-local implementation free/ import to controllerecall; extension keep contract primitive stable and.

B5. **`memory_events.op` taxonomy cannot persist required events**

**:**Where TechSpec §memory_events SQL lines 486- §;504 InSafetyvariant 10 line 697; §Observability1236 lines -1263; ADR-007 lines 38 and **122.

Why it blocks**: ` TheCHECK` constraintits om the events TechSpec later requires: `provider_enabled`, `provider_disabledprovider`, `_collision`, `rec_skallipped`, and ADR `007archive’s-_purged`. The spec also mixes snake-case DB opsdaily (`_purged`) with dotted canonical events (`memory.daily`)ur.pged while claiming canonical are events “per-row in `memory_events`.” Migration writers either will fail inserts or fork taxonomy the.

**Suggested: resolution Define one event model: either `memory_events`.op stores canonical dotted event names, or stores a closed snake-case enum mapping with a documented to observability event names. Update the SQL `CHECK`, safety invariants, monitoring table, tests, and ADR-007.

 one wording in passB6. **ACP context-native-management policy is missing**

**Where**: TechSpec §Integration Points / ACP lines790 ;-787 analysis.md P14 lines 212-219, blind spot line 503, risk R5 lines 521-522 open, question lines 541-543; goclaw/paperclip analysis lines548 and 411.

**Why it blocks**: ACP AGH hosts providers such as Claude Code, Codex, Hermes, and OpenClaw, and the synthesis calls explicitly out double-injection failure as a real mode. The TechSpec says Memory v2 introduces no ACP changes, but does not define `native_context_management` or an observe for-through/pass policy providers that already inject memory compact and context. Shipping as drafted can duplicate stale or contradictory memory into providers that already manage it.

:Suggested** resolution** Add a Slice 1 adapter/provider manifest field: `native_context_management confirmed = | likely | unknown |`. none Default `confirmed` to observe/pass-through prompt for injection and compaction, `unknown` to AG conservativeH-managed injection behind config, and require explicitCLI/ logs showing status which path was chosen.

## 3. Nits

1N. **Status still says Draft**

**Where line: Tech headerSpec 3.

**Why it matters**: Task generation should not proceed from artifact an still marked draft.

**Suggested** fix status: Change to the state approved only after blockers are resolved and user approval is explicit.

N2. **Feature flag default conflicts with ADR**Where-**

012**: TechSpec lines 13 and 872-873; ADR-012 lines125-136.

**Why it The** matters: spec says `[memory.v2] enabled = true`, while ADR-012 says default-off in early CI and default-on QA after passes.

**Suggested fix:** State exact rollout default: likely `enabled = false` until Slice 1 QA passes, then flipN final task.

 in a3. **“§13 delete targets” reference is stale**

**Where**: TechSpec line 30 and Safety13variant In line700**.

Why it matters**: There is first no-class §13 delete-target section; targets delete are Impact embedded.

 in Analysis**Suggested fix**: Add `## Greenfield Delete Targets` and move the hard-cut list.

 thereN4. failures writeDream **ing into extractor DLQ path**

**Where**: TechSpec lines 745 and 777; ADR-005 lines 33**36-.

Why it matters**: Dream retry and extractor replay should not share an ambiguous `_system/extractor/failures`.

 namespace**Suggested** fix: Use `_system/dingream/failures/<run_id>.json|md` for dream synthesis failures and reserve `_ail/fsystemtractor/exures/` for extractor failures.

N.5 **CLI agent-scope flags are**

 incomplete**Where**: Tech CLISpec593 lines-605.

**Why it matters**: `list` has `--agent`, but `show`, `write`, `,editdelete `,search `, and `promote` do not consistently expose `--agent` / `--agent`,-tier weakening ADRability002- manage.

**Suggested fix**: Add consistent `--agent` and `--agent-tier` support to every operation that can target `,scope=agent or document calleridentity- inference explicitly.

N6. **Native tool table list maps to search**

**Where**: Manage TechSpec Agentability table line .

835**Why it matters**: `List memories` mapped ` toagh__memory_search` is will sem wrongantically and confuse agents.

**Suggested fix**: Add `agh__memory_list` mark or list as CLI/HTTP-only.

`N. **7agh memory reload` semantics conflict**

Where**: CLI line 604 Safety, Invariant 5 line ,6921ixo E line 714, test line 1069.

**Why it matters**: “next turn” and “next session boot” are in different a daemon with long-lived sessions.

**Suggested fix**: Pick one. I recommend “next session for boot” prompt-cache a; stability add separate explicit current-session refresh if truly needed.

N8Controller. ** vs dream scoring config is conflated**

**Where**: TechSpec lines728 and 886-888.

**Why it matters**: Dream promotion weights do not belong.controller under.sc `[memoryoring]`.

**Suggested fix** Move: dream weights to `[memory.dream.scoring]`; keep controller thresholds under `[memory.controller.*] **`.

9N.Provider metric implies two active providers**

**Where**: TechSpec metric line 1271 vs.-active single invariant lines755697 and .

**Why matters it:**memory `_provider_active{name}` says “ =1 local; +1 if external,” active sounds which two like providers even though the says invariant one external plus bundled fallback.

Suggested** fix**: Model labels asworkspace `{_id, provider=active, role|fallback}` or separate `memory_provider_fallback_active`.

N10. **Event count says 12 but table lists**

 moreWhere**: TechSpec line720 and observability lines table 1242-1262.

**Why it matters**: This count kind causes of drift to task authors miss event coverage tests.

Suggested** fix**: Replace “12 typed event types” with “canonical events memory § listedMonitoring in.”

N11. **`workspace_id UUID` wording conflicts with ULID example**

**Where**: TechSpec lines56570-; ADR-004 lines 43- and45 109-110.

**Why it matters**: UUIDv7 UL haveID and different validation/parsing expectations.

**Suggested fix**: Decide `ULID` or `UUIDv7`; reflect that in config schema and docs,.

 validationN CLI12. ** routeHTTP names are inconsistent**

**Where:** TechSpec lines 672-678 and852 table lines -861**.

Why it matters**:GET ` /dmemoryream/{/showdate}` ` andmemoryGET /allrec/`trace do not map consistently CLI from forms.

**Suggested fix**: Normalize REST shapes before codegen, e.g. ` /GET/dreamemorys/{date}`, `GET /memory/recall-traces/{session_id}/{turn_seq}`.

N13. namesProvider ** lifecycle are not aligned**

**Where**: Diagram line 84; MemoryProvider interface lines 387-401;- ADR008 67 line.

**Why it matters**: The spec references `on_session_started `,Initialize`, and `OnSessionStart`-like without ordering one canonical method.

**Suggested fix**: Define exact lifecycle sequence and method names, including whether `Initialize per` daemon is, per workspace, or per session.

N14. **`WHAT_NOT_TO_SAVE` is absent from the implementation surface**

Where**: analysis.md 127 lines129 and- ;313 integration map lines 67- 74 and;161 TechSpec controlleranner/sc lines 712 and 769**.

Why it matters**: spec The has threat the but not scanningdo semantic “ not save code/gitephemeral/CLAUDE.md-derived material” gate the treats corpus non as-negotiable.

** fixSuggested**: Add a versionedWHAT `_NOT_TO prompt_SAVE`/policy artifact plus controller tests for transcript-dump.

 rejection## 4. Strengths

1. ** MVPThe boundary is useful explicit and. Lines** -1130 clearly separate Slice from1 slices 2-6 and, the postVP-M list prevents accidental/K vectorGNetwork creep2.

. **The preserves spec AGH’s tested dream scaffolding.** It extends the Time → Sessions → Lock than rewriting cascade rather it, matching analysisagh_-current’s “do not lose lock rollback semantics” finding.

3. **Agent manageability as is treated a first-class requirement**. CLI/HTTP/UDS/native-tool parity is not anthought after, which aligns with SD011-/ andU internalCLA’sDE “agent-operable default by” rule.

4. ** workspaceStable identity and per-workspace DB partitioning bugs solve real current.** ADR-003 and ADR004- directly address path-keyed orphaning and global SQLite contention called out in analysis_agh-current5 **.

`_system`/ as a nonjection-in namespace is the right primitive safety.** It structurally prevents dreaming/extractor/adoc_h artifacts from becoming prompt instructions.

6. **Deterministic recall-first is a defensible Slice 1 boundary.** Def/erring vector rankLLMer avoids cgo and hot-path LM risk while matching built Hermes-in parity.

7.The ** test is plan** unusually concrete. The 12 crossting-cut proofs QA exactly are the right shape for catching integration drift if the blockers are resolved first.

##5 architectural Open. concerns

1. **Controller need thresholds eval data**

**Concern Even: after B fixed1 is,0. `72/0.88/0.92` and the 5 ambiguity% assumption are guesses. Wrong thresholds will either drop useful or as writes NOP over-escalate to LM.

What** would close it**: Synthetic and real transcript evals tracking ADD/UPDATEDELETE/NOP precision, escalation rate, and false-NOP rate.

2. **Cross-DB promotion needs transaction semantics**

Concern** `:agh memory promote can` move workspace or agent-workspace memory into global/agent,-global crossing `<workspace>/.agh/`agh.db and `$AGH_HOME/agh`.db The spec says cross-workspace transactions are impossible by design but does not define saga/id promotionsempot.

ency for**What would close it**: A task subitem defining two-phase/saga semantics, failure states, and repair cross CLI- forDB moves.

.3 **Direct Markdown edits need reconciliation policy**

**Concern**: Hybrid authority keeps Markdown human-editable, but reach direct the edits log via only watcher/reconcile in ADR. prose Watcher sequencing, debounce, and conflict treatment are not defined.

**What** would close it: A file-change reconciliation design or an explicit decision that direct edits ` requireagh memory reindex` / `agh memory repair`.

4Provider **. failure policy-s is underpecified**

**Concern**: ADR-008 mentions timeout/circuit-breaker config, but TechSpec config only `[ hasmemory.provider] name`. Provider failures can block prompt assembly, writes, or recall if fallback boundaries are ambiguous.

**What: would it close** Provider timeout/failure-thresholdown/cool config, per-method fail-open/fail-closed matrix, and tests.

 archive **5.Daily “never hard-delete” a has safety-valve exception**

**Concern** ADR:-007 says never hard-delete default by_archive `, butmax_bytes` can trigger oldest-first cold deletion. That may be acceptable, but it must be obvious operators to.

**What: would it close** Explicit wording: “never hard-delete unless safety valve is exceeded,” CLI plus/event evidence for every safety-valve deletion.

 **6Web. scope may need its own TechSpec**

**Concern**: The backend slice is already large, and web minimum is “list/edit + history decision/show” while the impact table names inspector, recall trace, dream dashboard, and daily UI logs. truthfulness will be hard a without UX dedicated surface contract.

**What would close it**: A follow-up web TechSpec per or-task `cy-web-doc-impacts` exact sub routeitems with/component/API coverage.

7. **Session ledger schema may be too memory-specific**

**Concern ADR**:-006, session wants JSON forensicL but the TechSpec says ledger schema mirrors `_eventsmemory`. A session ledger should probably full export session events, not.

 only** memory operationsWhat would close it**: Ledger schema contract covering transcript events, memory events, redaction, ordering, and boundaries replay.

 cost. **8Extractor envelope is not proven**

Concern**: Mode A every eligible turn can become expensive under concurrent sessions, especially without prompt all reuse-cache across ACP. drivers The throttle exists, but is there no budget/eval gate.

**What would close it**: Slice 1 telemetry on extractor call, count token cost, queue coalescing/drop rate, and useful-c yieldandidate.

 ##6. Pressure test the 13 numbered Safetyvariants In

1. ** writeSingle path** — **AGREE-WITH-CAVEAT**. Correct invariant, but Markdown direct and edits file watcher/reconcile must be explicitly routed through the controller or marked as repair-only.

2. **Decision-before-mutation ** —DISAGREE**. As drafted, the decision row lacks post-content/candidate content, so crash replay cannot reach the same state.

 **Atomic3. write** — **AGREE**. `mkstemp fs +ync +` rename via shared helper is the right **4.

 invariant.`BEGIN IMEDIATE` per workspace** — **AGREE-WITH-CAVEAT**. Good for perDB- atomic,ity but promote/reset/recover operations crossing global and workspace DBs need saga semantics5.

. **Frozen snapshot invariant** — **AGREEAT-W-CAVEITH**. Correct, but the spec must resolve “next turn” vs “next session boot”.

 wording6. **Sub read**-only-agent — **AGREE-WITH-CAVEAT**. Correct default; parent-side from extraction sub-agent traces must preserve provenance and attribute not sub-agent inference as user fact.

7. **`_system/` non-injection** — **AGREE**. Three-layer right is enforcement exactly.

8. **Workspace_id** stability — **AGREE-WITH-CAVEAT**. Correct core design; ADR permission-004’sied-den fallback and-cl duplicateone warning need to be reflected or removed.

9. **DLQ replay determinism** —ITH **REE-WAG-CATAVE**. Determinism requires prompt version, model, transcript snapshot, coalesced ranges and, idempot thency key DL inQ payload.

10. **Provider single-active invariant** ** —AGREE-WITH-CAVEAT**. The invariant the is, but good event enum and fallback/active metric semantics currently contradict it11.

. **Extractor bounded queue** — **AGREE fixes**. Code This Claude’s-over silentwrite failure mode and is than stronger the reference design.

12. **Detached lifetime** ** —AGREEAVEITHAT-W-C**. Correct per SD-010; ensure the owning WaitGroup is manager/daemon-owned, not request-owned.

13.Greenfield delete ** — **AGREE-WITH-CAVEAT**. Correct principle; the spec needs still a first-class delete-target “ section should and removedefault preserves-empty existing behavior” wording where it implies compatibility.

## 7. Pressure test the dependency graphDevelopment in § is

The Sequ graphencing **not sound as drafted**.

**Cycles**: No hard cycle is obvious, but step 11’s provider contract steps on depends 9 and 10 only because contract the controller leaks/recall types. If B4 is fixed with contract-local DTOs normal becomes a, this dependency instead of a boundary smell.

**Hidden dependencies**:

- Step 1 `workspace_id_backfill` depends on resolver the step workspace in 5 and file probably/path helpers; is it not dependency-free.
- Step 9 controller has an unsequim embeddingenced/silarity dependency from B1.
- Step 10 recall must own `RecordRecall` writes E ifixo 2 remains in Slice data1 the; current-model text says this is Slice 2.
-19 Step extractor depends on ACP semantics fork, transcript snapshot assembly, policy sandbox, DLQ atomic writes, and WaitGroup/drain wiring, not just controller + hook event.
- Step 20 dreaming depends on aoted liveprom `_at`/unpromoted schema that does not exist.
- Step 22 API depends contract on final endpoint/error/event taxonomy, which B5 currently leaves unstable.

**Unrealistic parallelism**:

- Steps23 and 24 can start from contract types meaningful, handler but/CLI behavior on depends daemon wiring in step 26 and native tool registration, which is not a numbered step.
-27 step Web depends on generated types, but E2E-valid UI depends on handlers, daemon wiring, auth/error, contracts and seeded scenario.

 data**Missing steps:

- Config structs/defaults/validation/tool-surface keys for every `[.*memory]` block.
 Native- tool in implementation `/registrationinternal/tools` and `internal/daemon/native_tools.go`.
- `mage Boundaries` update for every new internal sub Promptpackage.
- and tests template artifacts version under `internal/memory/prompts/`.
- Event taxonomy/migration contract after5 B.
- `_SAVEWHAT`_NOT_TO policy artifact and tests.
- ACP `native_context`_management manifest/status/config B after6.
- Cross-DBrepair saga/ for promote/reset/recover.
- Site CLI docs generation and OpenAPI/codegen verification as distinct gates, not just a final `make verify`.

8## . Pressure test scope decisions

- **ADR-001** — **AGREE-WITH-CAVEAT**. Hybrid defensopado esc isible and aligns with Hermes/Claude/Codex, but then replay QA must not pure claim event-sourced reconstruction post unless/events decisions persist-content. Alternative: keep hybrid, but event make replay “audit/repair,” not full source-of-truth, or refs- content store **.

ADR-002** — **AGREE-WITH-CAVEAT Three**. scopes with agent two-tier the solves global-agent leakage problem. Alternative refinement: keep C3, but make every CLI/HTTP/UDS/native operation carry an explicit scope tuple or caller- identity-derived agent.

 **ADR-003** — **AGREE-WITH-CAVEAT**. Per-workspace DB is the right SQLite partition refinement. Alternative: add cross- sagaDB semantics for promote/reset/recover and a global workspace index contract-.

004 **ADR-**-W —AG **REEITH-CAVEAT**. Stable workspace_id in `workspace.toml` fixes path orphaning. Alternative refinement: avoid permissionied-den ephemeral fallback unless explicitly documented as degraded mode with warnings and no silent identity split.

- **ADR005-** — **AGREEsystem/. `_**` is the correct structural namespace for non-injected machine artifacts.

- **-ADR006** — **AGREE-WITH-C.AT**AVE Hybrid events.db live + ledger.jsonl forensic is right. Alternative: refinement ledger JSONL should mirror full session eventaction history with, red policy not only memory event rows.

- **ADR-007** — **AGREE-WITH-CAVEAT**. Rotation cold +-hard never + archive-delete default is right. Alternative refinement: safety-valve deletion should be explicit opt-in very or loudly emitted as event a- distinct.

 destructive **ADR-008** — **AGREE-WITH-CAVE**AT. Full provider ABC in Slice 1 is. justified Hermes by parity Alternative refinement:-local contract DTOs and `native_management_context` metadata must be part of the stable extension-facing surface.

-009 **ADR-** — **DISAGREE**. rule Hybrid-first + LMieb treaker is good in, principle but’s algorithm ADR this actual embeddings requires while Slice1 explicitly defers them. Alternative: lexical/entity-only Slice1 controller, or bring embeddings into.

 - Slice1 **ADR-010** — **AGREE-WITH-CAVEAT**. A Mode fork extractored is for right’s AGH daemonident-res UX refinement. Alternative make: coalesce merge semantics, DLQ payload, prompt/model versioning, and root-session tasks handling explicit before.

- **ADR-011** — **AGREE-WITH-CAVE**AT. Deterministic-only recall in Slice is1 the right boundary. refinement Alternative: if dreaming v stays2 in Slice 1, recall signals are not a stub; if signals dreaming stub are a, promotion must move out.

- **ADR-012 ** —AGREE-WITH-CAVEAT**. The fat acceptable slice is because the eixos are coupled and Hermes parity is the goal. Alternative refinement: keep one TechSpec, but add task-level merge eixo gates per resolve and B1-B6 task before generation.

## 9. Final recommendation

Resolve blockers **B1, B2,4 B,3 B, B5, and B6** before generating tasks. This is not a wording-only review: B1 and B2 core make B algorithms impossible,3 invalidates a safety invariant and QA proof, B4 destabilizes the extension boundary, B5 breaks migrations/event inserts, and B6 double risksject-ing memory into ACP providers with native context. management Track §5 concerns 1, 2,4 3 , and 7 as explicit task-list subitems; track concerns 5 and 8 in Slice1 QA/retros withpective metrics; concern defer 6 to a web follow-up TechSpec unless the generated include tasks UI implementation beyond the minimum inspector/decision-history surface.
