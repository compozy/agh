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

type SessionMessagePart = NonNullable<SessionMessage["parts"]>[number];

interface PartTurnMeta {
  turnId?: string;
  timestamp?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringField(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

// `validateUIMessages` parses every part against the AI SDK's schema, which drops
// unknown sibling keys — so AGH's custom `turn_id`/`timestamp` fields never survive
// validation. The turn-fold derivation needs them (turn boundaries + "Worked for Xs"
// duration), so capture them before validation and re-attach afterward. Validation
// preserves part order/count (an invalid part fails the whole message and throws),
// so index alignment is sound; the length guard is purely defensive.
function capturePartTurnMeta(messages: unknown): PartTurnMeta[][] {
  if (!Array.isArray(messages)) {
    return [];
  }
  return messages.map(message => {
    if (!isRecord(message) || !Array.isArray(message.parts)) {
      return [];
    }
    return message.parts.map((part): PartTurnMeta => {
      if (!isRecord(part)) {
        return {};
      }
      const meta: PartTurnMeta = {};
      const turnId = stringField(part, "turnId") ?? stringField(part, "turn_id");
      if (turnId) {
        meta.turnId = turnId;
      }
      const timestamp = stringField(part, "timestamp");
      if (timestamp) {
        meta.timestamp = timestamp;
      }
      return meta;
    });
  });
}

function reattachPartTurnMeta(
  validated: SessionMessage[],
  captured: PartTurnMeta[][]
): SessionMessage[] {
  return validated.map((message, index) => {
    const parts = message.parts;
    const metas = captured[index];
    if (!metas || !Array.isArray(parts) || parts.length !== metas.length) {
      return message;
    }
    let mutated = false;
    const nextParts = parts.map((part, partIndex) => {
      const meta = metas[partIndex];
      if (!meta || (!meta.turnId && !meta.timestamp)) {
        return part;
      }
      mutated = true;
      return {
        ...part,
        ...(meta.turnId ? { turnId: meta.turnId } : {}),
        ...(meta.timestamp ? { timestamp: meta.timestamp } : {}),
      } as SessionMessagePart;
    });
    return mutated ? ({ ...message, parts: nextParts } as SessionMessage) : message;
  });
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

  const capturedTurnMeta = capturePartTurnMeta(messages);
  const validated = await validateUIMessages<SessionMessage>({
    messages,
    dataSchemas: dataSchemasForMessages(messages),
  });
  return reattachPartTurnMeta(validated, capturedTurnMeta);
}
