import {
  getToolName,
  isDataUIPart,
  isReasoningUIPart,
  isTextUIPart,
  isToolUIPart,
  type DataUIPart,
  type DynamicToolUIPart,
  type ToolUIPart,
} from "ai";

import type {
  AghPermissionData,
  AgentEventPayload,
  PermissionDecision,
  PermissionRequest,
  SessionMessage,
  ToolUseResult,
} from "../types";

export type SessionToolPart = ToolUIPart | DynamicToolUIPart;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function getMessageText(message: SessionMessage): string {
  return message.parts
    .filter(isTextUIPart)
    .map(part => part.text)
    .join("");
}

export function getMessageReasoning(message: SessionMessage): string {
  return message.parts
    .filter(isReasoningUIPart)
    .map(part => part.text)
    .join("");
}

export function getMessageToolParts(message: SessionMessage): SessionToolPart[] {
  return message.parts.filter(isToolUIPart);
}

export function getMessagePermissionParts(
  message: SessionMessage
): DataUIPart<{ "agh-permission": AghPermissionData }>[] {
  return message.parts.filter(
    (part): part is DataUIPart<{ "agh-permission": AghPermissionData }> =>
      isDataUIPart(part) && part.type === "data-agh-permission"
  );
}

export function getToolPartName(part: SessionToolPart): string {
  return getToolName(part);
}

export function getToolPartInput(part: SessionToolPart): Record<string, unknown> {
  return isRecord(part.input) ? part.input : {};
}

export function getToolPartResult(part: SessionToolPart): ToolUseResult | null {
  if (part.state !== "output-available" && part.state !== "output-error") {
    return null;
  }

  const payload = part.state === "output-available" ? part.output : undefined;
  const output = isAgentEventPayload(payload) ? payload : undefined;

  if (!output) {
    if (part.state === "output-error") {
      return {
        error: part.errorText,
      };
    }
    return isRecord(payload)
      ? {
          rawOutput: payload,
        }
      : null;
  }

  return parseToolUseResult(output);
}

export function isPermissionRequestData(value: unknown): value is AghPermissionData {
  return isRecord(value) && typeof value.request_id === "string";
}

function normalizePermissionDecision(value: unknown): PermissionDecision | null {
  switch (typeof value === "string" ? value.trim() : "") {
    case "allow-once":
      return "allow-once";
    case "allow-always":
      return "allow-always";
    case "reject-once":
      return "reject-once";
    case "reject-always":
      return "reject-always";
    default:
      return null;
  }
}

function permissionSupportedDecisions(raw: Record<string, unknown> | undefined) {
  const options = Array.isArray(raw?.options) ? raw.options : [];
  const decisions: PermissionDecision[] = [];
  for (const option of options) {
    if (!isRecord(option)) continue;
    const decision = normalizePermissionDecision(option.decision ?? option.option_id);
    if (decision != null && !decisions.includes(decision)) {
      decisions.push(decision);
    }
  }
  return decisions.length > 0 ? decisions : undefined;
}

function permissionToolInput(raw: Record<string, unknown> | undefined): Record<string, unknown> {
  return isRecord(raw?.tool_input) ? raw.tool_input : {};
}

export function toPermissionRequest(data: AghPermissionData): PermissionRequest {
  const raw = data.raw;
  return {
    requestId: data.request_id,
    toolName: data.title ?? "unknown",
    toolInput: permissionToolInput(raw),
    action: data.action ?? "",
    resource: data.resource ?? "",
    supportedDecisions: permissionSupportedDecisions(raw),
    turnId: data.turn_id,
    toolCallId: data.tool_call_id,
  };
}

export function isAgentEventPayload(value: unknown): value is AgentEventPayload {
  return isRecord(value) && typeof value.type === "string";
}

export function parseToolUseResult(event: AgentEventPayload): ToolUseResult {
  if (isRecord(event.raw)) {
    return {
      stdout: typeof event.raw.stdout === "string" ? event.raw.stdout : undefined,
      stderr: typeof event.raw.stderr === "string" ? event.raw.stderr : undefined,
      filePath:
        typeof event.raw.filePath === "string"
          ? event.raw.filePath
          : typeof event.raw.file_path === "string"
            ? event.raw.file_path
            : undefined,
      content: typeof event.raw.content === "string" ? event.raw.content : undefined,
      structuredPatch: Array.isArray(event.raw.structuredPatch)
        ? event.raw.structuredPatch
        : Array.isArray(event.raw.structured_patch)
          ? event.raw.structured_patch
          : undefined,
      error: typeof event.raw.error === "string" ? event.raw.error : event.error,
      rawOutput: event.raw,
    };
  }

  return {
    content: event.text,
    error: event.error,
  };
}
