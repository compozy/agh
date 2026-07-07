import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
      <a
        href={typeof to === "string" ? to : "#"}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
      >
        {children as React.ReactNode}
      </a>
    ),
  };
});

const { LoopDetailView } = await import("../detail/loop-detail");
const { loopCatalogFixtures, loopDetailByName, loopRunFixtures } =
  await import("../../mocks/fixtures");
const { readLoopGraph } = await import("../../lib/loop-graph");
type LoopBindingRow = import("../../lib/loop-bindings").LoopBindingRow;

const loop = loopDetailByName.get("software-delivery")!;
const catalogEntry = loopCatalogFixtures.find(entry => entry.name === "software-delivery")!;
const recentRuns = loopRunFixtures.filter(run => run.loop_name === "software-delivery").slice(0, 5);
const bindings: LoopBindingRow[] = [
  { id: "job", name: "nightly", kind: "schedule", enabled: false, meta: "Cron 0 3 * * *" },
];

function renderDetail(handlers: Partial<Record<string, () => void>> = {}) {
  const noop = vi.fn();
  const merged = {
    onBack: noop,
    onRun: noop,
    onConfigure: noop,
    onFork: noop,
    onAddTrigger: noop,
    onAddSchedule: noop,
    ...handlers,
  };
  render(
    <LoopDetailView
      loop={loop}
      graph={readLoopGraph(loop.definition)}
      recentRuns={recentRuns}
      bindings={bindings}
      bindingsLoading={false}
      successRate={catalogEntry.success_rate_30d}
      aggregate={catalogEntry.aggregate_30d}
      {...merged}
    />
  );
  return merged;
}

describe("LoopDetailView", () => {
  it("Should render the full definition page: header, contract, DAG, runs, and the right rail", () => {
    renderDetail();
    expect(screen.getByRole("heading", { name: "software-delivery" })).toBeInTheDocument();
    expect(screen.getByTestId("loop-contract")).toBeInTheDocument();
    expect(screen.getByTestId("loop-dag")).toBeInTheDocument();
    expect(screen.getByTestId("loop-recent-runs")).toBeInTheDocument();
    expect(screen.getByTestId("loop-declared-inputs")).toBeInTheDocument();
    expect(screen.getByTestId("loop-start-bindings")).toBeInTheDocument();
    expect(screen.getByTestId("loop-limits")).toBeInTheDocument();
    expect(screen.getByTestId("loop-versions")).toBeInTheDocument();
    expect(screen.getByTestId("loop-stats")).toBeInTheDocument();
  });

  it("Should render the 8-node read-only body graph in order", () => {
    renderDetail();
    const nodes = screen.getAllByTestId("loop-dag-node").map(node => node.dataset.nodeId);
    expect(nodes).toEqual([
      "slug",
      "load_tasks",
      "implement",
      "execute_task",
      "collect",
      "review",
      "verify",
      "approve",
    ]);
  });

  it("Should render the six terminal outcomes and the 30d stats truthfully", () => {
    renderDetail();
    expect(screen.getAllByTestId("loop-terminal-chip")).toHaveLength(6);
    const stats = screen.getByTestId("loop-stats");
    expect(stats).toHaveTextContent("Success rate");
    expect(stats).toHaveTextContent("90%");
    expect(stats).toHaveTextContent("Total runs");
  });

  it("Should invoke the header actions", () => {
    const onRun = vi.fn();
    const onConfigure = vi.fn();
    const onFork = vi.fn();
    renderDetail({ onRun, onConfigure, onFork });
    fireEvent.click(screen.getByTestId("loop-run-action"));
    fireEvent.click(screen.getByTestId("loop-configure-action"));
    fireEvent.click(screen.getByTestId("loop-fork-action"));
    expect(onRun).toHaveBeenCalledTimes(1);
    expect(onConfigure).toHaveBeenCalledTimes(1);
    expect(onFork).toHaveBeenCalledTimes(1);
  });

  it("Should render the version without an unowned publish-state suffix", () => {
    renderDetail();
    expect(screen.getAllByText("v4").length).toBeGreaterThan(0);
    expect(screen.queryByText(/published/i)).not.toBeInTheDocument();
  });
});
