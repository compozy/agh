import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksInboxItem } from "./tasks-inbox-item";
import { buildInboxItemFixture } from "./test-fixtures";

describe("TasksInboxItem", () => {
  it("renders an accent StatusDot when the item is unread", () => {
    const item = buildInboxItemFixture({
      lane: "approvals",
      task: {
        id: "task_apr",
        identifier: "TASK-33",
        scope: "workspace",
        status: "pending",
        title: "Rotate keys",
      },
      triage: {
        actor: { kind: "human", ref: "op" },
        archived: false,
        dismissed: false,
        read: false,
        task_id: "task_apr",
        updated_at: "2026-04-17T10:00:00Z",
      },
    });

    render(<TasksInboxItem item={item} />);

    const dot = screen.getByTestId("tasks-inbox-item-unread-task_apr");
    expect(dot).toBeInTheDocument();
    expect(dot).toHaveAttribute("data-slot", "status-dot");
    expect(dot).toHaveAttribute("data-tone", "accent");
  });

  it("omits the unread dot when the item has been read", () => {
    const item = buildInboxItemFixture({
      triage: {
        actor: { kind: "human", ref: "op" },
        archived: false,
        dismissed: false,
        read: true,
        task_id: "task_001",
        updated_at: "2026-04-17T10:00:00Z",
      },
    });

    render(<TasksInboxItem item={item} />);

    expect(screen.queryByTestId("tasks-inbox-item-unread-task_001")).not.toBeInTheDocument();
  });

  it("renders three action buttons (Reject / Approve / Open) for approval items and fires onApprove once", () => {
    const onApprove = vi.fn();
    const onReject = vi.fn();
    const item = buildInboxItemFixture({
      lane: "approvals",
      approval_policy: "manual",
      approval_state: "pending",
      task: {
        id: "task_apr",
        identifier: "TASK-33",
        scope: "workspace",
        status: "pending",
        title: "Rotate keys",
      },
    });

    render(<TasksInboxItem item={item} onApprove={onApprove} onReject={onReject} />);

    const actions = screen.getByTestId("tasks-inbox-item-actions-task_apr");
    const buttons = actions.querySelectorAll("[data-slot=button]");
    expect(buttons).toHaveLength(3);

    expect(screen.getByTestId("tasks-inbox-item-reject-task_apr")).toHaveAttribute(
      "data-variant",
      "danger"
    );
    expect(screen.getByTestId("tasks-inbox-item-approve-task_apr")).toHaveAttribute(
      "data-variant",
      "primary"
    );
    expect(screen.getByTestId("tasks-inbox-item-open-task_apr")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-inbox-item-approve-task_apr"));
    expect(onApprove).toHaveBeenCalledTimes(1);
    expect(onApprove).toHaveBeenCalledWith("task_apr");
  });

  it("does not invoke row selection when the Reject button is clicked", () => {
    const onOpen = vi.fn();
    const onReject = vi.fn();
    const item = buildInboxItemFixture({
      lane: "approvals",
      approval_policy: "manual",
      approval_state: "pending",
      task: {
        id: "task_apr",
        identifier: "TASK-33",
        scope: "workspace",
        status: "pending",
        title: "Rotate keys",
      },
    });

    render(<TasksInboxItem item={item} onApprove={vi.fn()} onOpen={onOpen} onReject={onReject} />);

    fireEvent.click(screen.getByTestId("tasks-inbox-item-reject-task_apr"));

    expect(onReject).toHaveBeenCalledWith("task_apr");
    expect(onOpen).not.toHaveBeenCalled();
  });
});
