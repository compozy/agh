import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
}));

import { TasksInboxView } from "../tasks-inbox-view";
import { buildInboxFixture, buildInboxItemFixture } from "../test-fixtures";

function makeBaseProps() {
  return {
    laneFilter: "all" as const,
    onLaneChange: vi.fn(),
    unreadOnly: false,
    onToggleUnread: vi.fn(),
    searchQuery: "",
    onSearchChange: vi.fn(),
  };
}

describe("TasksInboxView", () => {
  it("Should render the five-lane switcher (My work / Mentions / Failed runs / Updates / Approvals)", () => {
    render(<TasksInboxView {...makeBaseProps()} inbox={buildInboxFixture()} />);

    expect(screen.getByTestId("tasks-inbox-lane-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-inbox-lane-all")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-inbox-lane-my_work")).toHaveTextContent(/My work/);
    expect(screen.getByTestId("tasks-inbox-lane-mentions")).toHaveTextContent(/Mentions/);
    expect(screen.getByTestId("tasks-inbox-lane-failed_runs")).toHaveTextContent(/Failed runs/);
    expect(screen.getByTestId("tasks-inbox-lane-updates")).toHaveTextContent(/Updates/);
    expect(screen.getByTestId("tasks-inbox-lane-approvals")).toHaveTextContent(/Approvals/);
  });

  it("Should render approval items under the Needs review group with a warning solid dot", () => {
    const inbox = buildInboxFixture({
      total: 1,
      unread_total: 1,
      groups: [
        {
          lane: "approvals",
          count: 1,
          unread_count: 1,
          items: [
            buildInboxItemFixture({
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
              triage: {
                actor: { kind: "human", ref: "op" },
                archived: false,
                dismissed: false,
                read: false,
                task_id: "task_apr",
                updated_at: "2026-04-17T10:00:00Z",
              },
            }),
          ],
        },
      ],
    });

    render(<TasksInboxView {...makeBaseProps()} inbox={inbox} />);

    const group = screen.getByTestId("tasks-inbox-group-needs_review");
    expect(group).toBeInTheDocument();
    const dot = screen.getByTestId("tasks-inbox-group-dot-needs_review");
    expect(dot).toHaveAttribute("data-tone", "warning");
    expect(dot).toHaveAttribute("data-variant", "solid");
  });

  it("Should render blocked items under the Blocked group with a danger solid dot", () => {
    const inbox = buildInboxFixture({
      total: 1,
      unread_total: 0,
      groups: [
        {
          lane: "blocked",
          count: 1,
          unread_count: 0,
          items: [
            buildInboxItemFixture({
              lane: "blocked",
              blocking_reason: "awaiting deps",
              task: {
                id: "task_block",
                identifier: "TASK-99",
                scope: "workspace",
                status: "blocked",
                title: "Blocked task",
              },
              triage: {
                actor: { kind: "human", ref: "op" },
                archived: false,
                dismissed: false,
                read: true,
                task_id: "task_block",
                updated_at: "2026-04-17T10:00:00Z",
              },
            }),
          ],
        },
      ],
    });

    render(<TasksInboxView {...makeBaseProps()} inbox={inbox} />);

    const group = screen.getByTestId("tasks-inbox-group-blocked");
    expect(group).toBeInTheDocument();
    const dot = screen.getByTestId("tasks-inbox-group-dot-blocked");
    expect(dot).toHaveAttribute("data-tone", "danger");
    expect(dot).toHaveAttribute("data-variant", "solid");
  });

  it("Should emit lane, search, and unread toggle changes", () => {
    const props = makeBaseProps();
    const inbox = buildInboxFixture({
      total: 1,
      groups: [
        {
          lane: "my_work",
          count: 1,
          unread_count: 0,
          items: [buildInboxItemFixture()],
        },
      ],
    });
    render(<TasksInboxView {...props} inbox={inbox} />);

    fireEvent.click(screen.getByTestId("tasks-inbox-lane-approvals"));
    expect(props.onLaneChange).toHaveBeenCalledWith("approvals");

    fireEvent.change(screen.getByTestId("tasks-inbox-search"), { target: { value: "rotate" } });
    expect(props.onSearchChange).toHaveBeenCalledWith("rotate");

    const toggle = screen
      .getByTestId("tasks-inbox-unread-toggle")
      .querySelector("[role=switch]") as HTMLElement;
    fireEvent.click(toggle);
    expect(props.onToggleUnread).toHaveBeenCalledTimes(1);
    expect(props.onToggleUnread.mock.calls[0]?.[0]).toBe(true);
  });

  it("Should render loading, error, and empty states", () => {
    const { rerender } = render(<TasksInboxView {...makeBaseProps()} inbox={null} isLoading />);
    expect(screen.getByTestId("tasks-inbox-loading")).toBeInTheDocument();

    rerender(<TasksInboxView {...makeBaseProps()} errorMessage="oops" inbox={null} />);
    expect(screen.getByTestId("tasks-inbox-error")).toHaveTextContent("oops");

    rerender(<TasksInboxView {...makeBaseProps()} inbox={buildInboxFixture()} />);
    expect(screen.getByTestId("tasks-inbox-empty")).toBeInTheDocument();
  });

  it("Should invoke approval, retry, archive, dismiss, and mark-read actions", () => {
    const handlers = {
      onApprove: vi.fn(),
      onReject: vi.fn(),
      onRetry: vi.fn(),
      onArchive: vi.fn(),
      onDismiss: vi.fn(),
      onMarkRead: vi.fn(),
    };

    const inbox = buildInboxFixture({
      total: 4,
      unread_total: 3,
      groups: [
        {
          lane: "approvals",
          count: 1,
          unread_count: 1,
          items: [
            buildInboxItemFixture({
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
              triage: {
                actor: { kind: "human", ref: "op" },
                archived: false,
                dismissed: false,
                read: false,
                task_id: "task_apr",
                updated_at: "2026-04-17T10:00:00Z",
              },
            }),
          ],
        },
        {
          lane: "failed_runs",
          count: 1,
          unread_count: 1,
          items: [
            buildInboxItemFixture({
              lane: "failed_runs",
              task: {
                id: "task_fail",
                identifier: "TASK-27",
                scope: "workspace",
                status: "failed",
                title: "Sync embeddings",
              },
              run: {
                attempt: 3,
                id: "run_fail",
                max_attempts: 3,
                queued_at: "2026-04-17T09:55:00Z",
                status: "failed",
                error: "session timeout",
                task_id: "task_fail",
              },
              triage: {
                actor: { kind: "human", ref: "op" },
                archived: false,
                dismissed: false,
                read: false,
                task_id: "task_fail",
                updated_at: "2026-04-17T10:00:00Z",
              },
            }),
          ],
        },
        {
          lane: "my_work",
          count: 1,
          unread_count: 1,
          items: [
            buildInboxItemFixture({
              task: {
                id: "task_my",
                identifier: "TASK-42",
                scope: "workspace",
                status: "ready",
                title: "Review work",
              },
            }),
          ],
        },
      ],
    });

    render(<TasksInboxView {...makeBaseProps()} {...handlers} inbox={inbox} />);

    fireEvent.click(screen.getByTestId("tasks-inbox-item-approve-task_apr"));
    fireEvent.click(screen.getByTestId("tasks-inbox-item-reject-task_apr"));
    fireEvent.click(screen.getByTestId("tasks-inbox-item-retry-task_fail"));
    fireEvent.click(screen.getByTestId("tasks-inbox-item-dismiss-task_fail"));
    fireEvent.click(screen.getByTestId("tasks-inbox-item-mark-read-task_my"));
    fireEvent.click(screen.getByTestId("tasks-inbox-item-archive-task_my"));

    expect(handlers.onApprove).toHaveBeenCalledWith("task_apr");
    expect(handlers.onReject).toHaveBeenCalledWith("task_apr");
    expect(handlers.onRetry).toHaveBeenCalledWith("task_fail");
    expect(handlers.onDismiss).toHaveBeenCalledWith("task_fail");
    expect(handlers.onMarkRead).toHaveBeenCalledWith("task_my");
    expect(handlers.onArchive).toHaveBeenCalledWith("task_my");
  });
});
