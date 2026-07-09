import type { ThreadMessage } from "@assistant-ui/react";

type MessageSource = "transcript" | "runtime";

const TURN_ID_KEYS = ["turn_id", "turnId"] as const;
const CLIENT_TEMP_ID_KEYS = [
  "client_temp_id",
  "clientTempId",
  "client_message_id",
  "clientMessageId",
  "temp_id",
  "tempId",
] as const;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringField(record: Record<string, unknown> | undefined, keys: readonly string[]) {
  if (!record) {
    return null;
  }
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.trim().length > 0) {
      return value;
    }
  }
  return null;
}

function messageCustom(message: ThreadMessage): Record<string, unknown> | undefined {
  const custom = message.metadata?.custom;
  return isRecord(custom) ? custom : undefined;
}

function dataRecords(message: ThreadMessage): Record<string, unknown>[] {
  const records: Record<string, unknown>[] = [];
  for (const part of message.content) {
    if (!isRecord(part)) {
      continue;
    }
    if ("data" in part && isRecord(part.data)) {
      records.push(part.data);
    }
  }
  return records;
}

function promotionKeys(message: ThreadMessage, source: MessageSource): string[] {
  const keys = new Set<string>();
  const candidates = [messageCustom(message), ...dataRecords(message)];

  for (const candidate of candidates) {
    const turnId = stringField(candidate, TURN_ID_KEYS);
    if (!turnId) {
      continue;
    }
    const clientTempId = stringField(candidate, CLIENT_TEMP_ID_KEYS);
    if (clientTempId) {
      keys.add(`${turnId}:${clientTempId}`);
    }
    if (source === "runtime") {
      keys.add(`${turnId}:${message.id}`);
    }
  }

  return [...keys];
}

export function mergeSessionThreadReadModel({
  transcriptMessages,
  runtimeMessages,
  includeRuntimeTail = true,
}: {
  transcriptMessages: readonly ThreadMessage[];
  runtimeMessages: readonly ThreadMessage[];
  includeRuntimeTail?: boolean;
}): readonly ThreadMessage[] {
  if (runtimeMessages.length === 0 || !includeRuntimeTail) {
    return transcriptMessages;
  }

  const transcriptIds = new Set(transcriptMessages.map(message => message.id));
  const transcriptPromotionKeys = new Set(
    transcriptMessages.flatMap(message => promotionKeys(message, "transcript"))
  );
  const optimisticTail: ThreadMessage[] = [];

  for (const message of runtimeMessages) {
    if (transcriptIds.has(message.id)) {
      continue;
    }
    const hasPromotedServerEcho = promotionKeys(message, "runtime").some(key =>
      transcriptPromotionKeys.has(key)
    );
    if (hasPromotedServerEcho) {
      continue;
    }
    optimisticTail.push(message);
  }

  return optimisticTail.length === 0
    ? transcriptMessages
    : [...transcriptMessages, ...optimisticTail];
}
