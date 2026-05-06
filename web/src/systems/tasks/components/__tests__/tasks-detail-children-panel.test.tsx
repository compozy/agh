import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksDetailChildrenPanel } from "../tasks-detail-children-panel";
import type { TaskChildSummary } from "../../types";

function buildChild(overrides: Partial<TaskChildSummary> = {}): TaskChildSummary {
  return {
    id: "child_001",
    identifier: "TASK-43",
    status: "ready",
    scope: "workspace",
    title: "Write migration",
    priority: "medium",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    ...overrides,
  } as TaskChildSummary;
}

describe("TasksDetailChildrenPanel", () => {
  it("renders empty state when no children exist", () => {
    render(<TasksDetailChildrenPanel items={[]} />);
    expect(screen.getByTestId("tasks-detail-children-empty")).toBeInTheDocument();
  });

  it("renders error state when fetch fails", () => {
    render(<TasksDetailChildrenPanel errorMessage="boom" items={[]} />);
    expect(screen.getByTestId("tasks-detail-children-error")).toHaveTextContent("boom");
  });

  it("renders child task rows with deep-link", () => {
    render(<TasksDetailChildrenPanel items={[buildChild(), buildChild({ id: "child_002" })]} />);
    expect(screen.getByTestId("tasks-detail-children-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-detail-children-item-child_001")).toHaveTextContent(
      "Write migration"
    );
    expect(screen.getByTestId("tasks-detail-children-link-child_002")).toBeInTheDocument();
  });
});
