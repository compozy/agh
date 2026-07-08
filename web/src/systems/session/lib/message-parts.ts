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

import type { ToolCallStatus } from "@agh/ui";

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

/**
 * Normalize a raw tool result payload into a `ToolUseResult`. Consolidates the
 * shape handling both render paths need (assistant-ui toolkit + derive layer):
 * AGH event payloads fold through `parseToolUseResult`, strings become `content`,
 * and any other record is preserved as `rawOutput`. Absent output stays absent.
 */
export function resolveToolResult(result: unknown): ToolUseResult | undefined {
  if (result === undefined || result === null) return undefined;
  if (isAgentEventPayload(result)) return parseToolUseResult(result);
  if (typeof result === "string") return { content: result };
  if (isRecord(result)) return { rawOutput: result };
  return { content: String(result) };
}

function nonEmptyString(value: unknown): boolean {
  return typeof value === "string" && value.trim().length > 0;
}

/** A tool call carries meaningful input once its args object has at least one key. */
export function hasToolInput(input: Record<string, unknown> | undefined): boolean {
  return input !== undefined && Object.keys(input).length > 0;
}

/** The tool result payload reports its own error field (error-shaped output). */
function toolResultHasError(result: ToolUseResult): boolean {
  return nonEmptyString(result.error);
}

/**
 * True when a resolved tool produced no substantive output. Only explicit output
 * channels count; `rawOutput` alone (a catch-all echo of the raw payload) does not
 * promote an otherwise-empty result to `success`. An empty result is neutral and
 * stays `empty` (Minus) until the turn settles.
 */
export function toolResultIsEmpty(result: ToolUseResult | undefined): boolean {
  if (!result) return true;
  if (nonEmptyString(result.content)) return false;
  if (nonEmptyString(result.stdout)) return false;
  if (nonEmptyString(result.stderr)) return false;
  if (nonEmptyString(result.filePath)) return false;
  if (Array.isArray(result.structuredPatch) && result.structuredPatch.length > 0) return false;
  return true;
}

// Last-resort failure flag: some providers report a clean lifecycle while the
// failure text lands on the error channel. Scoped to `stderr` + exit-code shapes
// only — never file `content` or `stdout` — to avoid the locale/substring false
// positives called out in analysis/02 §5. Explicit `toolError` / `result.error`
// are always preferred over this heuristic.
const STDERR_FAILURE_PATTERNS: readonly RegExp[] = [
  /command not found/i,
  /no such file or directory/i,
  /\bENOENT\b/,
  /exit(?:ed)? with (?:exit )?code\s+[1-9]\d*/i,
  /exit status\s+[1-9]\d*/i,
];

function toolOutputLooksLikeFailure(result: ToolUseResult): boolean {
  if (!nonEmptyString(result.stderr)) return false;
  const stderr = result.stderr as string;
  return STDERR_FAILURE_PATTERNS.some(pattern => pattern.test(stderr));
}

export interface ToolRowStatusInput {
  /** Explicit AGH runtime error (assistant-ui `output-error`). */
  toolError?: boolean;
  /** Normalized tool output; absent while the call is still in flight. */
  toolResult?: ToolUseResult;
  /** Whether the call already carries input args. */
  hasInput: boolean;
  /** True once the owning turn has settled — promotes neutral rows to success. */
  turnSettled: boolean;
}

export interface ToolRowStatus {
  status: ToolCallStatus;
  /** Only a true runtime error tones the heading danger. */
  runtimeError: boolean;
}

/**
 * Single source of truth for a tool row's state. Prefers explicit AGH signals
 * (`toolError`, then the result's own `error` field) and only falls back to the
 * stderr failure heuristic. Neutral (empty-output) rows show `empty` (Minus) while
 * the turn streams and promote to `success` (Check) once it settles — never before,
 * so no premature green appears mid-stream.
 */
export function deriveToolRowStatus(input: ToolRowStatusInput): ToolRowStatus {
  const { toolError, toolResult, hasInput, turnSettled } = input;
  if (toolError) {
    return { status: "failed", runtimeError: true };
  }
  if (toolResult === undefined) {
    return { status: hasInput ? "running" : "pending", runtimeError: false };
  }
  if (toolResultHasError(toolResult) || toolOutputLooksLikeFailure(toolResult)) {
    return { status: "failed", runtimeError: false };
  }
  if (!toolResultIsEmpty(toolResult)) {
    return { status: "success", runtimeError: false };
  }
  return { status: turnSettled ? "success" : "empty", runtimeError: false };
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
