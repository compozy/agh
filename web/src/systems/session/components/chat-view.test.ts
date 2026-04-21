import { describe, expect, it } from "vitest";

import type { UIMessage } from "../types";
import { buildRows, mergeToolPairs } from "./chat-view";

function msg(overrides: Partial<UIMessage> & { id: string; role: UIMessage["role"] }): UIMessage {
  return {
    content: "",
    timestamp: Date.now(),
    ...overrides,
  };
}

describe("buildRows", () => {
  it("returns empty array for empty messages", () => {
    const rows = buildRows([], false);
    expect(rows).toEqual([]);
  });

  it("preserves non-tool messages as individual message rows", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "user", content: "Hello" }),
      msg({ id: "2", role: "assistant", content: "Hi there" }),
    ];
    const rows = buildRows(messages, false);
    expect(rows).toHaveLength(2);
    expect(rows[0]).toEqual({ kind: "message", msg: messages[0] });
    expect(rows[1]).toEqual({ kind: "message", msg: messages[1] });
  });

  it("groups consecutive tool_call + tool_result messages into tool_group", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "user", content: "Read a file" }),
      msg({ id: "2", role: "tool_call", toolName: "Read" }),
      msg({ id: "3", role: "tool_result", content: "file content" }),
      msg({ id: "4", role: "assistant", content: "Done" }),
    ];
    const rows = buildRows(messages, false);
    expect(rows).toHaveLength(3);
    expect(rows[0]).toEqual({ kind: "message", msg: messages[0] });
    expect(rows[1]).toEqual({ kind: "tool_group", tools: [messages[1], messages[2]] });
    expect(rows[2]).toEqual({ kind: "message", msg: messages[3] });
  });

  it("groups multiple consecutive tool calls into a single tool_group", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "tool_call", toolName: "Read" }),
      msg({ id: "2", role: "tool_result", content: "a" }),
      msg({ id: "3", role: "tool_call", toolName: "Write" }),
      msg({ id: "4", role: "tool_result", content: "b" }),
    ];
    const rows = buildRows(messages, false);
    expect(rows).toHaveLength(1);
    expect(rows[0].kind).toBe("tool_group");
    if (rows[0].kind === "tool_group") {
      expect(rows[0].tools).toHaveLength(4);
    }
  });

  it("adds processing row when isStreaming is true and no active stream", () => {
    const messages: UIMessage[] = [msg({ id: "1", role: "user", content: "Hello" })];
    const rows = buildRows(messages, true);
    expect(rows).toHaveLength(2);
    expect(rows[1]).toEqual({ kind: "processing" });
  });

  it("does not add processing row once tool activity is visible for the current turn", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "user", content: "Hello" }),
      msg({ id: "tool-1", role: "tool_call", toolName: "Bash", isStreaming: true }),
    ];
    const rows = buildRows(messages, true);
    expect(rows).toHaveLength(2);
    expect(rows.every(row => row.kind !== "processing")).toBe(true);
  });

  it("still adds processing row when only historical tool activity exists", () => {
    const messages: UIMessage[] = [
      msg({ id: "old-user", role: "user", content: "First prompt" }),
      msg({ id: "old-tool", role: "tool_call", toolName: "Read" }),
      msg({ id: "old-result", role: "tool_result", content: "done" }),
      msg({ id: "new-user", role: "user", content: "Second prompt" }),
    ];
    const rows = buildRows(messages, true);
    expect(rows.at(-1)).toEqual({ kind: "processing" });
  });

  it("does not add processing row when a message is actively streaming content", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "user", content: "Hello" }),
      msg({ id: "2", role: "assistant", content: "I am thinking", isStreaming: true }),
    ];
    const rows = buildRows(messages, true);
    expect(rows).toHaveLength(2);
    expect(rows.every(r => r.kind !== "processing")).toBe(true);
  });

  it("does not add processing row when isStreaming is false", () => {
    const messages: UIMessage[] = [msg({ id: "1", role: "user", content: "Hello" })];
    const rows = buildRows(messages, false);
    expect(rows).toHaveLength(1);
    expect(rows.every(r => r.kind !== "processing")).toBe(true);
  });

  it("handles interleaved message and tool sequences", () => {
    const messages: UIMessage[] = [
      msg({ id: "1", role: "user", content: "Do stuff" }),
      msg({ id: "2", role: "tool_call", toolName: "Bash" }),
      msg({ id: "3", role: "tool_result", content: "ok" }),
      msg({ id: "4", role: "assistant", content: "Done with step 1" }),
      msg({ id: "5", role: "tool_call", toolName: "Read" }),
      msg({ id: "6", role: "tool_result", content: "file" }),
      msg({ id: "7", role: "assistant", content: "All done" }),
    ];
    const rows = buildRows(messages, false);
    expect(rows).toHaveLength(5);
    expect(rows[0].kind).toBe("message");
    expect(rows[1].kind).toBe("tool_group");
    expect(rows[2].kind).toBe("message");
    expect(rows[3].kind).toBe("tool_group");
    expect(rows[4].kind).toBe("message");
  });
});

describe("mergeToolPairs", () => {
  it("merges tool_call with matching tool_result by id", () => {
    const tools: UIMessage[] = [
      msg({ id: "tc-1", role: "tool_call", toolName: "Read", toolInput: { file_path: "/a.ts" } }),
      msg({ id: "tc-1", role: "tool_result", toolResult: { content: "file data" } }),
    ];
    const merged = mergeToolPairs(tools);
    expect(merged).toHaveLength(1);
    expect(merged[0].role).toBe("tool_call");
    expect(merged[0].toolName).toBe("Read");
    expect(merged[0].toolResult).toEqual({ content: "file data" });
  });

  it("preserves tool_call without matching result (executing)", () => {
    const tools: UIMessage[] = [
      msg({ id: "tc-2", role: "tool_call", toolName: "Bash", toolInput: { command: "ls" } }),
    ];
    const merged = mergeToolPairs(tools);
    expect(merged).toHaveLength(1);
    expect(merged[0].toolResult).toBeUndefined();
  });

  it("merges multiple tool_call/result pairs", () => {
    const tools: UIMessage[] = [
      msg({ id: "tc-1", role: "tool_call", toolName: "Read" }),
      msg({ id: "tc-1", role: "tool_result", toolResult: { content: "a" } }),
      msg({ id: "tc-2", role: "tool_call", toolName: "Write" }),
      msg({ id: "tc-2", role: "tool_result", toolResult: { content: "b" } }),
    ];
    const merged = mergeToolPairs(tools);
    expect(merged).toHaveLength(2);
    expect(merged[0].toolName).toBe("Read");
    expect(merged[0].toolResult).toEqual({ content: "a" });
    expect(merged[1].toolName).toBe("Write");
    expect(merged[1].toolResult).toEqual({ content: "b" });
  });

  it("merges toolError from result", () => {
    const tools: UIMessage[] = [
      msg({ id: "tc-1", role: "tool_call", toolName: "Bash" }),
      msg({ id: "tc-1", role: "tool_result", toolResult: { error: "fail" }, toolError: true }),
    ];
    const merged = mergeToolPairs(tools);
    expect(merged).toHaveLength(1);
    expect(merged[0].toolError).toBe(true);
  });

  it("skips orphan tool_result messages", () => {
    const tools: UIMessage[] = [
      msg({ id: "tc-1", role: "tool_result", toolResult: { content: "orphan" } }),
    ];
    const merged = mergeToolPairs(tools);
    expect(merged).toHaveLength(0);
  });
});
