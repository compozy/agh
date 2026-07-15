import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => {
    const { params: _params, to: _to, ...domRest } = rest as Record<string, unknown>;
    return <a {...domRest}>{children}</a>;
  },
  useRouter: () => ({ history: { back: () => undefined } }),
}));

import { TasksDetailHeader } from "../tasks-detail-header";
import { buildTaskRunRecordFixture } from "../../mocks/fixtures";
import type { TaskDetailView } from "../../types";

function buildSummaryActiveRun(
  overrides: Partial<NonNullable<TaskDetailView["summary"]["active_run"]>> = {}
): NonNullable<TaskDetailView["summary"]["active_run"]> {
  return {
    ...buildTaskRunRecordFixture(),
    max_attempts: 3,
    ...overrides,
  };
}

function buildDetail(
  overrides: Partial<TaskDetailView["task"]> = {},
  summaryOverrides: Partial<TaskDetailView["summary"]> = {}
): TaskDetailView {
  const task = {
    id: "task_001",
    identifier: "TASK-42",
    title: "Summarize review feedback",
    status: "ready",
    scope: "workspace",
    origin: { kind: "cli", ref: "op" },
    created_at: "2026-04-11T09:00:00Z",
    updated_at: "2026-04-11T09:00:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    max_attempts: 3,
    ...overrides,
  } as TaskDetailView["task"];

  return {
    task,
    summary: {
      ...(task as unknown as TaskDetailView["summary"]),
      ...summaryOverrides,
    } as TaskDetailView["summary"],
  } as TaskDetailView;
}

describe("TasksDetailHeader", () => {
  it("Should render the body header with title, MonoBadge id, status pill, and action slot in DOM order", () => {
    const { container } = render(<TasksDetailHeader detail={buildDetail()} />);

    expect(screen.getByTestId("tasks-detail-title")).toHaveTextContent("Summarize review feedback");
    expect(screen.getByTestId("tasks-detail-id")).toHaveTextContent("task-42");
    expect(screen.getByTestId("tasks-detail-status")).toHaveTextContent("Ready");
    expect(screen.getByTestId("tasks-detail-actions")).toBeInTheDocument();

    // Breadcrumb surfaces the short identifier
    expect(screen.getByTestId("tasks-detail-breadcrumb")).toHaveTextContent("TASK-42");

    // Priority is a pill, while the meta row keeps provenance and timestamps.
    expect(screen.getByTestId("tasks-detail-priority")).toHaveTextContent("High");
    expect(screen.getByTestId("tasks-detail-meta")).toHaveTextContent("pedro@");

    // Status dot rendered alongside title
    const dot = container.querySelector('[data-slot="pill-dot"]');
    expect(dot).not.toBeNull();
  });

  it("Should fire cancel, publish, and enqueue callbacks", () => {
    const onCancel = vi.fn();
    const onEnqueueRun = vi.fn();

    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "in_progress" })}
        onCancel={onCancel}
        onEnqueueRun={onEnqueueRun}
      />
    );

    fireEvent.click(screen.getByTestId("tasks-detail-cancel"));
    fireEvent.click(screen.getByTestId("tasks-detail-enqueue"));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onEnqueueRun).toHaveBeenCalledTimes(1);
  });

  it("Should open a confirmation dialog before deleting the task", () => {
    const onDelete = vi.fn();

    render(<TasksDetailHeader detail={buildDetail()} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId("tasks-detail-delete"));
    expect(screen.getByTestId("tasks-detail-delete-dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-detail-delete-confirm"));
    expect(onDelete).toHaveBeenCalledWith("task_001");
  });

  it("Should surface the publish button for draft tasks", () => {
    const onPublish = vi.fn();
    render(<TasksDetailHeader detail={buildDetail({ status: "draft" })} onPublish={onPublish} />);

    fireEvent.click(screen.getByTestId("tasks-detail-publish"));
    expect(onPublish).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
  });

  it("Should disable destructive actions while mutations are pending", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "in_progress" })}
        pending={{ cancel: true, delete: true, enqueue: true }}
        onDelete={() => {}}
        onCancel={() => {}}
        onEnqueueRun={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-delete")).toBeDisabled();
    expect(screen.getByTestId("tasks-detail-cancel")).toBeDisabled();
    expect(screen.getByTestId("tasks-detail-enqueue")).toBeDisabled();
  });

  it("Should require a reason before pausing a task", async () => {
    const onPause = vi.fn().mockResolvedValue(undefined);

    render(<TasksDetailHeader detail={buildDetail({ status: "ready" })} onPause={onPause} />);

    fireEvent.click(screen.getByTestId("tasks-detail-pause"));
    expect(screen.getByTestId("tasks-detail-pause-dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-detail-pause-confirm"));
    expect(screen.getByTestId("tasks-detail-pause-error")).toHaveTextContent(
      "Provide a pause reason."
    );

    fireEvent.change(screen.getByTestId("tasks-detail-pause-reason"), {
      target: { value: "provider incident" },
    });
    fireEvent.click(screen.getByTestId("tasks-detail-pause-confirm"));

    await waitFor(() => {
      expect(onPause).toHaveBeenCalledWith("provider incident");
    });
  });

  it("Should surface resume for directly paused tasks and block start-run copy", () => {
    const onResume = vi.fn();
    render(
      <TasksDetailHeader
        detail={buildDetail(
          {
            status: "ready",
            paused: true,
            paused_reason: "provider incident",
          },
          { effective_paused: true }
        )}
        onEnqueueRun={() => {}}
        onResume={onResume}
      />
    );

    expect(screen.getByTestId("tasks-detail-pause-state")).toHaveTextContent("Paused");
    expect(screen.getByTestId("tasks-detail-enqueue")).toBeDisabled();
    fireEvent.click(screen.getByTestId("tasks-detail-resume"));
    expect(onResume).toHaveBeenCalledTimes(1);
  });

  it("Should render the saved-intent lifecycle pill and hint for draft tasks without implying autonomy", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "draft", draft: true })}
        onPublish={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Saved intent");
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveAttribute(
      "title",
      expect.stringMatching(/saved intent/i)
    );
    expect(screen.getByTestId("tasks-detail-publish")).toHaveTextContent("Publish");
    expect(screen.getByTestId("tasks-detail-publish")).toHaveAttribute(
      "title",
      expect.stringMatching(/coordinator handoff/i)
    );
    expect(screen.queryByTestId("tasks-detail-coordination")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
  });

  it("Should label the start-run button as the coordinator handoff boundary", () => {
    render(<TasksDetailHeader detail={buildDetail({ status: "ready" })} onEnqueueRun={() => {}} />);

    const button = screen.getByTestId("tasks-detail-enqueue");
    expect(button).toHaveTextContent("Start run");
    expect(button).toHaveAttribute("title", expect.stringMatching(/coordinator handoff/i));
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Ready to start");
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveAttribute(
      "title",
      expect.stringMatching(/start enqueues a coordinator-handoff run/i)
    );
  });

  it.each(["queued", "claimed", "starting", "running"] as const)(
    "Should hide the start-run action while an active run is %s",
    status => {
      const activeRun = buildSummaryActiveRun({
        id: "run_42",
        task_id: "task_001",
        attempt: 1,
        status,
        queued_at: "2026-04-11T09:30:00Z",
      });
      const detail = buildDetail({ status: "ready" }, { active_run: activeRun });

      render(<TasksDetailHeader detail={detail} onEnqueueRun={() => {}} />);

      expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
    }
  );

  it("Should recover the active needs_attention run instead of offering Start run", () => {
    const onRecover = vi.fn();
    const activeRun = buildSummaryActiveRun({
      id: "run_attention",
      task_id: "task_001",
      attempt: 1,
      status: "needs_attention",
      queued_at: "2026-04-11T09:30:00Z",
    });

    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "ready", max_attempts: 2 }, { active_run: activeRun })}
        onEnqueueRun={() => {}}
        onRecover={onRecover}
      />
    );

    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Needs attention");
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Recover" }));
    expect(onRecover).toHaveBeenCalledTimes(1);
  });

  it("Should hide Recover and Start run when the active needs_attention run exhausted attempts", () => {
    const activeRun = buildSummaryActiveRun({
      id: "run_exhausted",
      task_id: "task_001",
      attempt: 1,
      status: "needs_attention",
      queued_at: "2026-04-11T09:30:00Z",
    });

    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "ready", max_attempts: 1 }, { active_run: activeRun })}
        onEnqueueRun={() => {}}
        onRecover={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Needs attention");
    expect(screen.queryByTestId("tasks-detail-recover")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();
  });

  it("Should render the needs_attention badge with its reason and fire recover", () => {
    const onRecover = vi.fn();
    render(
      <TasksDetailHeader
        detail={buildDetail({
          status: "needs_attention",
          needs_attention: true,
          needs_attention_reason: "Block recurrence limit reached for transient",
        })}
        onEnqueueRun={() => {}}
        onRecover={onRecover}
      />
    );

    const badge = screen.getByTestId("tasks-detail-needs-attention");
    // The badge surfaces the escalation reason (the status pill already carries the
    // "Needs Attention" label), so it must not duplicate that text.
    expect(badge).toHaveTextContent("Block recurrence limit reached for transient");
    expect(badge).toHaveAttribute("title", "Block recurrence limit reached for transient");
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Needs attention");
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveAttribute(
      "title",
      expect.stringMatching(/recover it before/i)
    );
    expect(screen.queryByTestId("tasks-detail-enqueue")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("tasks-detail-recover"));
    expect(onRecover).toHaveBeenCalledTimes(1);
  });

  it("Should hide the recover action and badge when the task is not escalated", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "ready", needs_attention: false })}
        onRecover={() => {}}
      />
    );

    expect(screen.queryByTestId("tasks-detail-needs-attention")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-recover")).not.toBeInTheDocument();
  });

  it("Should hide the badge and recover action on a terminal task even if needs_attention lingered", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "canceled", needs_attention: true })}
        onRecover={() => {}}
      />
    );

    expect(screen.queryByTestId("tasks-detail-needs-attention")).not.toBeInTheDocument();
    expect(screen.queryByTestId("tasks-detail-recover")).not.toBeInTheDocument();
  });

  it("Should disable the recover action while the mutation is pending", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ status: "needs_attention", needs_attention: true })}
        pending={{ recover: true }}
        onRecover={() => {}}
      />
    );

    expect(screen.getByTestId("tasks-detail-recover")).toBeDisabled();
  });

  it("Should show the wake indicator on for agent-created tasks and off on opt-out", () => {
    const { rerender } = render(
      <TasksDetailHeader
        detail={buildDetail({
          created_by: { kind: "agent_session", ref: "session_a" },
          wake_creator: true,
        })}
      />
    );
    expect(screen.getByTestId("tasks-detail-wake")).toHaveTextContent("Wake on");

    rerender(
      <TasksDetailHeader
        detail={buildDetail({
          created_by: { kind: "agent_session", ref: "session_a" },
          wake_creator: false,
        })}
      />
    );
    const wake = screen.getByTestId("tasks-detail-wake");
    expect(wake).toHaveTextContent("Wake off");
    expect(wake).toHaveAttribute("title", expect.stringMatching(/opted out/i));
  });

  it("Should suppress the wake indicator for tasks not created by an agent session", () => {
    render(
      <TasksDetailHeader
        detail={buildDetail({ created_by: { kind: "human", ref: "operator" }, wake_creator: true })}
      />
    );
    expect(screen.queryByTestId("tasks-detail-wake")).not.toBeInTheDocument();
  });

  it("Should surface the coordination channel chip when the active run is bound to a channel", () => {
    const activeRun = buildSummaryActiveRun({
      id: "run_42",
      task_id: "task_001",
      attempt: 1,
      status: "queued",
      queued_at: "2026-04-11T09:30:00Z",
    });
    const detail = buildDetail({ status: "in_progress" }, { active_run: activeRun });

    render(<TasksDetailHeader detail={detail} onEnqueueRun={() => {}} />);

    expect(screen.getByTestId("tasks-detail-coordination")).toHaveTextContent(
      "Channel: TASK-1 coordination"
    );
    expect(screen.getByTestId("tasks-detail-coordination")).toHaveAttribute(
      "title",
      expect.stringMatching(/channel messages support coordination only/i)
    );
    expect(screen.getByTestId("tasks-detail-lifecycle")).toHaveTextContent("Coordinator handoff");
  });
});
