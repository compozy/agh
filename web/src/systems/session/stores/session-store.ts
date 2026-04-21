import type { StateCreator } from "zustand";

import type { PermissionRequest, UIMessage } from "../types";

export interface ComposerDraft {
  text: string;
  channel?: string;
}

export interface SessionState {
  activeSessionId: string | null;
  historyMessages: UIMessage[];
  liveMessages: UIMessage[];
  isStreaming: boolean;
  awaitingTranscriptSync: boolean;
  pendingPermission: PermissionRequest | null;
  drafts: Record<string, ComposerDraft>;
}

export interface SessionActions {
  setActiveSession: (id: string, messages: UIMessage[]) => void;
  replaceHistoryMessages: (messages: UIMessage[]) => void;
  setLiveMessages: (messages: UIMessage[]) => void;
  clearLiveMessages: () => void;
  resetConversation: (messages?: UIMessage[]) => void;
  setAwaitingTranscriptSync: (value: boolean) => void;
  setStreaming: (value: boolean) => void;
  setPendingPermission: (req: PermissionRequest | null) => void;
  setDraft: (sessionId: string, patch: Partial<ComposerDraft>) => void;
  clearDraft: (sessionId: string) => void;
  clearSession: () => void;
}

export type SessionStore = SessionState & SessionActions;

export const initialSessionState: SessionState = {
  activeSessionId: null,
  historyMessages: [],
  liveMessages: [],
  isStreaming: false,
  awaitingTranscriptSync: false,
  pendingPermission: null,
  drafts: {},
};

export const createSessionStore: StateCreator<SessionStore> = set => ({
  ...initialSessionState,

  setActiveSession: (id, messages) =>
    set(state => ({
      activeSessionId: id,
      historyMessages: messages,
      liveMessages: [],
      isStreaming: false,
      awaitingTranscriptSync: false,
      pendingPermission: null,
      drafts: state.drafts,
    })),

  replaceHistoryMessages: historyMessages => set({ historyMessages }),

  setLiveMessages: liveMessages => set({ liveMessages }),

  clearLiveMessages: () => set({ liveMessages: [] }),

  resetConversation: (messages = []) =>
    set(state => ({
      activeSessionId: state.activeSessionId,
      historyMessages: messages,
      liveMessages: [],
      isStreaming: false,
      awaitingTranscriptSync: false,
      pendingPermission: null,
      drafts: state.drafts,
    })),

  setAwaitingTranscriptSync: awaitingTranscriptSync => set({ awaitingTranscriptSync }),

  setStreaming: isStreaming => set({ isStreaming }),

  setPendingPermission: pendingPermission => set({ pendingPermission }),

  setDraft: (sessionId, patch) =>
    set(state => {
      const current = state.drafts[sessionId] ?? { text: "" };
      const next: ComposerDraft = { ...current, ...patch };
      const isEmpty = !next.text && !next.channel;
      if (isEmpty) {
        if (!(sessionId in state.drafts)) {
          return state;
        }
        const { [sessionId]: _removed, ...rest } = state.drafts;
        return { drafts: rest };
      }
      return { drafts: { ...state.drafts, [sessionId]: next } };
    }),

  clearDraft: sessionId =>
    set(state => {
      if (!(sessionId in state.drafts)) {
        return state;
      }
      const { [sessionId]: _removed, ...rest } = state.drafts;
      return { drafts: rest };
    }),

  clearSession: () => set(initialSessionState),
});
