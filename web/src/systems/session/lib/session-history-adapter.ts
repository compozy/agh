import type { QueryClient } from "@tanstack/react-query";
import type {
  GenericThreadHistoryAdapter,
  MessageFormatAdapter,
  MessageFormatItem,
  MessageFormatRepository,
  ThreadHistoryAdapter,
} from "@assistant-ui/react";

import type { SessionMessage } from "../types";
import { sessionTranscriptOptions } from "./query-options";

type AISDKStorageFormat = Omit<SessionMessage, "id">;

const aiSDKV6FormatAdapter: MessageFormatAdapter<SessionMessage, AISDKStorageFormat> = {
  format: "ai-sdk/v6",
  encode({ message: { id: _id, ...message } }) {
    return message;
  },
  decode(stored) {
    return {
      parentId: stored.parent_id,
      message: {
        id: stored.id,
        ...stored.content,
      },
    };
  },
  getId(message) {
    return message.id;
  },
};

function toLinearRepository(messages: SessionMessage[]): MessageFormatRepository<SessionMessage> {
  let parentId: string | null = null;

  return {
    headId: messages.at(-1)?.id ?? null,
    messages: messages.map(message => {
      const item: MessageFormatItem<SessionMessage> = {
        parentId,
        message,
      };
      parentId = message.id;
      return item;
    }),
  };
}

function createAISDKV6HistoryAdapter(
  sessionId: string,
  queryClient: QueryClient
): GenericThreadHistoryAdapter<SessionMessage> {
  const adapter = {
    async load() {
      const messages = await queryClient.ensureQueryData(sessionTranscriptOptions(sessionId));
      return toLinearRepository(messages);
    },
    async append() {},
  };
  return adapter as GenericThreadHistoryAdapter<SessionMessage>;
}

export function createSessionHistoryAdapter(
  sessionId: string,
  queryClient: QueryClient
): ThreadHistoryAdapter {
  const aiSDKV6History = createAISDKV6HistoryAdapter(sessionId, queryClient);

  const adapter = {
    async load() {
      return { messages: [] };
    },
    async append() {},
    withFormat(
      formatAdapter: MessageFormatAdapter<SessionMessage, AISDKStorageFormat>
    ): GenericThreadHistoryAdapter<SessionMessage> {
      if (formatAdapter.format !== aiSDKV6FormatAdapter.format) {
        throw new Error(`Unsupported thread history format: ${formatAdapter.format}`);
      }

      // AGH serves persisted transcript history as AI SDK v6 messages only.
      // assistant-ui exposes this hook through a wider generic surface, but the
      // daemon persists exactly one thread-history format.
      return aiSDKV6History;
    },
  };
  return adapter as unknown as ThreadHistoryAdapter;
}
