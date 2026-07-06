import { describe, expect, it } from "vitest";

import { buildRunTimeline, latestGenerationBreadth } from "../loop-timeline";
import { loopDetailByName, loopRunDetailByRunId } from "../../mocks/fixtures";

const definition = loopDetailByName.get("software-delivery")!.definition;
const generations = loopRunDetailByRunId.get("looprun_running")!.generations!;

describe("loop-timeline model", () => {
  it("Should order generations newest-first and flag the latest", () => {
    const timeline = buildRunTimeline(generations, definition);
    expect(timeline.map(gen => gen.generation)).toEqual([2, 1]);
    expect(timeline[0].isLatest).toBe(true);
    expect(timeline[1].isLatest).toBe(false);
  });

  it("Should enrich each node with its class label and kind from the definition graph", () => {
    const timeline = buildRunTimeline(generations, definition);
    const g1 = timeline.find(gen => gen.generation === 1)!;
    const execute = g1.nodes.find(node => node.nodeId === "execute_task")!;
    expect(execute.kind).toBe("run-agent");
    expect(execute.classLabel).toBe("action");
    const review = g1.nodes.find(node => node.nodeId === "review")!;
    expect(review.isGate).toBe(true);
    expect(review.classLabel).toBe("control · gate");
  });

  it("Should mark carried-forward (reused) nodes read-only and never coerce their status", () => {
    const timeline = buildRunTimeline(generations, definition);
    const g2 = timeline.find(gen => gen.generation === 2)!;
    const reused = g2.nodes.filter(node => node.status === "reused");
    expect(reused.length).toBeGreaterThan(0);
    expect(reused.every(node => node.isCarriedForward)).toBe(true);
  });

  it("Should surface an awaiting_child node with its child run id and info tone", () => {
    const timeline = buildRunTimeline(generations, definition);
    const g2 = timeline.find(gen => gen.generation === 2)!;
    const awaiting = g2.nodes.find(node => node.status === "awaiting_child")!;
    expect(awaiting.childLoopRunId).toBe("looprun_child");
    expect(awaiting.tone).toBe("info");
    expect(awaiting.pulse).toBe(true);
  });

  it("Should compute the latest generation's materialized breadth from distinct item_index", () => {
    expect(latestGenerationBreadth(generations)).toBe(3);
    expect(latestGenerationBreadth([])).toBe(0);
  });
});
