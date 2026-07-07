import { http, HttpResponse, type HttpHandler } from "msw";

import {
  loopAnnotationsFixture,
  loopCatalogFixtures,
  loopConfigFixture,
  loopDetailByName,
  loopRunAggregatesFixture,
  loopRunDetailByRunId,
  loopRunFixtures,
} from "./fixtures";

const catalogByName = new Map(loopCatalogFixtures.map(entry => [entry.name, entry]));

const FAN_OUT_CEILING = 64;

interface MockLintIssue {
  node_id?: string;
  code: string;
  message: string;
  severity: string;
}

/** Detects the first node caught in an edge cycle, mirroring the daemon acyclicity check. */
function firstCycleNode(edges: { from: string; to: string }[]): string | null {
  const adjacency = new Map<string, string[]>();
  for (const edge of edges) {
    const list = adjacency.get(edge.from) ?? [];
    list.push(edge.to);
    adjacency.set(edge.from, list);
  }
  const visiting = new Set<string>();
  const done = new Set<string>();
  let found: string | null = null;
  const walk = (node: string) => {
    if (found || done.has(node)) return;
    visiting.add(node);
    for (const next of adjacency.get(node) ?? []) {
      if (visiting.has(next)) {
        found = next;
        return;
      }
      walk(next);
    }
    visiting.delete(node);
    done.add(node);
  };
  for (const node of adjacency.keys()) walk(node);
  return found;
}

/**
 * A faithful stand-in for the shared Go linter over the posted definition: it flags any
 * fan-out node whose `max_fan_out` exceeds the daemon ceiling and any acyclicity break,
 * returning the same `{ node_id, code, message, severity }` per-node shape the daemon
 * emits (ADR-023). This makes the editor's validate → 422 → Publish gate real in tests.
 */
function lintDefinition(graph?: { nodes?: unknown[]; edges?: unknown[] }): MockLintIssue[] {
  const issues: MockLintIssue[] = [];
  const nodes = Array.isArray(graph?.nodes) ? (graph!.nodes as Record<string, unknown>[]) : [];
  for (const node of nodes) {
    const fanOut = node.max_fan_out;
    if (typeof fanOut === "number" && fanOut > FAN_OUT_CEILING) {
      issues.push({
        node_id: String(node.id),
        code: "fan_out_ceiling_exceeded",
        message: `max_fan_out (${fanOut}) exceeds the daemon ceiling of ${FAN_OUT_CEILING}.`,
        severity: "error",
      });
    }
  }
  const edges = Array.isArray(graph?.edges)
    ? (graph!.edges as { from: string; to: string }[]).filter(edge => edge?.from && edge?.to)
    : [];
  const cycleNode = firstCycleNode(edges);
  if (cycleNode) {
    issues.push({
      node_id: cycleNode,
      code: "cycle_detected",
      message: `The graph is not acyclic: ${cycleNode} is part of a cycle.`,
      severity: "error",
    });
  }
  return issues;
}

export const handlers: HttpHandler[] = [
  http.get("/api/workspaces/:workspaceId/loops", () =>
    HttpResponse.json({ loops: loopCatalogFixtures })
  ),
  http.post("/api/workspaces/:workspaceId/loops", () =>
    HttpResponse.json({ loop: loopDetailByName.get("software-delivery") }, { status: 201 })
  ),
  http.get("/api/workspaces/:workspaceId/loops/:name", ({ params }) => {
    const detail = loopDetailByName.get(String(params.name));
    if (!detail) {
      return HttpResponse.json(
        { error: `Loop not found: ${String(params.name)}` },
        { status: 404 }
      );
    }
    return HttpResponse.json({ loop: detail });
  }),
  http.patch("/api/workspaces/:workspaceId/loops/:name", async ({ params, request }) => {
    const detail = loopDetailByName.get(String(params.name));
    if (!detail) {
      return HttpResponse.json(
        { error: `Loop not found: ${String(params.name)}` },
        { status: 404 }
      );
    }
    const body = (await request.json().catch(() => ({}))) as {
      definition?: Record<string, unknown>;
      expected_version?: number | null;
    };
    // Echo the published definition with a bumped monotonic meta.version (§9.13);
    // the store is not mutated so parallel tests stay isolated.
    const published = body.definition ?? detail.definition;
    const meta = (published.meta as Record<string, unknown>) ?? {};
    const nextVersion = (detail.version ?? 0) + 1;
    return HttpResponse.json({
      loop: {
        ...detail,
        version: nextVersion,
        definition: { ...published, meta: { ...meta, version: nextVersion } },
      },
    });
  }),
  http.delete("/api/workspaces/:workspaceId/loops/:name", ({ params }) => {
    if (!catalogByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Loop not found: ${String(params.name)}` },
        { status: 404 }
      );
    }
    return new HttpResponse(null, { status: 204 });
  }),
  http.get("/api/workspaces/:workspaceId/loops/:name/config", ({ params }) => {
    if (!catalogByName.has(String(params.name))) {
      return HttpResponse.json(
        { error: `Loop not found: ${String(params.name)}` },
        { status: 404 }
      );
    }
    return HttpResponse.json({ config: loopConfigFixture });
  }),
  http.put("/api/workspaces/:workspaceId/loops/:name/config", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { config?: unknown };
    return HttpResponse.json({ config: body.config ?? loopConfigFixture });
  }),
  http.get("/api/workspaces/:workspaceId/loops/:name/annotations", () =>
    HttpResponse.json({ annotations: loopAnnotationsFixture })
  ),
  http.put("/api/workspaces/:workspaceId/loops/:name/annotations", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { annotations?: unknown };
    return HttpResponse.json({ annotations: body.annotations ?? loopAnnotationsFixture });
  }),
  http.post("/api/workspaces/:workspaceId/loops/:name/run", ({ request, params }) => {
    const url = new URL(request.url);
    const name = String(params.name);
    const entry = catalogByName.get(name);
    if (!entry) {
      return HttpResponse.json({ error: `Loop not found: ${name}` }, { status: 404 });
    }
    const detail = entry.last_run ? loopRunDetailByRunId.get(entry.last_run.id) : undefined;
    if (url.searchParams.get("dry") === "true") {
      return HttpResponse.json({
        dry_run: {
          loop_name: name,
          generation: 1,
          resolved_inputs: {},
          contract: entry.contract,
          nodes: [{ id: "plan", kind: "run-agent", class: "action" }],
          effective_config: {
            iteration_cap: 12,
            budget_tokens: 500_000,
            budget_wall_sec: 3_600,
            budget_on_exceeded: "halt",
            fan_out_width: 4,
            gate_max_revisions: 3,
            human_gate_enabled: true,
            no_progress_window: 3,
            reattempt_strategy: "failed_only",
            enabled_checks_json: null,
          },
        },
      });
    }
    if (!detail) {
      return HttpResponse.json({ error: `Loop run not found for ${name}` }, { status: 404 });
    }
    return HttpResponse.json({ run: detail?.run }, { status: 201 });
  }),
  http.post("/api/workspaces/:workspaceId/loops/:name/validate", async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      definition?: { graph?: { nodes?: unknown[]; edges?: unknown[] } };
    };
    const errors = lintDefinition(body.definition?.graph);
    return HttpResponse.json(
      { valid: errors.length === 0, errors },
      { status: errors.length ? 422 : 200 }
    );
  }),
  http.get("/api/workspaces/:workspaceId/loop-runs", ({ request }) => {
    const url = new URL(request.url);
    const loop = url.searchParams.get("loop");
    const status = url.searchParams.get("status");
    const runs = loopRunFixtures.filter(run => {
      if (loop && run.loop_name !== loop) return false;
      if (status && run.status !== status) return false;
      return true;
    });
    return HttpResponse.json({ runs, aggregates: loopRunAggregatesFixture });
  }),
  http.get("/api/workspaces/:workspaceId/loop-runs/:runId", ({ params }) => {
    const detail = loopRunDetailByRunId.get(String(params.runId));
    if (!detail) {
      return HttpResponse.json(
        { error: `Loop run not found: ${String(params.runId)}` },
        { status: 404 }
      );
    }
    return HttpResponse.json(detail);
  }),
  http.post("/api/workspaces/:workspaceId/loop-runs/:runId/approve", () =>
    HttpResponse.json({ ok: true })
  ),
  http.post("/api/workspaces/:workspaceId/loop-runs/:runId/pause", () =>
    HttpResponse.json({ ok: true })
  ),
  http.post("/api/workspaces/:workspaceId/loop-runs/:runId/resume", () =>
    HttpResponse.json({ ok: true })
  ),
  http.post("/api/workspaces/:workspaceId/loop-runs/:runId/stop", () =>
    HttpResponse.json({ ok: true })
  ),
];
