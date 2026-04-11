import type { ToolUseResult, TranscriptMessage, TranscriptToolResult, UIMessage } from "../types";

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseTimestamp(raw: string): number {
  const timestamp = Date.parse(raw);
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function mapToolResult(result?: TranscriptToolResult | null): ToolUseResult | undefined {
  if (!result) return undefined;

  return {
    stdout: result.stdout,
    stderr: result.stderr,
    filePath: result.file_path,
    content: result.content,
    structuredPatch: Array.isArray(result.structured_patch) ? result.structured_patch : undefined,
    error: result.error,
    rawOutput: result.raw_output,
  };
}

function mapTranscriptRole(role: TranscriptMessage["role"]): UIMessage["role"] {
  switch (role) {
    case "assistant":
    case "user":
    case "tool_call":
    case "tool_result":
      return role;
    default:
      return "system";
  }
}

function mapTranscriptMessage(message: TranscriptMessage): UIMessage {
  const timestamp = parseTimestamp(message.timestamp);

  switch (message.role) {
    case "tool_call":
      return {
        id: message.id,
        role: "tool_call",
        content: message.content,
        toolName: message.tool_name,
        toolInput: isPlainObject(message.tool_input) ? message.tool_input : undefined,
        toolError: message.tool_error || undefined,
        isStreaming: false,
        timestamp,
      };
    case "tool_result":
      return {
        id: message.id,
        role: "tool_result",
        content: message.content,
        toolName: message.tool_name,
        toolResult: mapToolResult(message.tool_result),
        toolError: message.tool_error || undefined,
        isStreaming: false,
        timestamp,
      };
    default:
      return {
        id: message.id,
        role: mapTranscriptRole(message.role),
        content: message.content,
        thinking: message.thinking || undefined,
        thinkingComplete: message.thinking_complete || undefined,
        isStreaming: false,
        timestamp,
      };
  }
}

export function mapTranscriptToMessages(messages: TranscriptMessage[]): UIMessage[] {
  return messages.map(mapTranscriptMessage);
}
