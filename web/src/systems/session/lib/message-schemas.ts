import { validateUIMessages } from "ai";
import { z } from "zod";

import type { SessionMessage } from "../types";

const aghEventDataSchema = z
  .object({
    type: z.string(),
    session_id: z.string().optional(),
    turn_id: z.string().optional(),
    request_id: z.string().optional(),
    timestamp: z.string().optional(),
    text: z.string().optional(),
    title: z.string().optional(),
    tool_call_id: z.string().optional(),
    stop_reason: z.string().optional(),
    action: z.string().optional(),
    resource: z.string().optional(),
    decision: z.string().optional(),
    error: z.string().optional(),
    usage: z
      .object({
        turn_id: z.string().optional(),
        input_tokens: z.number().optional(),
        output_tokens: z.number().optional(),
        total_tokens: z.number().optional(),
        thought_tokens: z.number().optional(),
        cache_read_tokens: z.number().optional(),
        cache_write_tokens: z.number().optional(),
        context_used: z.number().optional(),
        context_size: z.number().optional(),
        cost_amount: z.number().optional(),
        cost_currency: z.string().optional(),
        timestamp: z.string().optional(),
      })
      .optional(),
    runtime: z
      .object({
        turn_id: z.string().optional(),
        turn_source: z.string().optional(),
        turn_started_at: z.string().nullable().optional(),
        deadline_at: z.string().nullable().optional(),
        last_activity_at: z.string().nullable().optional(),
        last_activity_kind: z.string().optional(),
        last_activity_detail: z.string().optional(),
        current_tool: z.string().optional(),
        tool_call_id: z.string().optional(),
        last_progress_at: z.string().nullable().optional(),
        iteration_current: z.number().optional(),
        iteration_max: z.number().optional(),
        idle_seconds: z.number().optional(),
        elapsed_ms: z.number(),
        elapsed_seconds: z.number().optional(),
      })
      .optional(),
    raw: z.unknown().optional(),
  })
  .passthrough();

const aghPermissionDataSchema = aghEventDataSchema.extend({
  request_id: z.string(),
  raw: z.record(z.string(), z.unknown()).optional(),
});

const unknownDataSchema = z.unknown();

const knownDataSchemas: Record<string, z.ZodType<unknown>> = {
  "agh-event": aghEventDataSchema,
  "agh-permission": aghPermissionDataSchema,
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function dataPartName(part: unknown): string | null {
  if (!isRecord(part) || typeof part.type !== "string" || !part.type.startsWith("data-")) {
    return null;
  }

  const name = part.type.slice(5);
  return name === "" ? null : name;
}

function dataSchemasForMessages(messages: unknown): Record<string, z.ZodType<unknown>> {
  const dataSchemas = { ...knownDataSchemas };
  if (!Array.isArray(messages)) {
    return dataSchemas;
  }

  for (const message of messages) {
    if (!isRecord(message) || !Array.isArray(message.parts)) {
      continue;
    }

    for (const part of message.parts) {
      const name = dataPartName(part);
      if (name && dataSchemas[name] === undefined) {
        dataSchemas[name] = unknownDataSchema;
      }
    }
  }

  return dataSchemas;
}

export async function normalizeTranscriptMessages(messages: unknown): Promise<SessionMessage[]> {
  if (Array.isArray(messages) && messages.length === 0) {
    return [];
  }

  return validateUIMessages<SessionMessage>({
    messages,
    dataSchemas: dataSchemasForMessages(messages),
  });
}
