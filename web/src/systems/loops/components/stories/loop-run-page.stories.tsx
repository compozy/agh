import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import type { LoopRunLiveState } from "../../lib/loop-events";
import { buildRunMeters } from "../../lib/loop-run-meters";
import { buildRunTimeline, latestGenerationBreadth } from "../../lib/loop-timeline";
import { isTerminalLoopStatus } from "../../lib/loop-formatters";
import { loopDetailByName, loopRunDetailByRunId } from "../../mocks/fixtures";
import type { LoopRunRecord } from "../../types";
import { LoopApprovalGate } from "../run-page/loop-approval-gate";
import { LoopGenerationTimeline } from "../run-page/loop-generation-timeline";
import { LoopRunContractHeader } from "../run-page/loop-run-contract-header";
import { LoopRunControls } from "../run-page/loop-run-controls";
import { LoopRunEventsRail } from "../run-page/loop-run-events-rail";
import { LoopRunFacts } from "../run-page/loop-run-facts";
import { LoopRunMeters } from "../run-page/loop-run-meters";
import { LoopStatusLegend } from "../run-page/loop-status-legend";

const deliveryDefinition = loopDetailByName.get("software-delivery")!.definition;
const watchDefinition = loopDetailByName.get("reviews-watch")!.definition;
const deliveryDetail = loopRunDetailByRunId.get("looprun_running")!;
const watchDetail = loopRunDetailByRunId.get("looprun_watching")!;

const liveState: LoopRunLiveState = {
  events: [
    {
      seq: 8,
      at: "2026-07-06T14:38:00Z",
      kind: "node_running",
      tone: "neutral",
      message: "execute_task · b3",
    },
    {
      seq: 7,
      at: "2026-07-06T14:36:00Z",
      kind: "gate_verdict",
      tone: "err",
      message: "review → revise",
    },
    {
      seq: 6,
      at: "2026-07-06T14:35:00Z",
      kind: "node_failed",
      tone: "err",
      message: "execute_task · b3",
    },
    {
      seq: 5,
      at: "2026-07-06T14:31:00Z",
      kind: "node_succeeded",
      tone: "ok",
      message: "execute_task · b2",
    },
    {
      seq: 4,
      at: "2026-07-06T14:27:00Z",
      kind: "generation_started",
      tone: "warn",
      message: "gen 2 · failed-only",
    },
  ],
  channelMessages: [
    {
      id: "m1",
      author: "implementer",
      role: "run-agent",
      text: "Fixed 7 of 10; 3 documented invalid.",
      isResult: false,
    },
    {
      id: "m2",
      author: "reviewer",
      role: "agent-judge",
      text: "Two issues still unhandled in batch 3.",
      isResult: false,
    },
    {
      id: "m3",
      author: "coordinator",
      role: "result",
      text: "Route batch 3 back for a failed-only re-run.",
      isResult: true,
    },
  ],
  gateVerdicts: {
    review: {
      nodeId: "review",
      generation: 1,
      verdict: "revise",
      confidence: 0.91,
      reason: "A batch with unhandled ids is a failed batch; no-progress window 1 of 2.",
      route: "revise",
      criteria: [
        { id: "all_issues_handled", type: "agent-judge", status: "revise", note: "agent reviewer" },
      ],
      blockingIssues: [
        { id: "issue_022", note: "no triage decision in the batch 3 harvest" },
        { id: "issue_024", note: "triaged valid but no fix landed" },
      ],
    },
  },
  needsApproval: {
    gateId: "approve",
    title: "Approve merge to main?",
    prompt: "The delivery loop reached its human gate before targeting main.",
    facts: [
      { label: "Branch", value: "main" },
      { label: "Diff", value: "+412 / −86" },
      { label: "Tests", value: "project checks · pass" },
    ],
  },
  tokensUsed: null,
};

interface RunPageProps {
  run: LoopRunRecord;
  definition: typeof deliveryDefinition;
  generations: typeof deliveryDetail.generations;
  live: LoopRunLiveState;
}

function RunPage({ run, definition, generations, live }: RunPageProps) {
  const isLive = !isTerminalLoopStatus(run.status);
  const timeline = buildRunTimeline(generations, definition);
  const meters = buildRunMeters(run, latestGenerationBreadth(generations));
  return (
    <StorySurface className="flex h-[900px] p-0">
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto">
        <LoopRunContractHeader
          run={run}
          contract={definition.contract}
          controls={
            <LoopRunControls
              status={run.status}
              pauseRequested={run.pause_requested}
              onPause={() => undefined}
              onResume={() => undefined}
              onStop={() => undefined}
            />
          }
          meters={<LoopRunMeters meters={meters} />}
        />
        <div className="flex flex-col gap-4 px-6 py-5">
          {run.status === "needs-approval" ? (
            <LoopApprovalGate run={run} request={live.needsApproval} onDecision={() => undefined} />
          ) : null}
          <LoopGenerationTimeline
            generations={timeline}
            gateVerdicts={live.gateVerdicts}
            channelMessages={live.channelMessages}
            isLive={isLive}
          />
        </div>
      </div>
      <aside className="w-[332px] shrink-0 overflow-y-auto border-l border-line bg-canvas-soft">
        <LoopRunEventsRail events={live.events} isLive={isLive} />
        <LoopRunFacts run={run} loopVersion={4} concurrency="forbid" />
        <LoopStatusLegend />
      </aside>
    </StorySurface>
  );
}

const meta: Meta<typeof RunPage> = {
  title: "systems/loops/components/LoopRunPage",
  component: RunPage,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Running: Story = {
  render: () => (
    <RunPage
      run={deliveryDetail.run}
      definition={deliveryDefinition}
      generations={deliveryDetail.generations}
      live={liveState}
    />
  ),
};

export const NeedsApproval: Story = {
  render: () => (
    <RunPage
      run={{ ...deliveryDetail.run, status: "needs-approval" }}
      definition={deliveryDefinition}
      generations={deliveryDetail.generations}
      live={liveState}
    />
  ),
};

export const WatchChannel: Story = {
  render: () => (
    <RunPage
      run={watchDetail.run}
      definition={watchDefinition}
      generations={watchDetail.generations}
      live={liveState}
    />
  ),
};

export const TerminalExhausted: Story = {
  render: () => (
    <RunPage
      run={{ ...deliveryDetail.run, status: "exhausted" }}
      definition={deliveryDefinition}
      generations={deliveryDetail.generations}
      live={{ ...liveState, events: [] }}
    />
  ),
};
