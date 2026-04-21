import type { UIMessage as AIUIMessage } from "ai";

import type { ToolUseResult, UIMessage } from "../types";

type TimestampResolver = (key: string) => number;

interface AssistantSegment {
  content: string;
  thinking: string;
  thinkingComplete: boolean;
  isStreaming: boolean;
}

type ToolPart = Extract<AIUIMessage["parts"][number], { toolCallId: string }>;

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isToolPart(part: AIUIMessage["parts"][number]): part is ToolPart {
  return "toolCallId" in part;
}

function extractTextContent(message: AIUIMessage): string {
  let content = "";
  for (const part of message.parts) {
    if (part.type === "text") {
      content += part.text;
    }
  }
  return content;
}

function toolNameFromPart(part: ToolPart): string {
  if (part.type === "dynamic-tool") {
    return part.toolName ?? "tool";
  }
  return part.type.replace("tool-", "");
}

function normalizeToolInput(input: unknown): Record<string, unknown> | undefined {
  if (input === undefined) {
    return undefined;
  }
  if (isPlainObject(input)) {
    return input;
  }
  return { value: input };
}

function normalizeToolResult(output: unknown, errorText?: string): ToolUseResult | undefined {
  if (typeof errorText === "string" && errorText.length > 0) {
    return { error: errorText, rawOutput: output };
  }
  if (output === undefined) {
    return undefined;
  }
  if (typeof output === "string") {
    return { content: output, rawOutput: output };
  }
  if (typeof output === "number" || typeof output === "boolean") {
    return { content: String(output), rawOutput: output };
  }
  if (isPlainObject(output)) {
    return {
      stdout: typeof output.stdout === "string" ? output.stdout : undefined,
      stderr: typeof output.stderr === "string" ? output.stderr : undefined,
      filePath:
        typeof output.filePath === "string"
          ? output.filePath
          : typeof output.file_path === "string"
            ? output.file_path
            : undefined,
      content: typeof output.content === "string" ? output.content : undefined,
      structuredPatch: Array.isArray(output.structuredPatch)
        ? output.structuredPatch
        : Array.isArray(output.structured_patch)
          ? output.structured_patch
          : undefined,
      error: typeof output.error === "string" ? output.error : undefined,
      rawOutput: output,
    };
  }
  return { rawOutput: output };
}

function pushAssistantSegment(
  rows: UIMessage[],
  segment: AssistantSegment,
  messageId: string,
  segmentIndex: number,
  resolveTimestamp: TimestampResolver
) {
  if (!segment.content && !segment.thinking) {
    return;
  }

  rows.push({
    id: `${messageId}:assistant:${segmentIndex}`,
    role: "assistant",
    content: segment.content,
    thinking: segment.thinking || undefined,
    thinkingComplete: segment.thinking ? segment.thinkingComplete || undefined : undefined,
    isStreaming: segment.isStreaming || undefined,
    timestamp: resolveTimestamp(`${messageId}:assistant:${segmentIndex}`),
  });
}

function pushToolRows(rows: UIMessage[], part: ToolPart, resolveTimestamp: TimestampResolver) {
  const toolName = toolNameFromPart(part);
  const toolInput = normalizeToolInput(part.input);

  rows.push({
    id: part.toolCallId,
    role: "tool_call",
    content: "",
    toolName,
    toolInput,
    isStreaming:
      part.state === "input-streaming" ||
      part.state === "input-available" ||
      part.state === "approval-requested" ||
      part.state === "approval-responded",
    timestamp: resolveTimestamp(`${part.toolCallId}:call`),
  });

  if (
    part.state === "output-available" ||
    part.state === "output-error" ||
    part.state === "output-denied"
  ) {
    rows.push({
      id: part.toolCallId,
      role: "tool_result",
      content: typeof part.errorText === "string" ? part.errorText : "",
      toolName,
      toolResult: normalizeToolResult(part.output, part.errorText),
      toolError: part.state !== "output-available",
      timestamp: resolveTimestamp(`${part.toolCallId}:result`),
    });
  }
}

function materializeAssistantMessage(
  message: AIUIMessage,
  resolveTimestamp: TimestampResolver
): UIMessage[] {
  const rows: UIMessage[] = [];
  let segmentIndex = 0;
  let segment: AssistantSegment = {
    content: "",
    thinking: "",
    thinkingComplete: false,
    isStreaming: false,
  };

  for (const part of message.parts) {
    if (part.type === "text") {
      segment.content += part.text;
      segment.isStreaming ||= part.state === "streaming";
      continue;
    }

    if (part.type === "reasoning") {
      segment.thinking += part.text;
      segment.thinkingComplete ||= part.state === "done";
      segment.isStreaming ||= part.state === "streaming";
      continue;
    }

    if (part.type === "step-start") {
      pushAssistantSegment(rows, segment, message.id, segmentIndex, resolveTimestamp);
      segmentIndex += 1;
      segment = {
        content: "",
        thinking: "",
        thinkingComplete: false,
        isStreaming: false,
      };
      continue;
    }

    if (isToolPart(part)) {
      pushAssistantSegment(rows, segment, message.id, segmentIndex, resolveTimestamp);
      segmentIndex += 1;
      segment = {
        content: "",
        thinking: "",
        thinkingComplete: false,
        isStreaming: false,
      };
      pushToolRows(rows, part, resolveTimestamp);
    }
  }

  pushAssistantSegment(rows, segment, message.id, segmentIndex, resolveTimestamp);
  return rows;
}

export function mapLiveChatMessages(
  messages: AIUIMessage[],
  resolveTimestamp: TimestampResolver
): UIMessage[] {
  const rows: UIMessage[] = [];

  for (const message of messages) {
    if (message.role === "user") {
      rows.push({
        id: message.id,
        role: "user",
        content: extractTextContent(message),
        timestamp: resolveTimestamp(`${message.id}:user`),
      });
      continue;
    }

    if (message.role === "assistant") {
      rows.push(...materializeAssistantMessage(message, resolveTimestamp));
    }
  }

  return rows;
}
