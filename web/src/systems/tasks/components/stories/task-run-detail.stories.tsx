import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames, storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import type { TaskRunDetailView, TaskTimelineItem } from "../../types";
import { TaskRunDetailHeader } from "../task-run-detail-header";
import { TaskRunTimelinePanel } from "../task-run-timeline-panel";

const meta: Meta = {
  title: "systems/tasks/TaskRunDetail",
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function buildRun(overrides: Partial<TaskRunDetailView> = {}): TaskRunDetailView {
  return {
    run: {
      id: "run_7k2m9x",
      task_id: "task_001",
      attempt: 2,
      status: "running",
      queued_at: "2026-04-11T14:30:00Z",
      started_at: "2026-04-11T14:37:45Z",
      origin: { kind: "cli", ref: "op" },
      session_id: "sess_jf8d21",
      idempotency_key: "payout-review-run",
      claimed_by: { kind: "agent_session", ref: storyAgentNames.fraud },
    },
    task: {
      id: "task_001",
      identifier: "TASK-42",
      status: "in_progress",
      scope: "workspace",
      title: "Review payout holds for top LATAM merchants",
    },
    summary: {
      last_activity_at: "2026-04-11T14:40:45Z",
      last_event_type: "task.run_progress",
      tool_call_count: 4,
      input_tokens: 14281,
      output_tokens: 3046,
      total_tokens: 17327,
      turn_count: 6,
      total_cost: 0.18,
      cost_currency: "USD",
    },
    session: {
      session_id: "sess_jf8d21",
      created_at: "2026-04-11T14:30:00Z",
      updated_at: "2026-04-11T14:40:45Z",
      agent_name: storyAgentNames.fraud,
      workspace_id: storyDefaultWorkspaceId,
      state: "active",
    },
    ...overrides,
  } as unknown as TaskRunDetailView;
}

const sampleEvents: TaskTimelineItem[] = [
  {
    event_id: "evt_001",
    sequence: 12,
    event_type: "task.run_enqueued",
    timestamp: "2026-04-11T14:30:00Z",
    payload: { message: "Run queued by operator" },
    run: { id: "run_7k2m9x", attempt: 2, status: "queued" },
    origin: { kind: "cli", ref: "op" },
    task: { id: "task_001", identifier: "TASK-42" },
  },
  {
    event_id: "evt_002",
    sequence: 13,
    event_type: "task.run_claimed",
    timestamp: "2026-04-11T14:35:00Z",
    payload: undefined,
    run: { id: "run_7k2m9x", attempt: 2, status: "claimed" },
    origin: { kind: "cli", ref: "op" },
    task: { id: "task_001", identifier: "TASK-42" },
  },
  {
    event_id: "evt_003",
    sequence: 14,
    event_type: "task.run_started",
    timestamp: "2026-04-11T14:37:45Z",
    payload: undefined,
    run: { id: "run_7k2m9x", attempt: 2, status: "running" },
    origin: { kind: "cli", ref: "op" },
    task: { id: "task_001", identifier: "TASK-42" },
  },
  {
    event_id: "evt_004",
    sequence: 15,
    event_type: "task.run_progress",
    timestamp: "2026-04-11T14:40:45Z",
    payload: { message: "Resolved 18 merchant accounts; remaining 12." },
    run: { id: "run_7k2m9x", attempt: 2, status: "running" },
    origin: { kind: "cli", ref: "op" },
    task: { id: "task_001", identifier: "TASK-42" },
  },
] as unknown as TaskTimelineItem[];

interface DetailSurfaceProps {
  run: TaskRunDetailView;
  items?: TaskTimelineItem[];
  isLive?: boolean;
}

function DetailSurface({ run, items = sampleEvents, isLive = true }: DetailSurfaceProps) {
  return (
    <PanelSurface className="min-h-[820px] flex-col p-0">
      <div className="flex min-h-0 flex-1 flex-col">
        <TaskRunDetailHeader
          onCancelRun={() => undefined}
          onForceFailRun={() => undefined}
          onForceReleaseRun={() => undefined}
          onRetryRun={() => undefined}
          run={run}
        />
        <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5">
          <TaskRunTimelinePanel isLive={isLive} items={items} run={run} />
        </div>
      </div>
    </PanelSurface>
  );
}

export const Running: Story = {
  name: "Populated",
  render: () => <DetailSurface run={buildRun()} />,
};

export const Completed: Story = {
  render: () => (
    <DetailSurface
      isLive={false}
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "completed",
          ended_at: "2026-04-11T14:45:45Z",
          result: { status: "ok", summary: "Payout release summary posted." },
        },
      } as Partial<TaskRunDetailView>)}
      items={[
        ...sampleEvents,
        {
          event_id: "evt_005",
          sequence: 16,
          event_type: "task.run_completed",
          timestamp: "2026-04-11T14:45:45Z",
          payload: undefined,
          run: { id: "run_7k2m9x", attempt: 2, status: "completed" },
          origin: { kind: "cli", ref: "op" },
          task: { id: "task_001", identifier: "TASK-42" },
        } as unknown as TaskTimelineItem,
      ]}
    />
  ),
};

export const Claimed: Story = {
  render: () => (
    <DetailSurface
      isLive={false}
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "claimed",
          started_at: null,
        },
      } as Partial<TaskRunDetailView>)}
      items={sampleEvents.slice(0, 2)}
    />
  ),
};

export const Failed: Story = {
  name: "Error",
  render: () => (
    <DetailSurface
      isLive={false}
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "failed",
          ended_at: "2026-04-11T14:43:00Z",
          error: "partner settlement export returned 429",
        },
      } as Partial<TaskRunDetailView>)}
      items={[
        ...sampleEvents,
        {
          event_id: "evt_006",
          sequence: 17,
          event_type: "task.run_failed",
          timestamp: "2026-04-11T14:43:00Z",
          payload: { message: "partner settlement export returned 429" },
          run: {
            id: "run_7k2m9x",
            attempt: 2,
            status: "failed",
            error: "partner settlement export returned 429",
          },
          origin: { kind: "cli", ref: "op" },
          task: { id: "task_001", identifier: "TASK-42" },
        } as unknown as TaskTimelineItem,
      ]}
    />
  ),
};

/**
 * Needs-attention run detail with the scheduler diagnostic visible on the run card and timeline.
 */
export const NeedsAttention: Story = {
  args: {},
  render: () => (
    <DetailSurface
      isLive={false}
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "needs_attention",
          started_at: null,
          session_id: undefined,
          claimed_by: undefined,
          error: "No capable agent claimed this run before escalation.",
        },
        session: undefined,
      } as Partial<TaskRunDetailView>)}
      items={[
        sampleEvents[0],
        {
          event_id: "evt_007",
          sequence: 18,
          event_type: "task.run_starved",
          timestamp: "2026-04-11T14:44:00Z",
          payload: undefined,
          run: { id: "run_7k2m9x", attempt: 2, status: "queued" },
          origin: { kind: "scheduler", ref: "convergence" },
          task: { id: "task_001", identifier: "TASK-42" },
        } as unknown as TaskTimelineItem,
        {
          event_id: "evt_008",
          sequence: 19,
          event_type: "task.run_needs_attention",
          timestamp: "2026-04-11T14:46:00Z",
          payload: { diagnostic: "No capable agent claimed this run before escalation." },
          run: {
            id: "run_7k2m9x",
            attempt: 2,
            status: "needs_attention",
            error: "No capable agent claimed this run before escalation.",
          },
          origin: { kind: "scheduler", ref: "convergence" },
          task: { id: "task_001", identifier: "TASK-42" },
        } as unknown as TaskTimelineItem,
      ]}
    />
  ),
};

export const Empty: Story = {
  name: "Empty",
  render: () => <DetailSurface isLive={false} items={[]} run={buildRun()} />,
};

export const Queued: Story = {
  name: "Pending",
  render: () => (
    <DetailSurface
      isLive={false}
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "queued",
          started_at: null,
        },
      } as Partial<TaskRunDetailView>)}
      items={sampleEvents.slice(0, 1)}
    />
  ),
};
