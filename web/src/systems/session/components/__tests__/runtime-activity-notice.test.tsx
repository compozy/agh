import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { AgentEventPayload, RuntimeActivityPayload } from "../../types";
import {
  isSessionErrorEvent,
  isRuntimeActivityEvent,
  isTranscriptMarkerEvent,
  RuntimeActivityNotice,
  SessionActivityInline,
} from "../runtime-activity-notice";

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

  it("recognizes fatal session errors when error or failure details exist", () => {
    expect(
      isSessionErrorEvent({
        type: "error",
        error: '{"code":-32603,"message":"Internal error"}',
      })
    ).toBe(true);
    expect(isSessionErrorEvent({ type: "error", failure: { kind: "process_exit" } })).toBe(false);
    expect(
      isSessionErrorEvent({
        type: "error",
        failure: { kind: "process_exit", summary: "peer disconnected before response" },
      })
    ).toBe(true);
    expect(isSessionErrorEvent({ type: "runtime_warning", error: "failed" })).toBe(false);
  });

  it("recognizes transcript marker events", () => {
    expect(isTranscriptMarkerEvent({ type: "transcript_marker.created" })).toBe(true);
    expect(isTranscriptMarkerEvent({ type: "transcript_marker.redacted" })).toBe(true);
    expect(isTranscriptMarkerEvent({ type: "runtime_warning" })).toBe(false);
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

  it("renders session errors with alert semantics and failure detail", () => {
    render(
      <RuntimeActivityNotice
        event={{
          type: "error",
          error:
            '{"code":-32603,"message":"Internal error","data":{"error":"peer disconnected before response"}}',
          failure: {
            kind: "process_exit",
            summary: "peer disconnected before response",
          },
        }}
      />
    );

    expect(screen.getByRole("alert")).toHaveAttribute("data-tone", "danger");
    expect(screen.getByTestId("session-error-notice")).toHaveTextContent("Session failed");
    expect(screen.getByTestId("session-error-meta")).toHaveTextContent("process_exit");
    expect(screen.getByTestId("session-error-detail")).toHaveTextContent(
      "peer disconnected before response"
    );
  });

  it("renders transcript markers with marker semantics", () => {
    render(
      <RuntimeActivityNotice
        event={{
          type: "transcript_marker.created",
          text: "Runtime activity timed out.",
          title: "transcript_marker.prompt_timeout",
          raw: {
            kind: "transcript_marker.prompt_timeout",
            occurred_at: "2026-04-20T12:00:00Z",
            summary: "Runtime activity timed out.",
          },
        }}
      />
    );

    expect(screen.getByRole("alert")).toHaveAttribute("data-tone", "danger");
    expect(screen.getByTestId("transcript-marker-notice")).toHaveTextContent("Transcript marker");
    expect(screen.getByTestId("transcript-marker-kind")).toHaveTextContent(
      "transcript_marker.prompt_timeout"
    );
    expect(screen.getByTestId("transcript-marker-summary")).toHaveTextContent(
      "Runtime activity timed out."
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
