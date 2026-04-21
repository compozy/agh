import type { StateCreator } from "zustand";

export interface ComposerDraft {
  text: string;
  channel?: string;
}

export interface SessionState {
  drafts: Record<string, ComposerDraft>;
}

export interface SessionActions {
  setDraft: (sessionId: string, patch: Partial<ComposerDraft>) => void;
  clearDraft: (sessionId: string) => void;
  clearAllDrafts: () => void;
}

export type SessionStore = SessionState & SessionActions;

export const initialSessionState: SessionState = {
  drafts: {},
};

export const createSessionStore: StateCreator<SessionStore> = set => ({
  ...initialSessionState,

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

  clearAllDrafts: () => set({ drafts: {} }),
});
