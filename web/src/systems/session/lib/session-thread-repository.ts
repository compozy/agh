import {
  ExportedMessageRepository,
  type ThreadMessage,
  type ThreadMessageLike,
} from "@assistant-ui/react";

import type { SessionMessage } from "../types";

type SessionMessagePart = NonNullable<SessionMessage["parts"]>[number];
type ThreadContentPart = Exclude<ThreadMessageLike["content"], string>[number];
type SessionMessageWithStatus = SessionMessage & { status?: ThreadMessageLike["status"] };
type ExportedThreadMessageItem = { message: ThreadMessage };
type JSONValue = null | string | number | boolean | readonly JSONValue[] | JSONObject;
type JSONObject = { readonly [key: string]: JSONValue };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringField(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" ? value : undefined;
}

function jsonText(value: unknown): string {
  if (value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function isJSONValue(value: unknown, depth: number = 0): value is JSONValue {
  if (depth > 100) {
    return false;
  }

  if (value === null || typeof value === "string" || typeof value === "boolean") {
    return true;
  }

  if (typeof value === "number") {
    return Number.isFinite(value);
  }

  if (Array.isArray(value)) {
    return value.every(item => isJSONValue(item, depth + 1));
  }

  if (isRecord(value)) {
    return Object.values(value).every(item => isJSONValue(item, depth + 1));
  }

  return false;
}

function toJSONObject(value: unknown): JSONObject {
  if (!isRecord(value)) {
    return {};
  }
  return isJSONValue(value) ? value : {};
}

function toToolPart(record: Record<string, unknown>, type: string): ThreadContentPart {
  const toolName = type.slice("tool-".length).trim() || stringField(record, "toolName") || "tool";
  const toolCallId =
    stringField(record, "toolCallId") || stringField(record, "tool_call_id") || `${toolName}-call`;
  const input = record.input;
  const state = stringField(record, "state");

  return {
    type: "tool-call" as const,
    toolCallId,
    toolName,
    args: toJSONObject(input),
    argsText: jsonText(input),
    result: record.output,
    isError: state === "output-error" || Boolean(record.isError),
  };
}

function toThreadPart(part: SessionMessagePart): ThreadContentPart | null {
  if (!isRecord(part)) {
    return null;
  }

  const type = stringField(part, "type");
  if (!type) {
    return null;
  }

  if (type === "text") {
    return { type: "text" as const, text: stringField(part, "text") ?? "" };
  }

  if (type === "reasoning") {
    return { type: "reasoning" as const, text: stringField(part, "text") ?? "" };
  }

  if (type.startsWith("data-")) {
    return { type: type as `data-${string}`, data: (part as { data?: unknown }).data };
  }

  if (type.startsWith("tool-")) {
    return toToolPart(part, type);
  }

  return null;
}

function toThreadRole(role: SessionMessage["role"]): ThreadMessageLike["role"] {
  if (role === "user" || role === "assistant" || role === "system") {
    return role;
  }
  return "assistant";
}

export function toThreadMessageLikes(messages: SessionMessage[]): ThreadMessageLike[] {
  return messages.map(message => {
    const parts = message.parts?.map(toThreadPart).filter(part => part !== null) ?? [];
    const role = toThreadRole(message.role);
    const status = role === "assistant" ? (message as SessionMessageWithStatus).status : undefined;
    return {
      id: message.id,
      role,
      content: parts,
      status,
    } satisfies ThreadMessageLike;
  });
}

export function toReadonlyThreadMessages(messages: SessionMessage[]): ThreadMessage[] {
  const repository: { messages: ExportedThreadMessageItem[] } = ExportedMessageRepository.fromArray(
    toThreadMessageLikes(messages)
  );
  return repository.messages.map(item => item.message);
}

export function transcriptSignature(messages: SessionMessage[]): string {
  return JSON.stringify(
    messages.map(message => ({
      id: message.id,
      role: message.role,
      parts: message.parts ?? [],
      status: (message as SessionMessageWithStatus).status ?? null,
    }))
  );
}
