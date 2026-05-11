import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailDependenciesPanel } from "../tasks-detail-dependencies-panel";
import type { TaskDetailView } from "../../types";

type DependencyReference = NonNullable<TaskDetailView["dependency_references"]>[number];

function buildDependency(overrides: Partial<DependencyReference["depends_on"]> = {}) {
  return {
    created_at: "2026-04-11T09:00:00Z",
    task_id: "task_001",
    depends_on_task_id: "dep_001",
    kind: "blocks",
    depends_on: {
      id: "dep_001",
      identifier: "TASK-19",
      status: "completed",
      scope: "workspace",
      title: "Write tests",
      priority: "medium",
      owner: { kind: "agent_session", ref: "Coder" },
      ...overrides,
    },
  } as DependencyReference;
}

describe("TasksDetailDependenciesPanel", () => {
  it("renders empty state when no dependencies exist", () => {
    render(<TasksDetailDependenciesPanel dependencies={[]} />);
    expect(screen.getByTestId("tasks-detail-dependencies-empty")).toBeInTheDocument();
  });

  it("renders error state when a message is provided", () => {
    render(<TasksDetailDependenciesPanel dependencies={[]} errorMessage="boom" />);
    expect(screen.getByTestId("tasks-detail-dependencies-error")).toHaveTextContent("boom");
  });

  it("renders dependency rows with deep-link to the blocking task", () => {
    render(
      <TasksDetailDependenciesPanel
        dependencies={[
          buildDependency(),
          buildDependency({ id: "dep_002", identifier: "TASK-32", title: "Approve" }),
        ]}
      />
    );

    expect(screen.getByTestId("tasks-detail-dependencies-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-dependencies-item-dep_001")).toHaveTextContent(
      "Write tests"
    );
    expect(screen.getByTestId("tasks-detail-dependencies-item-dep_002")).toHaveTextContent(
      "task-32"
    );
    expect(screen.getByTestId("tasks-detail-dependencies-link-dep_001")).toBeInTheDocument();
  });
});
