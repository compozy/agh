import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { UIProvider } from "@agh/ui";

import { TasksListSurface } from "../tasks-list-surface";
import type { TasksListSurfaceProps } from "../tasks-list-surface";
import type { TaskFilterOwnerOption } from "../../lib/tasks-list-filters";
import { buildTaskFixture } from "./fixtures";

const FIXTURE_TASKS = [
  buildTaskFixture({
    id: "task_a",
    status: "in_progress",
    title: "Investigate auth middleware leak",
  }),
  buildTaskFixture({
    id: "task_b",
    status: "ready",
    title: "Wire daemon /healthz to status footer",
    priority: "medium",
  }),
  buildTaskFixture({
    id: "task_c",
    status: "blocked",
    title: "Land RFC-002 protocol receipt schema",
    priority: "urgent",
  }),
  buildTaskFixture({
    id: "task_d",
    status: "pending",
    title: "Define agent execution profile defaults",
    priority: "medium",
  }),
  buildTaskFixture({
    id: "task_e",
    status: "completed",
    title: "Patch SQLite migration registry race on first boot",
    priority: "high",
  }),
  buildTaskFixture({
    id: "task_f",
    status: "failed",
    title: "Backfill task_runs.queue_position for legacy rows",
    priority: "high",
  }),
];

const OWNER_OPTIONS: TaskFilterOwnerOption[] = [
  { ref: "Coder", kind: "agent_session" },
  { ref: "Reviewer", kind: "agent_session" },
  { ref: "pedro@", kind: "human" },
];

function Stateful(props: Partial<TasksListSurfaceProps>) {
  const [statusFilter, setStatusFilter] = useState<TasksListSurfaceProps["statusFilter"]>(null);
  const [ownerFilter, setOwnerFilter] = useState<TasksListSurfaceProps["ownerFilter"]>(null);
  const [priorityFilter, setPriorityFilter] =
    useState<TasksListSurfaceProps["priorityFilter"]>(null);
  const [scopeFilter, setScopeFilter] = useState<TasksListSurfaceProps["scopeFilter"]>("all");
  const [sortBy, setSortBy] = useState<TasksListSurfaceProps["sortBy"]>("recent");
  const [searchQuery, setSearchQuery] = useState("");

  return (
    <UIProvider reducedMotion="always">
      <div className="flex h-screen flex-col bg-canvas">
        <TasksListSurface
          listUpdatedAt={Date.now() - 120_000}
          onOwnerChange={setOwnerFilter}
          onPriorityChange={setPriorityFilter}
          onScopeChange={setScopeFilter}
          onSearchQueryChange={setSearchQuery}
          onSelectTask={() => {}}
          onSortChange={setSortBy}
          onStatusChange={setStatusFilter}
          ownerFilter={ownerFilter}
          ownerOptions={OWNER_OPTIONS}
          priorityFilter={priorityFilter}
          searchQuery={searchQuery}
          scopeFilter={scopeFilter}
          sortBy={sortBy}
          statusFilter={statusFilter}
          tasks={FIXTURE_TASKS}
          totalCount={FIXTURE_TASKS.length}
          workspaceName="agh-runtime"
          {...props}
        />
      </div>
    </UIProvider>
  );
}

const meta: Meta<typeof TasksListSurface> = {
  title: "systems/tasks/TasksListSurface",
  component: TasksListSurface,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof TasksListSurface>;

export const Default: Story = {
  render: () => <Stateful />,
};

export const Loading: Story = {
  render: () => <Stateful isLoading tasks={[]} totalCount={0} />,
};

export const Empty: Story = {
  render: () => <Stateful tasks={[]} totalCount={0} />,
};

export const ErrorState: Story = {
  render: () => (
    <Stateful tasks={[]} totalCount={0} errorMessage="Daemon unreachable on /api/tasks." />
  ),
};

export const SingleGroup: Story = {
  render: () => (
    <Stateful
      tasks={FIXTURE_TASKS.filter(task => task.status === "in_progress")}
      totalCount={FIXTURE_TASKS.length}
    />
  ),
};
