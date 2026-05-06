import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksDetailOrchestrationPanel } from "../tasks-detail-orchestration-panel";

const noop = async () => undefined;

describe("TasksDetailOrchestrationPanel", () => {
  it("renders all orchestration cards with their empty states", () => {
    render(
      <TasksDetailOrchestrationPanel
        notifications={{
          isLoading: false,
          subscriptions: [],
          onCreate: noop,
          onDelete: noop,
        }}
        profile={{
          taskId: "task_001",
          profile: null,
          onSetProfile: noop,
          onDeleteProfile: noop,
        }}
        reviews={{
          reviews: [],
        }}
        stream={{
          hasLatestEventSeq: true,
          latestEventSeq: 14,
          streamErrorMessage: null,
          streamSeedSequence: 14,
          streamState: "idle",
        }}
      />
    );
    expect(screen.getByTestId("tasks-detail-orchestration-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-execution-profile-empty")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-reviews-card-empty")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-bridge-notifications-empty")).toBeInTheDocument();
    expect(screen.getByTestId("tasks-stream-resume-card")).toBeInTheDocument();
  });
});
