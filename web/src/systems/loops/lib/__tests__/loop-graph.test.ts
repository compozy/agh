import { describe, expect, it } from "vitest";

import { loopDetailByName } from "../../mocks/fixtures";
import type { LoopDefinition } from "../../types";
import { fanOutSummary, nodeClassLabel, readLoopGraph } from "../loop-graph";

const definition = loopDetailByName.get("software-delivery")!.definition;

describe("loop-graph", () => {
  it("Should project the daemon graph (opaque in OpenAPI) into typed nodes and edges", () => {
    const graph = readLoopGraph(definition);
    expect(graph.nodes).toHaveLength(8);
    expect(graph.edges).toHaveLength(7);
    const fanOut = graph.nodes.find(node => node.id === "implement");
    expect(fanOut).toMatchObject({
      nodeClass: "control",
      kind: "fan-out",
      batchSize: 1,
      maxParallel: 1,
      maxFanOut: 64,
    });
    const gate = graph.nodes.find(node => node.id === "review");
    expect(gate?.isGate).toBe(true);
  });

  it("Should drop unreadable nodes and edges rather than surface empty rows", () => {
    const malformed = {
      graph: {
        nodes: [{ id: "ok", class: "action", kind: "run-agent" }, { class: "action" }, 42, null],
        edges: [{ from: "ok", to: "next" }, { from: "" }, "bad"],
      },
    } as unknown as Pick<LoopDefinition, "graph">;
    const graph = readLoopGraph(malformed);
    expect(graph.nodes).toHaveLength(1);
    expect(graph.nodes[0].id).toBe("ok");
    expect(graph.edges).toHaveLength(1);
  });

  it("Should accept a missing graph without throwing", () => {
    expect(readLoopGraph({ graph: undefined } as unknown as Pick<LoopDefinition, "graph">)).toEqual(
      {
        nodes: [],
        edges: [],
      }
    );
  });

  it("Should label node class neutrally, tinting only gate/fan-out sublabels", () => {
    const graph = readLoopGraph(definition);
    const fanOut = graph.nodes.find(node => node.id === "implement")!;
    const gate = graph.nodes.find(node => node.id === "review")!;
    const source = graph.nodes.find(node => node.id === "slug")!;
    expect(nodeClassLabel(fanOut)).toBe("control · fan-out");
    expect(nodeClassLabel(gate)).toBe("control · gate");
    expect(nodeClassLabel(source)).toBe("source");
  });

  it("Should summarize fan-out knobs, marking sequential and unbounded execution", () => {
    const graph = readLoopGraph(definition);
    const fanOut = graph.nodes.find(node => node.id === "implement")!;
    expect(fanOutSummary(fanOut)).toBe("batch 1 · seq · ≤64");
    const source = graph.nodes.find(node => node.id === "slug")!;
    expect(fanOutSummary(source)).toBeNull();
  });
});
