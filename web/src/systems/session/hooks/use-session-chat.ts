import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useChat } from "@ai-sdk/react";
import { DefaultChatTransport } from "ai";

import { cancelSessionPrompt } from "../adapters/session-api";
import { extractPermissionRequest } from "../lib/event-mapper";
import { mapLiveChatMessages } from "../lib/live-message-mapper";
import { sessionKeys } from "../lib/query-keys";
import { useSessionStore } from "./use-session-store";
import type { AgentEventPayload } from "../types";

type ChatStatus = "submitted" | "streaming" | "ready" | "error";

export interface UseSessionChatOptions {
  sessionId: string | null;
}

export interface UseSessionChatReturn {
  sendMessage: (text: string) => void;
  stop: () => void;
  resetLiveState: () => void;
  status: ChatStatus;
  error: Error | undefined;
  isStoppingPrompt: boolean;
}

export function useSessionChat({ sessionId }: UseSessionChatOptions): UseSessionChatReturn {
  const queryClient = useQueryClient();
  const timestampsRef = useRef(new Map<string, number>());
  const [isStoppingPrompt, setIsStoppingPrompt] = useState(false);

  const transport = useMemo(() => {
    if (!sessionId) {
      return undefined;
    }

    return new DefaultChatTransport({
      api: `/api/sessions/${sessionId}/prompt`,
    });
  }, [sessionId]);

  const resolveTimestamp = useCallback((key: string): number => {
    const existing = timestampsRef.current.get(key);
    if (existing !== undefined) {
      return existing;
    }

    const next = Date.now();
    timestampsRef.current.set(key, next);
    return next;
  }, []);

  const resetTimestamps = useCallback(() => {
    timestampsRef.current.clear();
  }, []);

  const chat = useChat({
    id: sessionId ?? undefined,
    transport,
    experimental_throttle: 16,
    onData: dataPart => {
      if ((dataPart.type as string) !== "data-agh-permission") {
        return;
      }

      const permission = extractPermissionRequest(dataPart.data as AgentEventPayload);
      if (permission) {
        useSessionStore.getState().setPendingPermission(permission);
      }
    },
    onError: error => {
      console.error("[session-chat] Stream error:", error);
      const store = useSessionStore.getState();
      store.setStreaming(false);
      store.setAwaitingTranscriptSync(false);
    },
    onFinish: () => {
      if (!sessionId) {
        return;
      }

      const store = useSessionStore.getState();
      store.setStreaming(false);
      store.setAwaitingTranscriptSync(true);

      void queryClient.invalidateQueries({ queryKey: sessionKeys.detail(sessionId) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.history(sessionId) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.transcript(sessionId) });
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });

  useEffect(() => {
    if (!sessionId) {
      useSessionStore.getState().clearLiveMessages();
      useSessionStore.getState().setStreaming(false);
      return;
    }

    const liveMessages = mapLiveChatMessages(chat.messages, resolveTimestamp);
    const isStreaming = chat.status === "submitted" || chat.status === "streaming";
    const store = useSessionStore.getState();
    store.setLiveMessages(liveMessages);
    store.setStreaming(isStreaming);
  }, [chat.messages, chat.status, resolveTimestamp, sessionId]);

  useEffect(() => {
    if (!isStoppingPrompt) {
      return;
    }

    if (chat.status !== "submitted" && chat.status !== "streaming") {
      setIsStoppingPrompt(false);
    }
  }, [chat.status, isStoppingPrompt]);

  useEffect(() => {
    resetTimestamps();
    setIsStoppingPrompt(false);
  }, [resetTimestamps, sessionId]);

  const resetLiveState = useCallback(() => {
    resetTimestamps();
    chat.setMessages([]);

    const store = useSessionStore.getState();
    store.clearLiveMessages();
    store.setStreaming(false);
    store.setAwaitingTranscriptSync(false);
    store.setPendingPermission(null);
  }, [chat.setMessages, resetTimestamps]);

  const sendMessage = useCallback(
    (text: string) => {
      if (!sessionId) {
        return;
      }

      const store = useSessionStore.getState();
      store.setAwaitingTranscriptSync(false);
      store.setPendingPermission(null);
      store.setStreaming(true);
      chat.sendMessage({ text });
    },
    [chat.sendMessage, sessionId]
  );

  const stop = useCallback(() => {
    if (!sessionId || isStoppingPrompt) {
      return;
    }

    setIsStoppingPrompt(true);

    void cancelSessionPrompt(sessionId).catch(error => {
      setIsStoppingPrompt(false);
      console.error("[session-chat] Cancel prompt failed:", error);
    });
  }, [isStoppingPrompt, sessionId]);

  return {
    sendMessage,
    stop,
    resetLiveState,
    status: chat.status,
    error: chat.error,
    isStoppingPrompt,
  };
}
