import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TasksStreamResumeCard } from "../tasks-stream-resume-card";

describe("TasksStreamResumeCard", () => {
  it("renders zero-state when no latest event seq is known", () => {
    render(
      <TasksStreamResumeCard
        hasLatestEventSeq={false}
        latestEventSeq={null}
        streamErrorMessage={null}
        streamSeedSequence={0}
        streamState="idle"
      />
    );
    expect(screen.getByTestId("tasks-stream-resume-latest")).toHaveTextContent("—");
    expect(screen.getByTestId("tasks-stream-resume-seed")).toHaveTextContent("0");
    expect(screen.getByTestId("tasks-stream-resume-status")).toHaveTextContent(
      "awaiting first frame"
    );
  });

  it("renders latest event seq + connected state when stream is live", () => {
    render(
      <TasksStreamResumeCard
        hasLatestEventSeq
        latestEventSeq={42}
        streamErrorMessage={null}
        streamSeedSequence={42}
        streamState="connected"
      />
    );
    expect(screen.getByTestId("tasks-stream-resume-latest")).toHaveTextContent("42");
    expect(screen.getByTestId("tasks-stream-resume-seed")).toHaveTextContent("42");
    expect(screen.getByTestId("tasks-stream-resume-status")).toHaveTextContent("connected");
  });

  it("surfaces the disconnected state with the error message", () => {
    render(
      <TasksStreamResumeCard
        hasLatestEventSeq
        latestEventSeq={42}
        streamErrorMessage="dropped"
        streamSeedSequence={42}
        streamState="error"
      />
    );
    expect(screen.getByTestId("tasks-stream-resume-error")).toHaveTextContent("dropped");
  });

  it("renders disabled hint when stream subscription is disabled", () => {
    render(
      <TasksStreamResumeCard
        hasLatestEventSeq={false}
        latestEventSeq={null}
        streamErrorMessage={null}
        streamSeedSequence={0}
        streamState="disabled"
      />
    );
    expect(screen.getByTestId("tasks-stream-resume-disabled")).toBeInTheDocument();
  });
});
