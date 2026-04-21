import { describe, expect, it } from "vitest";
import type { UIMessage as AIUIMessage } from "ai";

import { mapLiveChatMessages } from "./live-message-mapper";

describe("mapLiveChatMessages", () => {
  it("maps a streaming assistant tool call into visible tool rows immediately", () => {
    const messages: AIUIMessage[] = [
      {
        id: "user-1",
        role: "user",
        parts: [{ type: "text", text: "run ls" }],
      },
      {
        id: "assistant-1",
        role: "assistant",
        parts: [
          { type: "reasoning", text: "Checking the workspace", state: "streaming" },
          {
            type: "dynamic-tool",
            toolName: "Bash",
            toolCallId: "tool-1",
            state: "input-available",
            input: { command: "ls" },
          },
        ],
      },
    ];

    const rows = mapLiveChatMessages(messages, () => 1);

    expect(rows).toEqual([
      {
        id: "user-1",
        role: "user",
        content: "run ls",
        timestamp: 1,
      },
      {
        id: "assistant-1:assistant:0",
        role: "assistant",
        content: "",
        thinking: "Checking the workspace",
        thinkingComplete: undefined,
        isStreaming: true,
        timestamp: 1,
      },
      {
        id: "tool-1",
        role: "tool_call",
        content: "",
        toolName: "Bash",
        toolInput: { command: "ls" },
        isStreaming: true,
        timestamp: 1,
      },
    ]);
  });

  it("maps completed tool output into a tool_result row", () => {
    const messages: AIUIMessage[] = [
      {
        id: "assistant-1",
        role: "assistant",
        parts: [
          {
            type: "tool-bash",
            toolCallId: "tool-1",
            state: "output-available",
            input: { command: "pwd" },
            output: { stdout: "/workspace" },
          },
        ],
      },
    ];

    const rows = mapLiveChatMessages(messages, () => 1);

    expect(rows).toEqual([
      {
        id: "tool-1",
        role: "tool_call",
        content: "",
        toolName: "bash",
        toolInput: { command: "pwd" },
        isStreaming: false,
        timestamp: 1,
      },
      {
        id: "tool-1",
        role: "tool_result",
        content: "",
        toolName: "bash",
        toolResult: {
          stdout: "/workspace",
          stderr: undefined,
          filePath: undefined,
          content: undefined,
          structuredPatch: undefined,
          error: undefined,
          rawOutput: { stdout: "/workspace" },
        },
        toolError: false,
        timestamp: 1,
      },
    ]);
  });
});
