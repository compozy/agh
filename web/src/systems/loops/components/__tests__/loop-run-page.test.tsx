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

const { LoopRunContractHeader } = await import("../run-page/loop-run-contract-header");
const { LoopGenerationTimeline } = await import("../run-page/loop-generation-timeline");
const { LoopGenerationCard } = await import("../run-page/loop-generation-card");
const { LoopRunControls } = await import("../run-page/loop-run-controls");
const { LoopApprovalGate } = await import("../run-page/loop-approval-gate");
const { LoopWatchEventsPanel } = await import("../run-page/loop-watch-events-panel");
type LoopWatchEventsState = import("../../types").LoopWatchEventsState;
const { buildRunTimeline } = await import("../../lib/loop-timeline");
type LoopTimelineGeneration = import("../../lib/loop-timeline").LoopTimelineGeneration;
type GoalTurnTimelineItem = import("../../hooks/use-goal-turns").GoalTurnTimelineItem;
const { loopDetailByName, loopRunDetailByRunId } = await import("../../mocks/fixtures");
type LoopRunRecord = import("../../types").LoopRunRecord;

const detail = loopRunDetailByRunId.get("looprun_running")!;
const definition = loopDetailByName.get("software-delivery")!.definition;
const contract = definition.contract;

function run(overrides: Partial<LoopRunRecord> = {}): LoopRunRecord {
  return { ...detail.run, ...overrides };
}

describe("LoopRunContractHeader", () => {
  it("Should render the goal, definition of done, verification rows, and terminal chips", () => {
    render(<LoopRunContractHeader run={run()} contract={contract} />);
    expect(screen.getByTestId("loop-run-goal")).toHaveTextContent(contract.goal);
    expect(screen.getByTestId("loop-run-dod")).toHaveTextContent(contract.definition_of_done);
    expect(screen.getAllByTestId("loop-run-verification-row").length).toBeGreaterThan(0);
    expect(screen.getAllByTestId("loop-run-terminal-chip")).toHaveLength(6);
  });

  it("Should leave live status and operator controls to the OS topbar", () => {
    render(<LoopRunContractHeader run={run({ status: "exhausted" })} contract={contract} />);
    expect(screen.queryByTestId("loop-run-status-pill")).not.toBeInTheDocument();
    expect(screen.queryByTestId("loop-run-controls")).not.toBeInTheDocument();
  });
});

describe("LoopGenerationTimeline", () => {
  const timeline = buildRunTimeline(detail.generations, definition);
  const verdicts = {
    review: {
      nodeId: "review",
      generation: 1,
      verdict: "revise" as const,
      confidence: 0.91,
      route: "revise",
      criteria: [{ id: "all_handled", type: "agent-judge", status: "revise" as const }],
      blockingIssues: [{ id: "issue_022" }],
    },
  };

  it("Should render collapsible generations newest-first with the node spine", () => {
    render(
      <LoopGenerationTimeline
        generations={timeline}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive
      />
    );
    expect(screen.getByTestId("loop-generation-2")).toBeInTheDocument();
    expect(screen.getByTestId("loop-generation-1")).toBeInTheDocument();
  });

  it("Should mark carried-forward nodes and link an awaiting_child node to its child run", () => {
    render(
      <LoopGenerationTimeline
        generations={timeline}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive
      />
    );
    // Fan-out branches carry a per-instance testid suffix (N-003), so target by prefix.
    expect(
      screen.getAllByTestId(/^loop-node-carry-forward-execute_task-\d+$/).length
    ).toBeGreaterThan(0);
    const childLink = screen.getByTestId("loop-node-child-link-child_delivery");
    expect(childLink).toHaveAttribute("data-params", JSON.stringify({ runId: "looprun_child" }));
  });

  it("Should render a gate verdict card with its routing", () => {
    render(
      <LoopGenerationTimeline
        generations={timeline}
        gateVerdicts={verdicts}
        channelMessages={[]}
        isLive
      />
    );
    const cards = screen.getAllByTestId("loop-gate-card");
    expect(cards.length).toBeGreaterThan(0);
    expect(cards[0]).toHaveAttribute("data-verdict", "revise");
    expect(cards[0]).toHaveTextContent("issue_022");
  });

  it("Should render an action failure cause and recovery from the durable output", () => {
    const failedTimeline = buildRunTimeline(
      [
        {
          generation: 1,
          outputs: [
            {
              node_id: "load_tasks",
              status: "failed",
              output_ref: JSON.stringify({
                kind: "action_failure",
                code: "tool_invalid_input",
                cause: "No task set matched .compozy/tasks/helix-v1-launch/task_*.md.",
                recovery:
                  "Create the matching task set or correct the Loop input, then retry the run.",
              }),
            },
          ],
        },
      ],
      definition
    );

    render(
      <LoopGenerationTimeline
        generations={failedTimeline}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive={false}
      />
    );

    expect(
      screen.getByText("No task set matched .compozy/tasks/helix-v1-launch/task_*.md.")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Create the matching task set or correct the Loop input, then retry the run."
      )
    ).toBeInTheDocument();
  });
});

describe("LoopGenerationCard summary", () => {
  function generation(statuses: string[]): LoopTimelineGeneration {
    return {
      generation: 1,
      isLatest: false,
      nodes: statuses.map((status, index) => ({
        nodeId: `n${index}`,
        classLabel: "action",
        kind: "run-agent",
        status,
        tone: "neutral" as const,
        pulse: false,
        isGate: false,
        isCarriedForward: false,
      })),
    };
  }

  it("Should name blocked and failed nodes separately, never lumping blocked into failed (N-001)", () => {
    const { rerender } = render(
      <LoopGenerationCard
        generation={generation(["succeeded", "blocked"])}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive={false}
      />
    );
    const toggle = () => screen.getByTestId("loop-generation-toggle-1");
    expect(toggle()).toHaveTextContent("1 blocked");
    expect(toggle()).not.toHaveTextContent("failed");
    rerender(
      <LoopGenerationCard
        generation={generation(["failed", "blocked"])}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive={false}
      />
    );
    expect(toggle()).toHaveTextContent("1 failed · 1 blocked");
  });

  it("Should nest only the matching Goal turn axis inside a Goal node card", () => {
    const goalGeneration: LoopTimelineGeneration = {
      generation: 2,
      isLatest: true,
      nodes: [
        {
          nodeId: "goal",
          classLabel: "action",
          kind: "goal",
          status: "running",
          tone: "accent",
          pulse: true,
          isGate: false,
          isCarriedForward: false,
          itemIndex: 0,
        },
      ],
    };
    const matching: GoalTurnTimelineItem = {
      key: "goal:0:2:1:1:prompt_1",
      seq: 1,
      generation: 2,
      nodeId: "goal",
      itemIndex: 0,
      turn: 1,
      promptAttempt: 1,
      promptId: "prompt_1",
      sessionId: "session_1",
      resultStatus: null,
      stopReason: null,
      reasonCode: null,
      verdictOutcome: null,
      blockingIssues: [],
      evidenceRef: null,
      tokensUsed: null,
      startedAt: "2026-07-10T12:00:00Z",
      endedAt: null,
    };
    render(
      <LoopGenerationCard
        generation={goalGeneration}
        gateVerdicts={{}}
        channelMessages={[]}
        isLive
        goalTurns={[
          matching,
          { ...matching, key: "other-generation", seq: 2, generation: 1, turn: 99 },
          { ...matching, key: "other-item", seq: 3, itemIndex: 1, turn: 98 },
        ]}
      />
    );
    const timeline = screen.getByRole("region", { name: "Goal turn timeline" });
    expect(timeline).toHaveTextContent("Turn 1");
    expect(timeline).not.toHaveTextContent("Turn 99");
    expect(timeline).not.toHaveTextContent("Turn 98");
  });
});

describe("LoopRunControls", () => {
  it("Should show Pause + Stop while running and fire the callback", () => {
    const onPause = vi.fn();
    render(
      <LoopRunControls status="running" onPause={onPause} onResume={vi.fn()} onStop={vi.fn()} />
    );
    expect(screen.getByTestId("loop-run-pause")).toBeInTheDocument();
    expect(screen.getByTestId("loop-run-stop")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-run-resume")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("loop-run-pause"));
    expect(onPause).toHaveBeenCalledTimes(1);
  });

  it("Should show Resume while paused and render nothing for a terminal run", () => {
    const { rerender } = render(
      <LoopRunControls status="paused" onPause={vi.fn()} onResume={vi.fn()} onStop={vi.fn()} />
    );
    expect(screen.getByTestId("loop-run-resume")).toBeInTheDocument();
    expect(screen.queryByTestId("loop-run-pause")).not.toBeInTheDocument();
    rerender(
      <LoopRunControls status="done" onPause={vi.fn()} onResume={vi.fn()} onStop={vi.fn()} />
    );
    expect(screen.queryByTestId("loop-run-controls")).not.toBeInTheDocument();
  });
});

describe("LoopApprovalGate", () => {
  it("Should route each closed HITL decision correctly", () => {
    const onDecision = vi.fn();
    render(<LoopApprovalGate run={run({ status: "needs-approval" })} onDecision={onDecision} />);
    fireEvent.click(screen.getByTestId("loop-approval-approve"));
    fireEvent.click(screen.getByTestId("loop-approval-request-changes"));
    fireEvent.click(screen.getByTestId("loop-approval-reject"));
    expect(onDecision).toHaveBeenNthCalledWith(1, "approve", "approve");
    expect(onDecision).toHaveBeenNthCalledWith(2, "request_changes", "approve");
    expect(onDecision).toHaveBeenNthCalledWith(3, "reject", "approve");
  });
});

describe("LoopWatchEventsPanel", () => {
  const parked: LoopWatchEventsState = {
    subscriptions: [
      { kind: "task.status_changed", filter: "event.payload.to_status == 'completed'" },
      { kind: "loop.terminal" },
    ],
    cursors: { task_events: 42, loop_run_events: 7 },
    last_wake_at: "2026-07-08T12:00:00Z",
  };

  it("Should render subscriptions, cursors, and last wake from the parked read-model", () => {
    render(<LoopWatchEventsPanel state={parked} />);
    const subscriptions = screen.getAllByTestId("loop-watch-events-subscription");
    expect(subscriptions).toHaveLength(2);
    expect(subscriptions[0]).toHaveTextContent("task.status_changed");
    expect(screen.getByTestId("loop-watch-events-filter")).toHaveTextContent(
      "event.payload.to_status == 'completed'"
    );
    // The unfiltered subscription is truthful about matching every event of its kind.
    expect(subscriptions[1]).toHaveTextContent("matches every event");
    expect(screen.getByTestId("loop-watch-events-cursors")).toHaveTextContent("task_events");
    expect(screen.getByTestId("loop-watch-events-last-wake")).toHaveAttribute(
      "datetime",
      "2026-07-08T12:00:00Z"
    );
  });

  it("Should render nothing when the run is not parked on events (truthful UI)", () => {
    const { container } = render(<LoopWatchEventsPanel state={undefined} />);
    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByTestId("loop-watch-events-panel")).not.toBeInTheDocument();
  });

  it("Should render nothing when the read-model carries no subscriptions", () => {
    render(<LoopWatchEventsPanel state={{ subscriptions: [], cursors: {} }} />);
    expect(screen.queryByTestId("loop-watch-events-panel")).not.toBeInTheDocument();
  });

  it("Should render duplicate subscriptions without React key collisions", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const subscription = { kind: "loop.terminal" as const };

    try {
      render(
        <LoopWatchEventsPanel
          state={{ subscriptions: [subscription, subscription], cursors: {} }}
        />
      );

      expect(screen.getAllByTestId("loop-watch-events-subscription")).toHaveLength(2);
      expect(consoleError).not.toHaveBeenCalled();
    } finally {
      consoleError.mockRestore();
    }
  });
});
