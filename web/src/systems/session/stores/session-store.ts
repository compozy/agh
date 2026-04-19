import type { StateCreator } from "zustand";
import type { PermissionRequest, UIMessage } from "../types";

export interface ComposerDraft {
  text: string;
  skillId?: string;
  channel?: string;
}

export interface SessionState {
  activeSessionId: string | null;
  messages: UIMessage[];
  isStreaming: boolean;
  pendingPermission: PermissionRequest | null;
  drafts: Record<string, ComposerDraft>;
}

export interface SessionActions {
  setActiveSession: (id: string, messages: UIMessage[]) => void;
  appendMessage: (msg: UIMessage) => void;
  updateLastMessage: (partial: Partial<UIMessage>) => void;
  setPendingPermission: (req: PermissionRequest | null) => void;
  setDraft: (sessionId: string, patch: Partial<ComposerDraft>) => void;
  clearDraft: (sessionId: string) => void;
  clearSession: () => void;
}

export type SessionStore = SessionState & SessionActions;

export const initialSessionState: SessionState = {
  activeSessionId: null,
  messages: [],
  isStreaming: false,
  pendingPermission: null,
  drafts: {},
};

export const createSessionStore: StateCreator<SessionStore> = set => ({
  ...initialSessionState,

  setActiveSession: (id, messages) =>
    set(state => ({
      activeSessionId: id,
      messages,
      isStreaming: false,
      pendingPermission: null,
      // Drafts survive session switches so unsent text persists across route navigations.
      drafts: state.drafts,
    })),

  appendMessage: msg => set(state => ({ messages: [...state.messages, msg] })),

  updateLastMessage: partial =>
    set(state => {
      const lastMessage = state.messages.at(-1);
      if (!lastMessage) {
        return state;
      }

      return {
        messages: [...state.messages.slice(0, -1), { ...lastMessage, ...partial }],
      };
    }),

  setPendingPermission: pendingPermission => set({ pendingPermission }),

  setDraft: (sessionId, patch) =>
    set(state => {
      const current = state.drafts[sessionId] ?? { text: "" };
      const next: ComposerDraft = { ...current, ...patch };
      const isEmpty = !next.text && !next.skillId && !next.channel;
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
