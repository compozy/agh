import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { AgentEventPayload, RuntimeActivityPayload } from "../types";
import {
  isRuntimeActivityEvent,
  RuntimeActivityNotice,
  SessionActivityInline,
} from "./runtime-activity-notice";

const runtime: RuntimeActivityPayload = {
  turn_id: "turn_001",
  turn_source: "user",
  current_tool: "Bash",
  idle_seconds: 42,
  elapsed_ms: 660_000,
  elapsed_seconds: 660,
  last_activity_kind: "tool_call",
  last_activity_detail: "running command",
};

describe("RuntimeActivityNotice", () => {
  it("recognizes runtime progress and warning events only when activity exists", () => {
    expect(isRuntimeActivityEvent({ type: "runtime_progress", runtime })).toBe(true);
    expect(isRuntimeActivityEvent({ type: "runtime_warning", runtime })).toBe(true);
    expect(isRuntimeActivityEvent({ type: "runtime_progress" })).toBe(false);
    expect(isRuntimeActivityEvent({ type: "agent_message", runtime })).toBe(false);
  });

  it("renders progress as a separate status notice", () => {
    const event: AgentEventPayload = {
      type: "runtime_progress",
      text: "Still working",
      runtime,
    };

    render(<RuntimeActivityNotice event={event} />);

    expect(screen.getByTestId("runtime-activity-notice")).toHaveAttribute("data-tone", "progress");
    expect(screen.getByText("Still working")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-activity-detail")).toHaveTextContent("Using Bash");
    expect(screen.getByTestId("runtime-activity-meta")).toHaveTextContent("11m elapsed, 42s idle");
  });

  it("renders warnings with alert semantics", () => {
    const event: AgentEventPayload = {
      type: "runtime_warning",
      runtime: {
        ...runtime,
        current_tool: undefined,
        last_activity_detail: "no provider activity observed",
      },
    };

    render(<RuntimeActivityNotice event={event} />);

    expect(screen.getByRole("alert")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByText("Runtime warning")).toBeInTheDocument();
    expect(screen.getByTestId("runtime-activity-detail")).toHaveTextContent(
      "no provider activity observed"
    );
  });

  it("does not render non-runtime events", () => {
    render(<RuntimeActivityNotice event={{ type: "agent_message", text: "hello" }} />);

    expect(screen.queryByTestId("runtime-activity-notice")).not.toBeInTheDocument();
  });
});

describe("SessionActivityInline", () => {
  it("renders compact session activity for headers", () => {
    render(<SessionActivityInline activity={runtime} />);

    expect(screen.getByTestId("session-activity-inline")).toHaveTextContent("Using Bash");
    expect(screen.getByTestId("session-activity-inline")).toHaveTextContent("42s");
  });

  it("does not render without activity", () => {
    render(<SessionActivityInline activity={null} />);

    expect(screen.queryByTestId("session-activity-inline")).not.toBeInTheDocument();
  });
});
