# Memory v2 Real-Scenario QA Charter

## Scenario

This task validates Memory v2 Slice 1 in a fresh AGH QA lab:

- Lab root: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-20260505-221648-724447-lab`
- Runtime home: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-20260505-221648-724447-lab/.agh/runtime`
- API target: `http://127.0.0.1:60156`
- UDS socket: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-20260505-221648-724447-lab/.agh/runtime/aghd.sock`

## Operator Intent

The operator configures and uses Memory v2 through supported public surfaces, then
verifies that controller-backed writes are searchable immediately, durable state
matches the controller/WAL contract, UI surfaces render truthful daemon state, and
generated docs/reference files match the final hard-cut surface.

## Highest-Risk Probe

TC-SCEN-001 is the first P0 scenario: a workspace memory written through the CLI
must be immediately searchable through CLI, HTTP, and UDS without running `agh
memory reindex`.

## Live Provider Boundary

This pass uses daemon-controlled local runtime surfaces. Provider-backed LLM
sessions are attempted only when reachable from the isolated provider home. If the
native provider boundary is unavailable, the verification report records the
blocker and uses the repository E2E harness plus public daemon surfaces as the
reachable proof.

