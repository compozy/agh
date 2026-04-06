import { describe, expect, it } from "vitest";

import type { TranscriptMessage } from "../types";
import { mapTranscriptToMessages } from "./transcript-mapper";

describe("mapTranscriptToMessages", () => {
  it("maps assistant transcript messages with persisted timestamps", () => {
    const messages: TranscriptMessage[] = [
      {
        id: "msg-1",
        role: "assistant",
        content: "Hello",
        thinking: "Inspecting",
        thinking_complete: true,
        tool_error: false,
        timestamp: "2026-04-03T12:00:00Z",
      },
    ];

    const result = mapTranscriptToMessages(messages);
    expect(result).toEqual([
      {
        id: "msg-1",
        role: "assistant",
        content: "Hello",
        thinking: "Inspecting",
        thinkingComplete: true,
        isStreaming: false,
        timestamp: Date.parse("2026-04-03T12:00:00Z"),
      },
    ]);
  });

  it("maps tool call and tool result transcript messages", () => {
    const messages: TranscriptMessage[] = [
      {
        id: "tool-1",
        role: "tool_call",
        content: "",
        tool_name: "Read",
        tool_input: { file_path: "/tmp/demo.ts" },
        thinking_complete: false,
        tool_error: false,
        timestamp: "2026-04-03T12:00:01Z",
      },
      {
        id: "tool-1",
        role: "tool_result",
        content: "",
        tool_name: "Read",
        tool_result: {
          content: "line1\nline2",
          raw_output: "line1\nline2",
        },
        thinking_complete: false,
        tool_error: false,
        timestamp: "2026-04-03T12:00:02Z",
      },
    ];

    const result = mapTranscriptToMessages(messages);
    expect(result[0]).toMatchObject({
      id: "tool-1",
      role: "tool_call",
      toolName: "Read",
      toolInput: { file_path: "/tmp/demo.ts" },
      isStreaming: false,
    });
    expect(result[1]).toMatchObject({
      id: "tool-1",
      role: "tool_result",
      toolName: "Read",
      toolResult: {
        content: "line1\nline2",
        rawOutput: "line1\nline2",
      },
      isStreaming: false,
    });
  });
});
