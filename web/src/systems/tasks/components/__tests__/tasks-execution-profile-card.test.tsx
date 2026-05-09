import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TasksExecutionProfileCard } from "../tasks-execution-profile-card";
import { buildTaskExecutionProfileFixture } from "../../mocks/fixtures";

function noopAsync<T>(): T {
  return undefined as unknown as T;
}

describe("TasksExecutionProfileCard", () => {
  it("renders empty state when profile is null and no error", () => {
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={async () => noopAsync<void>()}
        profile={null}
        taskId="task_001"
      />
    );
    expect(screen.getByTestId("tasks-execution-profile-empty")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-execution-profile-delete")).toBeDisabled();
    expect(screen.getByTestId("tasks-execution-profile-edit")).toBeEnabled();
  });

  it("renders error state when profile fetch fails", () => {
    render(
      <TasksExecutionProfileCard
        errorMessage="boom"
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={async () => noopAsync<void>()}
        profile={null}
        taskId="task_001"
      />
    );
    expect(screen.getByTestId("tasks-execution-profile-error")).toHaveTextContent("boom");
  });

  it("renders read view with worker, coordinator, review, sandbox sections", () => {
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={async () => noopAsync<void>()}
        profile={buildTaskExecutionProfileFixture()}
        taskId="task_001"
      />
    );
    const summary = screen.getByTestId("tasks-execution-profile-summary");
    expect(summary).toBeInTheDocument();
    expect(summary).toHaveTextContent("Worker mode");
    expect(summary).toHaveTextContent("select");
    expect(summary).toHaveTextContent("Coordinator mode");
    expect(summary).toHaveTextContent("guided");
    expect(summary).toHaveTextContent("Sandbox mode");
    expect(summary).toHaveTextContent("ref");
  });

  it("disables edit and delete when active run blocks profile mutation", () => {
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={async () => noopAsync<void>()}
        profile={buildTaskExecutionProfileFixture()}
        state={{ hasActiveRun: true }}
        taskId="task_001"
      />
    );
    expect(screen.getByTestId("tasks-execution-profile-active-run-warning")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-execution-profile-edit")).toBeDisabled();
    expect(screen.getByTestId("tasks-execution-profile-delete")).toBeDisabled();
  });

  it("submits a parsed profile through the editor dialog", async () => {
    const onSetProfile = vi.fn().mockResolvedValue(undefined);
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={onSetProfile}
        profile={buildTaskExecutionProfileFixture()}
        taskId="task_001"
      />
    );

    fireEvent.click(screen.getByTestId("tasks-execution-profile-edit"));
    expect(screen.getByTestId("tasks-execution-profile-editor-dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("tasks-execution-profile-editor-submit"));

    await waitFor(() => expect(onSetProfile).toHaveBeenCalledTimes(1));
    const [payload] = onSetProfile.mock.calls[0]!;
    expect(payload.task_id).toBe("task_001");
    expect(payload.worker).toMatchObject({ mode: "select" });
  });

  it("surfaces JSON parse errors without firing the mutation", async () => {
    const onSetProfile = vi.fn().mockResolvedValue(undefined);
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={async () => noopAsync<void>()}
        onSetProfile={onSetProfile}
        profile={null}
        taskId="task_001"
      />
    );

    fireEvent.click(screen.getByTestId("tasks-execution-profile-edit"));
    const input = screen.getByTestId("tasks-execution-profile-editor-input");
    fireEvent.change(input, { target: { value: "{ not json" } });
    fireEvent.click(screen.getByTestId("tasks-execution-profile-editor-submit"));

    await waitFor(() =>
      expect(screen.getByTestId("tasks-execution-profile-editor-error")).toBeInTheDocument()
    );
    expect(onSetProfile).not.toHaveBeenCalled();
  });

  it("delegates delete confirmation to onDeleteProfile", async () => {
    const onDeleteProfile = vi.fn().mockResolvedValue(undefined);
    render(
      <TasksExecutionProfileCard
        onDeleteProfile={onDeleteProfile}
        onSetProfile={async () => noopAsync<void>()}
        profile={buildTaskExecutionProfileFixture()}
        taskId="task_001"
      />
    );

    fireEvent.click(screen.getByTestId("tasks-execution-profile-delete"));
    fireEvent.click(screen.getByTestId("tasks-execution-profile-delete-confirm"));
    await waitFor(() => expect(onDeleteProfile).toHaveBeenCalledTimes(1));
  });
});
