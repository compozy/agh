import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames, storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import type { TaskRunDetailView } from "../../types";
import { TaskRunDetailHeader } from "../task-run-detail-header";
import {
  TaskRunActivityPanel,
  TaskRunIdentityPanel,
  TaskRunProgressPanel,
} from "../task-run-detail-panels";

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

function DetailSurface({ run }: { run: TaskRunDetailView }) {
  return (
    <PanelSurface className="min-h-[820px] flex-col p-0">
      <div className="flex min-h-0 flex-1 flex-col">
        <TaskRunDetailHeader onCancelRun={() => undefined} run={run} />
        <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-5">
          <TaskRunIdentityPanel run={run} />
          <TaskRunProgressPanel run={run} />
          <TaskRunActivityPanel run={run} />
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
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "completed",
          ended_at: "2026-04-11T14:45:45Z",
          result: { status: "ok", summary: "Payout release summary posted." },
        },
      } as Partial<TaskRunDetailView>)}
    />
  ),
};

export const Failed: Story = {
  name: "Error",
  render: () => (
    <DetailSurface
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "failed",
          ended_at: "2026-04-11T14:43:00Z",
          error: "partner settlement export returned 429",
        },
      } as Partial<TaskRunDetailView>)}
    />
  ),
};

export const NoSession: Story = {
  name: "Empty",
  render: () => (
    <DetailSurface
      run={buildRun({
        session: null,
        run: {
          ...buildRun().run,
          session_id: undefined,
        },
      } as Partial<TaskRunDetailView>)}
    />
  ),
};

export const Queued: Story = {
  name: "Pending",
  render: () => (
    <DetailSurface
      run={buildRun({
        run: {
          ...buildRun().run,
          status: "queued",
          started_at: null,
        },
      } as Partial<TaskRunDetailView>)}
    />
  ),
};
