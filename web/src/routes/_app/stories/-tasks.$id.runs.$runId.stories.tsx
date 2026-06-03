import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { storyAgentNames, storyPeople, storySessionIds } from "@/storybook/fintech-scenario";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story-meta";
import { buildTaskRunDetailFixture } from "@/systems/tasks/mocks";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/tasks/run-detail",
    "Task run-detail route stories rendered inside the persistent tasks shell, covering live, terminal, sessionless, loading, and not-found branches."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default running run-detail route with live session linkage and progress metrics.
 */
export const Running: Story = {
  args: {},
  parameters: appRouteParameters("/tasks/task_001/runs/run_001"),
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Completed run-detail route with a result payload surfaced in the activity panel.
 */
export const Completed: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001/runs/run_001"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/task-runs/:id", () =>
          HttpResponse.json({
            run: buildTaskRunDetailFixture({
              run: {
                id: "run_001",
                task_id: "task_001",
                attempt: 2,
                status: "completed",
                queued_at: "2026-04-17T09:58:00Z",
                started_at: "2026-04-17T09:59:00Z",
                ended_at: "2026-04-17T10:03:00Z",
                origin: { kind: "cli", ref: storyPeople.primaryOperator },
                session_id: storySessionIds.fraud,
                idempotency_key: "payout-review-run",
                claimed_by: { kind: "agent_session", ref: storyAgentNames.fraud },
                result: { status: "ok", summary: "Payout release summary posted." },
              },
            }),
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Failed run-detail route showing the run error state.
 */
export const Failed: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001/runs/run_001"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/task-runs/:id", () =>
          HttpResponse.json({
            run: buildTaskRunDetailFixture({
              run: {
                id: "run_001",
                task_id: "task_001",
                attempt: 3,
                status: "failed",
                queued_at: "2026-04-17T09:58:00Z",
                started_at: "2026-04-17T09:59:00Z",
                ended_at: "2026-04-17T10:02:00Z",
                origin: { kind: "cli", ref: storyPeople.primaryOperator },
                session_id: storySessionIds.fraud,
                idempotency_key: "payout-review-run",
                claimed_by: { kind: "agent_session", ref: storyAgentNames.fraud },
                error: "partner settlement export returned 429",
              },
            }),
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Sessionless run-detail route where the run exists but has no linked session yet.
 */
export const NoSession: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001/runs/run_001"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/task-runs/:id", () =>
          HttpResponse.json({
            run: buildTaskRunDetailFixture({
              run: {
                id: "run_001",
                task_id: "task_001",
                attempt: 1,
                status: "queued",
                queued_at: "2026-04-17T10:04:00Z",
                started_at: null,
                origin: { kind: "cli", ref: storyPeople.primaryOperator },
                session_id: undefined,
              },
              session: null,
            }),
          })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Loading branch while the run detail payload is still in flight.
 */
export const Loading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001/runs/run_001"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/task-runs/:id", async () => {
          await delay("infinite");
          return HttpResponse.json({ run: null });
        }),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};

/**
 * Not-found branch for a missing run id while the tasks shell remains mounted.
 */
export const NotFound: Story = {
  args: {},
  parameters: {
    ...appRouteParameters("/tasks/task_001/runs/run_missing"),
    ...storybookMswParameters({
      tasks: [
        http.get("/api/task-runs/:id", ({ params }) =>
          HttpResponse.json({ error: `Task run not found: ${String(params.id)}` }, { status: 404 })
        ),
      ],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
};
